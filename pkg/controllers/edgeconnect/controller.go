package edgeconnect

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	edgeconnectClient "github.com/Dynatrace/dynatrace-operator/pkg/clients/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/config"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/deployment"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	k8sdeployment "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/deployment"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	fastUpdateInterval    = 1 * time.Minute
	defaultUpdateInterval = 30 * time.Minute

	finalizerName = "server"
)

var (
	ErrTokenNotFound                = errors.New("token not found")
	ErrUnsupportedConfigFileVersion = errors.New("unsupported config file version")
)

type oauthCredentialsType struct {
	clientId     string
	clientSecret string
}

type edgeConnectClientBuilderType func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error)

// Controller reconciles an EdgeConnect object
type Controller struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the api-server
	client                   client.Client
	apiReader                client.Reader
	registryClientBuilder    registry.ClientBuilder
	config                   *rest.Config
	timeProvider             *timeprovider.Provider
	edgeConnectClientBuilder edgeConnectClientBuilderType
}

func Add(mgr manager.Manager, _ string) error {
	return NewController(mgr).SetupWithManager(mgr)
}

func NewController(mgr manager.Manager) *Controller {
	return &Controller{
		client:                   mgr.GetClient(),
		apiReader:                mgr.GetAPIReader(),
		registryClientBuilder:    registry.NewClient,
		config:                   mgr.GetConfig(),
		timeProvider:             timeprovider.New(),
		edgeConnectClientBuilder: newEdgeConnectClient(),
	}
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgeconnect.EdgeConnect{}).
		Named("edgeconnect-controller").
		Owns(&appsv1.Deployment{}).
		Complete(controller)
}

func (controller *Controller) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	_log := log.WithValues("namespace", request.Namespace, "name", request.Name)

	_log.Info("reconciling EdgeConnect")

	ec, err := controller.getEdgeConnect(ctx, request.Name, request.Namespace)
	if err != nil {
		_log.Debug("reconciliation of EdgeConnect failed")

		return reconcile.Result{}, err
	} else if ec == nil {
		_log.Debug("EdgeConnect object does not exist")

		return reconcile.Result{}, nil
	}

	if deletionTimestamp := ec.GetDeletionTimestamp(); deletionTimestamp != nil {
		_log.Debug("EdgeConnect object shall be deleted", "timestamp", deletionTimestamp.String())

		return reconcile.Result{}, controller.reconcileEdgeConnectDeletion(ctx, ec)
	}

	_log.Debug("EdgeConnect object needs reconcile")

	return controller.reconcileEdgeConnect(ctx, ec)
}

//nolint:revive
func (controller *Controller) reconcileEdgeConnectDeletion(ctx context.Context, ec *edgeconnect.EdgeConnect) error {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name, "scenario", "deletion")

	_log.Info("reconciling EdgeConnect deletion", "name", ec.Name, "namespace", ec.Namespace)

	edgeConnectIdFromSecret, err := controller.getEdgeConnectIdFromClientSecret(ctx, ec)
	if err != nil {
		return err
	}

	ec.ObjectMeta.Finalizers = nil
	if err := controller.client.Update(ctx, ec); err != nil {
		_log.Debug("updating the EdgeConnect object failed, couldn't remove the finalizers")

		return errors.WithStack(err)
	}

	edgeConnectClient, err := controller.buildEdgeConnectClient(ctx, ec)
	if err != nil {
		_log.Debug("building EdgeConnect client failed")

		return err
	}

	tenantEdgeConnect, err := getEdgeConnectByName(edgeConnectClient, ec.Name)
	if err != nil {
		_log.Debug("failed to get EdgeConnect by name")

		return err
	}

	switch {
	case tenantEdgeConnect.ID == "":
		_log.Info("EdgeConnect not found on the tenant")

		return nil
	case !tenantEdgeConnect.ManagedByDynatraceOperator:
		_log.Info("can't delete EdgeConnect configuration from the tenant because it has been created manually by a user")

		return nil
	case edgeConnectIdFromSecret == "":
		_log.Info("EdgeConnect client secret is missing")
	default:
		if tenantEdgeConnect.ID != edgeConnectIdFromSecret {
			_log.Info("EdgeConnect client secret contains invalid Id")
		}
	}

	if ec.IsK8SAutomationEnabled() && ec.IsProvisionerModeEnabled() {
		err = controller.deleteConnectionSetting(edgeConnectClient, ec)
		if err != nil {
			_log.Info("reconcile deletion: Deleting connection setting failed")

			return err
		}
	}

	return edgeConnectClient.DeleteEdgeConnect(tenantEdgeConnect.ID)
}

func (controller *Controller) deleteConnectionSetting(edgeConnectClient edgeconnectClient.Client, ec *edgeconnect.EdgeConnect) error {
	envSetting, err := GetConnectionSetting(edgeConnectClient, ec.Name, ec.Namespace, ec.Status.KubeSystemUID)
	if err != nil {
		return err
	}

	if (envSetting != edgeconnectClient.EnvironmentSetting{}) {
		err = edgeConnectClient.DeleteConnectionSetting(*envSetting.ObjectId)
		if err != nil {
			return err
		}
	}

	return nil
}

func (controller *Controller) reconcileEdgeConnect(ctx context.Context, ec *edgeconnect.EdgeConnect) (reconcile.Result, error) {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name)

	oldStatus := *ec.Status.DeepCopy()

	err := controller.reconcileEdgeConnectCR(ctx, ec)

	if err != nil {
		ec.Status.SetPhase(status.Error)
		_log.Debug("error reconciling EdgeConnect, setting phase 'Error'")
	} else {
		_log.Debug("moving EdgeConnect to correct phase")
		ec.Status.SetPhase(controller.determineEdgeConnectPhase(ec))
	}

	if isDifferentStatus, err := hasher.IsDifferent(oldStatus, ec.Status); err != nil {
		_log.Error(errors.WithStack(err), "failed to generate hash for the status section")
	} else if isDifferentStatus {
		_log.Info("status changed, updating EdgeConnect")

		if errClient := controller.updateEdgeConnectStatus(ctx, ec); errClient != nil {
			retErr := errors.WithMessagef(errClient, "failed to update EdgeConnect after failure, original error: %s", err)

			_log.Debug("reconcileEdgeConnect status update error")

			return reconcile.Result{RequeueAfter: fastUpdateInterval}, retErr
		}
	}

	_log.Info("reconciling EdgeConnect done")

	if err != nil {
		_log.Debug("reconcileEdgeConnect error")

		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: defaultUpdateInterval}, nil
}

func (controller *Controller) reconcileEdgeConnectCR(ctx context.Context, ec *edgeconnect.EdgeConnect) error {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name)

	if err := controller.updateFinalizers(ctx, ec); err != nil {
		_log.Debug("updating finalizers failed")

		return err
	}

	if err := controller.updateVersionInfo(ctx, ec); err != nil {
		_log.Debug("updating version info failed")

		return err
	}

	if ec.Status.KubeSystemUID == "" {
		_log.Debug("reconcile EdgeConnect kube-system UID")

		kubeSystemUID, err := kubesystem.GetUID(ctx, controller.apiReader)
		if err != nil {
			return err
		}

		ec.Status.KubeSystemUID = string(kubeSystemUID)
	}

	if ec.IsProvisionerModeEnabled() {
		_log.Debug("reconcile EdgeConnect provisioner")

		return controller.reconcileEdgeConnectProvisioner(ctx, ec)
	}

	_log.Debug("reconcile regular EdgeConnect")

	return controller.reconcileEdgeConnectRegular(ctx, ec)
}

func (controller *Controller) getEdgeConnect(ctx context.Context, name, namespace string) (*edgeconnect.EdgeConnect, error) {
	ec := &edgeconnect.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := controller.apiReader.Get(ctx, client.ObjectKey{Name: ec.Name, Namespace: ec.Namespace}, ec)
	if k8serrors.IsNotFound(err) {
		log.Debug("EdgeConnect object not found", "name", ec.Name, "namespace", ec.Namespace)

		return nil, nil //nolint:nilnil
	} else if err != nil {
		log.Debug("Unable to get EdgeConnect object ",
			"name", ec.Name, "namespace", ec.Namespace)

		return nil, errors.WithStack(err)
	}

	return ec, nil
}

func (controller *Controller) updateFinalizers(ctx context.Context, ec *edgeconnect.EdgeConnect) error {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name)

	if ec.IsProvisionerModeEnabled() && len(ec.ObjectMeta.Finalizers) == 0 {
		_log.Info("updating finalizers")

		ec.ObjectMeta.Finalizers = []string{finalizerName}
		if err := controller.client.Update(ctx, ec); err != nil {
			_log.Debug("updating finalizers failed")

			return errors.WithStack(err)
		}
	}

	return nil
}

func (controller *Controller) updateVersionInfo(ctx context.Context, ec *edgeconnect.EdgeConnect) error {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name)

	_log.Info("updating version info")

	transport := http.DefaultTransport.(*http.Transport).Clone()
	keyChainSecret := ec.EmptyPullSecret()

	registryClient, err := controller.registryClientBuilder(
		registry.WithContext(ctx),
		registry.WithApiReader(controller.apiReader),
		registry.WithTransport(transport),
		registry.WithKeyChainSecret(&keyChainSecret),
	)
	if err != nil {
		_log.Debug("updating finalizers failed", "secretName", keyChainSecret.Name)

		return errors.WithStack(err)
	}

	versionReconciler := version.NewReconciler(controller.apiReader, registryClient, timeprovider.New(), ec)
	if err = versionReconciler.Reconcile(ctx); err != nil {
		_log.Debug("reconciliation of EdgeConnect version failed")

		return err
	}

	_log.Debug("EdgeConnect version info updated")

	return nil
}

func (controller *Controller) updateEdgeConnectStatus(ctx context.Context, ec *edgeconnect.EdgeConnect) error {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name)

	ec.Status.UpdatedTimestamp = *controller.timeProvider.Now()

	err := controller.client.Status().Update(ctx, ec)
	if k8serrors.IsConflict(err) {
		_log.Info("could not update EdgeConnect status due to conflict")

		return errors.WithStack(err)
	} else if err != nil {
		return errors.WithStack(err)
	}

	_log.Info("EdgeConnect status updated", "timestamp", ec.Status.UpdatedTimestamp)

	return nil
}

func (controller *Controller) reconcileEdgeConnectRegular(ctx context.Context, ec *edgeconnect.EdgeConnect) error {
	desiredDeployment := deployment.New(ec)

	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name, "deploymentName", desiredDeployment.Name)

	if err := controllerutil.SetControllerReference(ec, desiredDeployment, scheme.Scheme); err != nil {
		return errors.WithStack(err)
	}

	_, secretHash, err := controller.createOrUpdateEdgeConnectConfigSecret(ctx, ec)
	if err != nil {
		return err
	}

	desiredDeployment.Spec.Template.Annotations = map[string]string{consts.EdgeConnectAnnotationSecretHash: secretHash}

	ddHash, err := hasher.GenerateHash(desiredDeployment)
	if err != nil {
		_log.Debug("Unable to generate hash for EdgeConnect deployment")

		return err
	}

	desiredDeployment.Annotations[hasher.AnnotationHash] = ddHash

	_, err = k8sdeployment.Query(controller.client, controller.apiReader, log).WithOwner(ec).CreateOrUpdate(ctx, desiredDeployment)
	if err != nil {
		_log.Info("could not create or update deployment for EdgeConnect")

		return err
	}

	return nil
}

func (controller *Controller) reconcileEdgeConnectProvisioner(ctx context.Context, ec *edgeconnect.EdgeConnect) error { //nolint:revive
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name)

	_log.Info("reconcileEdgeConnectProvisioner")

	edgeConnectClient, err := controller.buildEdgeConnectClient(ctx, ec)
	if err != nil {
		_log.Debug("unable to build EdgeConnect client")

		return err
	}

	tenantEdgeConnect, err := getEdgeConnectByName(edgeConnectClient, ec.Name)
	if err != nil {
		return err
	}

	edgeConnectIdFromSecret, err := controller.getEdgeConnectIdFromClientSecret(ctx, ec)
	if err != nil {
		return err
	}

	if tenantEdgeConnect.ID != "" && !tenantEdgeConnect.ManagedByDynatraceOperator {
		_log.Info("can't delete EdgeConnect configuration from the tenant because it has been created manually by a user", "name", tenantEdgeConnect.Name)

		return nil
	}

	if tenantEdgeConnect.ID != "" {
		if edgeConnectIdFromSecret == "" {
			_log.Info("EdgeConnect has to be recreated due to missing secret")

			if err := edgeConnectClient.DeleteEdgeConnect(tenantEdgeConnect.ID); err != nil {
				return err
			}

			tenantEdgeConnect.ID = ""
		} else if tenantEdgeConnect.ID != edgeConnectIdFromSecret {
			_log.Info("EdgeConnect has to be recreated due to invalid Id")

			if err := edgeConnectClient.DeleteEdgeConnect(tenantEdgeConnect.ID); err != nil {
				return err
			}

			tenantEdgeConnect.ID = ""
		}
	}

	if tenantEdgeConnect.ID == "" {
		err := controller.createEdgeConnect(ctx, edgeConnectClient, ec)
		if err != nil {
			return err
		}

		return controller.createOrUpdateEdgeConnectDeploymentAndSettings(ctx, ec)
	}

	err = controller.updateEdgeConnect(ctx, edgeConnectClient, ec)
	if err != nil {
		return err
	}

	return controller.createOrUpdateEdgeConnectDeploymentAndSettings(ctx, ec)
}

func (controller *Controller) buildEdgeConnectClient(ctx context.Context, ec *edgeconnect.EdgeConnect) (edgeconnectClient.Client, error) {
	oauthCredentials, err := controller.getOauthCredentials(ctx, ec)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return controller.edgeConnectClientBuilder(ctx, ec, oauthCredentials)
}

func (controller *Controller) getOauthCredentials(ctx context.Context, ec *edgeconnect.EdgeConnect) (oauthCredentialsType, error) {
	query := k8ssecret.Query(controller.client, controller.apiReader, log)

	secret, err := query.Get(ctx, types.NamespacedName{
		Name:      ec.Spec.OAuth.ClientSecret,
		Namespace: ec.Namespace,
	})
	if err != nil {
		return oauthCredentialsType{}, errors.WithStack(err)
	}

	oauthClientId, err := k8ssecret.ExtractToken(secret, consts.KeyEdgeConnectOauthClientID)
	if err != nil {
		return oauthCredentialsType{}, errors.WithStack(err)
	}

	oauthClientSecret, err := k8ssecret.ExtractToken(secret, consts.KeyEdgeConnectOauthClientSecret)
	if err != nil {
		return oauthCredentialsType{}, errors.WithStack(err)
	}

	return oauthCredentialsType{clientId: oauthClientId, clientSecret: oauthClientSecret}, nil
}

func newEdgeConnectClient() func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnectClient.Client, error) {
		edgeConnectClient, err := edgeconnectClient.NewClient(
			oauthCredentials.clientId,
			oauthCredentials.clientSecret,
			edgeconnectClient.WithBaseURL("https://"+ec.Spec.ApiServer),
			edgeconnectClient.WithTokenURL(ec.Spec.OAuth.Endpoint),
			edgeconnectClient.WithOauthScopes([]string{
				"app-engine:edge-connects:read",
				"app-engine:edge-connects:write",
				"app-engine:edge-connects:delete",
				"oauth2:clients:manage",
				"settings:objects:read",
				"settings:objects:write",
			}),
			edgeconnectClient.WithContext(ctx),
		)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		return edgeConnectClient, nil
	}
}

func getEdgeConnectByName(edgeConnectClient edgeconnectClient.Client, name string) (edgeconnectClient.GetResponse, error) {
	_log := log.WithValues("name", name)

	ecs, err := edgeConnectClient.GetEdgeConnects(name)
	if err != nil {
		log.Debug("Unable to get EdgeConnect object")

		return edgeconnectClient.GetResponse{}, errors.WithStack(err)
	}

	if len(ecs.EdgeConnects) > 1 {
		_log.Debug("Found multiple EdgeConnect objects with the same name", "count", ecs.EdgeConnects)

		return edgeconnectClient.GetResponse{}, errors.New("many EdgeConnects have the same name")
	}

	if len(ecs.EdgeConnects) == 1 {
		_log.Debug("Found one EdgeConnect objects with matching name", "count", ecs.EdgeConnects)

		return ecs.EdgeConnects[0], nil
	}

	_log.Debug("No EdgeConnect object found with matching name", "count", ecs.EdgeConnects)

	return edgeconnectClient.GetResponse{}, nil
}

func (controller *Controller) getEdgeConnectIdFromClientSecret(ctx context.Context, ec *edgeconnect.EdgeConnect) (string, error) {
	clientSecretName := ec.ClientSecretName()

	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name, "clientSecretName", clientSecretName)

	query := k8ssecret.Query(controller.client, controller.apiReader, log)

	secret, err := query.Get(ctx, types.NamespacedName{Name: clientSecretName, Namespace: ec.Namespace})
	if err != nil {
		if k8serrors.IsNotFound(errors.Cause(err)) {
			_log.Debug("EdgeConnect client secret not found")

			return "", nil
		} else {
			_log.Debug("EdgeConnect client secret query failed")

			return "", errors.WithStack(err)
		}
	}

	id, err := k8ssecret.ExtractToken(secret, consts.KeyEdgeConnectId)
	if err != nil {
		log.Debug("unable to extract EdgeConnect tokens")

		return "", errors.WithStack(err)
	}

	log.Debug("successfully read EdgeConnect id from client secret", "id", "***")

	return id, nil
}

func (controller *Controller) createEdgeConnect(ctx context.Context, edgeConnectClient edgeconnectClient.Client, ec *edgeconnect.EdgeConnect) error {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name)

	createResponse, err := edgeConnectClient.CreateEdgeConnect(edgeconnectClient.NewRequest(ec.Name, ec.HostPatterns(), ec.HostMappings(), ""))
	if err != nil {
		_log.Debug("creating EdgeConnect failed")

		return errors.WithStack(err)
	}

	_log.Debug("createResponse", "id", createResponse.ID)

	ecOAuthSecret, err := k8ssecret.Build(ec, ec.ClientSecretName(), map[string][]byte{
		consts.KeyEdgeConnectOauthClientID:     []byte(createResponse.OauthClientId),
		consts.KeyEdgeConnectOauthClientSecret: []byte(createResponse.OauthClientSecret),
		consts.KeyEdgeConnectOauthResource:     []byte(createResponse.OauthClientResource),
		consts.KeyEdgeConnectId:                []byte(createResponse.ID)})

	if err != nil {
		_log.Debug("unable to create EdgeConnect secret")

		return errors.WithStack(err)
	}

	query := k8ssecret.Query(controller.client, controller.apiReader, _log)

	_, err = query.CreateOrUpdate(ctx, ecOAuthSecret)
	if err != nil {
		_log.Debug("could not create or update secret for edge-connect client")

		return errors.WithStack(err)
	}

	_log.Debug("EdgeConnect created")

	return nil
}

func (controller *Controller) updateEdgeConnect(ctx context.Context, edgeConnectClient edgeconnectClient.Client, ec *edgeconnect.EdgeConnect) error {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name)

	secretQuery := k8ssecret.Query(controller.client, controller.apiReader, log)

	secret, err := secretQuery.Get(ctx, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace})
	if err != nil {
		_log.Debug("EdgeConnect ID token not found")

		return err
	}

	id, err := k8ssecret.ExtractToken(secret, consts.KeyEdgeConnectId)
	if err != nil {
		_log.Debug("EdgeConnect ID token not found")

		return err
	}

	oauthClientId, err := k8ssecret.ExtractToken(secret, consts.KeyEdgeConnectOauthClientID)
	if err != nil {
		_log.Debug("EdgeConnect OAuth client token not found")

		return err
	}

	edgeConnectResponse, err := edgeConnectClient.GetEdgeConnect(id)
	if err != nil {
		_log.Debug("EdgeConnect object not found")

		return errors.WithStack(err)
	}

	if slices.Equal(ec.HostPatterns(), edgeConnectResponse.HostPatterns) {
		_log.Debug("EdgeConnect host patterns in response match", "patterns", ec.Spec.HostPatterns)

		return nil
	}

	log.Debug("updating EdgeConnect", "name", ec.Name)

	err = edgeConnectClient.UpdateEdgeConnect(id, edgeconnectClient.NewRequest(ec.Name, ec.HostPatterns(), ec.HostMappings(), oauthClientId))
	if err != nil {
		_log.Debug("updating EdgeConnect failed")

		return errors.WithStack(err)
	}

	_log.Debug("EdgeConnect updated")

	return nil
}

func (controller *Controller) createOrUpdateEdgeConnectDeploymentAndSettings(ctx context.Context, ec *edgeconnect.EdgeConnect) error {
	clientSecretName := ec.ClientSecretName()

	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name, "clientSecretName", clientSecretName)

	edgeConnectToken, secretHash, err := controller.createOrUpdateEdgeConnectConfigSecret(ctx, ec)
	if err != nil {
		return err
	}

	desiredDeployment := deployment.New(ec)

	desiredDeployment.Spec.Template.Annotations = map[string]string{consts.EdgeConnectAnnotationSecretHash: secretHash}
	_log = _log.WithValues("deploymentName", desiredDeployment.Name)

	if err := controllerutil.SetControllerReference(ec, desiredDeployment, scheme.Scheme); err != nil {
		_log.Debug("Could not set controller reference")

		return errors.WithStack(err)
	}

	ddHash, err := hasher.GenerateHash(desiredDeployment)
	if err != nil {
		_log.Debug("EdgeConnect hash generation failed")

		return err
	}

	desiredDeployment.Annotations[hasher.AnnotationHash] = ddHash

	_, err = k8sdeployment.Query(controller.client, controller.apiReader, _log).WithOwner(ec).CreateOrUpdate(ctx, desiredDeployment)
	if err != nil {
		_log.Debug("could not create or update deployment for EdgeConnect")

		return err
	}

	if ec.IsK8SAutomationEnabled() {
		edgeConnectClient, err := controller.buildEdgeConnectClient(ctx, ec)
		if err != nil {
			_log.Debug("building EdgeConnect client failed")

			return err
		}

		err = controller.createOrUpdateConnectionSetting(edgeConnectClient, ec, edgeConnectToken)
		if err != nil {
			_log.Debug("creating EdgeConnect connection setting failed")

			return err
		}

		_log.Debug("EdgeConnect deployment created/updated successfully")
	}

	return nil
}

func (controller *Controller) createOrUpdateConnectionSetting(edgeConnectClient edgeconnectClient.Client, ec *edgeconnect.EdgeConnect, latestToken string) error {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name)

	envSetting, err := GetConnectionSetting(edgeConnectClient, ec.Name, ec.Namespace, ec.Status.KubeSystemUID)
	if err != nil {
		_log.Info("Failed getting EdgeConnect connection setting object")

		return err
	}

	if (envSetting == edgeconnectClient.EnvironmentSetting{}) {
		_log.Debug("Creating edgeconnectClient connection setting object...")

		err = edgeConnectClient.CreateConnectionSetting(
			edgeconnectClient.EnvironmentSetting{
				SchemaId: edgeconnectClient.KubernetesConnectionSchemaID,
				Scope:    edgeconnectClient.KubernetesConnectionScope,
				Value: edgeconnectClient.EnvironmentSettingValue{
					Name:      ec.Name,
					UID:       ec.Status.KubeSystemUID,
					Namespace: ec.Namespace,
					Token:     latestToken,
				},
			},
		)
		if err != nil {
			return err
		}
	}

	if (envSetting != edgeconnectClient.EnvironmentSetting{}) {
		if envSetting.Value.Token != latestToken {
			_log.Debug("Updating EdgeConnect connection setting object...")

			envSetting.Value.Token = latestToken
			err = edgeConnectClient.UpdateConnectionSetting(envSetting)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (controller *Controller) createOrUpdateEdgeConnectConfigSecret(ctx context.Context, ec *edgeconnect.EdgeConnect) (token string, hash string, err error) {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name)

	// Get a Token from edgeconnectClient.yaml secret data
	token, err = controller.getToken(ctx, ec)

	// check token not found and not all errors
	if err != nil {
		if k8serrors.IsNotFound(err) || errors.Is(err, ErrTokenNotFound) || errors.Is(err, ErrUnsupportedConfigFileVersion) {
			_log.Debug("creating new token", "error", err.Error())

			newToken, err := dttoken.New("dt0e01")
			if err != nil {
				conditions.SetSecretGenFailed(ec.Conditions(), consts.SecretConfigConditionType, err)

				return "", "", err
			}

			token = newToken.String()
		} else {
			conditions.SetSecretGenFailed(ec.Conditions(), consts.SecretConfigConditionType, err)

			return "", "", err
		}
	}

	configFile, err := secret.PrepareConfigFile(ctx, ec, controller.apiReader, token)
	if err != nil {
		conditions.SetSecretGenFailed(ec.Conditions(), consts.SecretConfigConditionType, err)

		return "", "", err
	}

	secretData := make(map[string][]byte)
	secretData[consts.EdgeConnectConfigFileName] = configFile

	secretConfig, err := k8ssecret.Build(ec,
		ec.Name+"-"+consts.EdgeConnectSecretSuffix,
		secretData,
	)

	if err != nil {
		conditions.SetSecretGenFailed(ec.Conditions(), consts.SecretConfigConditionType, err)

		return "", "", errors.WithStack(err)
	}

	query := k8ssecret.Query(controller.client, controller.apiReader, log)

	_, err = query.CreateOrUpdate(ctx, secretConfig)
	if err != nil {
		log.Info("could not create or update secret for ec.yaml", "name", secretConfig.Name)
		conditions.SetKubeApiError(ec.Conditions(), consts.SecretConfigConditionType, err)

		return "", "", err
	}

	conditions.SetSecretCreated(ec.Conditions(), consts.SecretConfigConditionType, secretConfig.Name)

	hash, err = hasher.GenerateHash(secretConfig.Data)
	if err != nil {
		conditions.SetSecretGenFailed(ec.Conditions(), consts.SecretConfigConditionType, err)
	}

	return token, hash, err
}

func (controller *Controller) getToken(ctx context.Context, ec *edgeconnect.EdgeConnect) (string, error) {
	query := k8ssecret.Query(controller.client, controller.apiReader, log)
	secretV, err := query.Get(ctx, types.NamespacedName{Name: ec.Name + "-" + consts.EdgeConnectSecretSuffix, Namespace: ec.Namespace})

	if err != nil {
		return "", err
	}

	cfg := secretV.Data[consts.EdgeConnectConfigFileName]

	ecCfg := config.EdgeConnect{}

	err = yaml.Unmarshal(cfg, &ecCfg)
	if err != nil {
		var typeError *yaml.TypeError
		if errors.As(err, &typeError) {
			return "", ErrUnsupportedConfigFileVersion
		}

		return "", errors.WithStack(err)
	}

	if len(ecCfg.Secrets) > 0 {
		return ecCfg.Secrets[0].Token, nil
	}

	return "", ErrTokenNotFound
}

func GetConnectionSetting(edgeConnectClient edgeconnectClient.Client, name, namespace, uid string) (edgeconnectClient.EnvironmentSetting, error) {
	connectionSettings, err := edgeConnectClient.GetConnectionSettings()
	if err != nil {
		return edgeconnectClient.EnvironmentSetting{}, err
	}

	for _, connectionSetting := range connectionSettings {
		if connectionSetting.Value.Name == name &&
			connectionSetting.Value.Namespace == namespace &&
			connectionSetting.Value.UID == uid {
			return connectionSetting, nil
		}
	}

	return edgeconnectClient.EnvironmentSetting{}, nil
}
