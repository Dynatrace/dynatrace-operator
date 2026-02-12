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
	ecsecret "github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sevent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/system"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	fastUpdateInterval    = 1 * time.Minute
	defaultUpdateInterval = 30 * time.Minute

	controllerName = "edgeconnect-controller"
	finalizerName  = "server"
)

var (
	ErrTokenNotFound                = errors.New("token not found")
	ErrUnsupportedConfigFileVersion = errors.New("unsupported config file version")
)

type oauthCredentialsType struct {
	clientID     string
	clientSecret string
}

type edgeConnectClientBuilderType func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, customCA []byte) (edgeconnectClient.Client, error)

// Controller reconciles an EdgeConnect object
type Controller struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the api-server
	client                   client.Client
	apiReader                client.Reader
	eventRecorder            record.EventRecorder
	registryClientBuilder    registry.ClientBuilder
	config                   *rest.Config
	timeProvider             *timeprovider.Provider
	edgeConnectClientBuilder edgeConnectClientBuilderType
	secrets                  k8ssecret.QueryObject
}

func Add(mgr manager.Manager, _ string) error {
	return NewController(mgr).SetupWithManager(mgr)
}

func NewController(mgr manager.Manager) *Controller {
	return &Controller{
		client:                   mgr.GetClient(),
		apiReader:                mgr.GetAPIReader(),
		eventRecorder:            mgr.GetEventRecorderFor(controllerName), //nolint
		registryClientBuilder:    registry.NewClient,
		config:                   mgr.GetConfig(),
		timeProvider:             timeprovider.New(),
		edgeConnectClientBuilder: newEdgeConnectClient(),
		secrets:                  k8ssecret.Query(mgr.GetClient(), mgr.GetAPIReader(), log),
	}
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgeconnect.EdgeConnect{}).
		Named(controllerName).
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

	isCrdLatestVersion, err := k8scrd.IsLatestVersion(ctx, controller.apiReader, k8scrd.EdgeConnectName)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !isCrdLatestVersion {
		_log.Debug("sending k8s event about CRD version mismatch")
		k8sevent.SendCRDVersionMismatch(controller.eventRecorder, ec)
	}

	if deletionTimestamp := ec.GetDeletionTimestamp(); deletionTimestamp != nil {
		_log.Debug("EdgeConnect object shall be deleted", "timestamp", deletionTimestamp.String())

		return reconcile.Result{}, controller.reconcileEdgeConnectDeletion(ctx, ec)
	}

	_log.Debug("EdgeConnect object needs reconcile")

	return controller.reconcileEdgeConnect(ctx, ec)
}

func (controller *Controller) reconcileEdgeConnectDeletion(ctx context.Context, ec *edgeconnect.EdgeConnect) error {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name, "scenario", "deletion")

	_log.Info("reconciling EdgeConnect deletion", "name", ec.Name, "namespace", ec.Namespace)

	edgeConnectIDFromSecret, err := controller.getEdgeConnectIDFromClientSecret(ctx, ec)
	if err != nil {
		return err
	}

	ec.Finalizers = nil
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
	case edgeConnectIDFromSecret == "":
		_log.Info("EdgeConnect client secret is missing")
	default:
		if tenantEdgeConnect.ID != edgeConnectIDFromSecret {
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
		err = edgeConnectClient.DeleteConnectionSetting(*envSetting.ObjectID)
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

		kubeSystemUID, err := system.GetUID(ctx, controller.apiReader)
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

	if ec.IsProvisionerModeEnabled() && len(ec.Finalizers) == 0 {
		_log.Info("updating finalizers")

		ec.Finalizers = []string{finalizerName}
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
		registry.WithAPIReader(controller.apiReader),
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
	depl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ec.Name,
			Namespace: ec.Namespace,
		},
	}
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name, "deploymentName", depl.Name)

	op, err := controllerutil.CreateOrUpdate(ctx, controller.client, depl, func() error {
		_, secretHash, err := controller.createOrUpdateEdgeConnectConfigSecret(ctx, ec)
		if err != nil {
			return err
		}

		if depl.Spec.Template.Annotations == nil {
			depl.Spec.Template.Annotations = map[string]string{}
		}
		depl.Annotations = maputils.MergeMap(depl.Annotations, deployment.Annotations())
		depl.Labels = maputils.MergeMap(depl.Labels, deployment.Labels(ec))
		depl.Spec.Template.Annotations[consts.EdgeConnectAnnotationSecretHash] = secretHash

		depl.Spec = deployment.CreateSpec(ec)

		if err := controllerutil.SetControllerReference(ec, depl, scheme.Scheme); err != nil {
			return errors.WithStack(err)
		}

		return nil
	})
	if err != nil {
		_log.Info("could not create or update deployment for EdgeConnect")

		return err
	}

	_log.Info("deployment for EdgeConnect", "operation", op)

	return nil
}

// func (controller *Controller) reconcileEdgeConnectRegular(ctx context.Context, ec *edgeconnect.EdgeConnect) error {
//	desiredDeployment := &appsv1.Deployment{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:        ec.Name,
//			Namespace:   ec.Namespace,
//			Labels:      deployment.Labels(ec),
//			Annotations: deployment.Annotations(),
//		},
//	}
//	desiredDeployment.Spec = deployment.CreateSpec(ec)
//
//	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name, "deploymentName", desiredDeployment.Name)
//
//	if err := controllerutil.SetControllerReference(ec, desiredDeployment, scheme.Scheme); err != nil {
//		return errors.WithStack(err)
//	}
//
//	_, secretHash, err := controller.createOrUpdateEdgeConnectConfigSecret(ctx, ec)
//	if err != nil {
//		return err
//	}
//
//	if desiredDeployment.Spec.Template.Annotations == nil {
//		desiredDeployment.Spec.Template.Annotations = map[string]string{}
//	}
//
//	desiredDeployment.Spec.Template.Annotations[consts.EdgeConnectAnnotationSecretHash] = secretHash
//
//	_, err = k8sdeployment.Query(controller.client, controller.apiReader, log).WithOwner(ec).CreateOrUpdate(ctx, desiredDeployment)
//	if err != nil {
//		_log.Info("could not create or update deployment for EdgeConnect")
//
//		return err
//	}
//
//	return nil
//}

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

	edgeConnectIDFromSecret, err := controller.getEdgeConnectIDFromClientSecret(ctx, ec)
	if err != nil {
		return err
	}

	if tenantEdgeConnect.ID != "" && !tenantEdgeConnect.ManagedByDynatraceOperator {
		_log.Info("can't delete EdgeConnect configuration from the tenant because it has been created manually by a user", "name", tenantEdgeConnect.Name)

		return nil
	}

	if tenantEdgeConnect.ID != "" {
		if edgeConnectIDFromSecret == "" {
			_log.Info("EdgeConnect has to be recreated due to missing secret")

			if err := edgeConnectClient.DeleteEdgeConnect(tenantEdgeConnect.ID); err != nil {
				return err
			}

			tenantEdgeConnect.ID = ""
		} else if tenantEdgeConnect.ID != edgeConnectIDFromSecret {
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

	customCA, err := ec.TrustedCAs(ctx, controller.client)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return controller.edgeConnectClientBuilder(ctx, ec, oauthCredentials, customCA)
}

func (controller *Controller) getOauthCredentials(ctx context.Context, ec *edgeconnect.EdgeConnect) (oauthCredentialsType, error) {
	secret, err := controller.secrets.Get(ctx, types.NamespacedName{
		Name:      ec.Spec.OAuth.ClientSecret,
		Namespace: ec.Namespace,
	})
	if err != nil {
		return oauthCredentialsType{}, errors.WithStack(err)
	}

	oauthClientID, err := k8ssecret.ExtractToken(secret, consts.KeyEdgeConnectOauthClientID)
	if err != nil {
		return oauthCredentialsType{}, errors.WithStack(err)
	}

	oauthClientSecret, err := k8ssecret.ExtractToken(secret, consts.KeyEdgeConnectOauthClientSecret)
	if err != nil {
		return oauthCredentialsType{}, errors.WithStack(err)
	}

	return oauthCredentialsType{clientID: oauthClientID, clientSecret: oauthClientSecret}, nil
}

func newEdgeConnectClient() func(context.Context, *edgeconnect.EdgeConnect, oauthCredentialsType, []byte) (edgeconnectClient.Client, error) {
	return func(ctx context.Context, ec *edgeconnect.EdgeConnect, oauthCredentials oauthCredentialsType, customCA []byte) (edgeconnectClient.Client, error) {
		oauthScopes := []string{
			"app-engine:edge-connects:read",
			"app-engine:edge-connects:write",
			"app-engine:edge-connects:delete",
			"oauth2:clients:manage",
		}
		if ec.IsK8SAutomationEnabled() {
			oauthScopes = append(oauthScopes, "settings:objects:read", "settings:objects:write")
		}

		edgeConnectClient, err := edgeconnectClient.NewClient(
			oauthCredentials.clientID,
			oauthCredentials.clientSecret,
			edgeconnectClient.WithBaseURL("https://"+ec.Spec.APIServer),
			edgeconnectClient.WithTokenURL(ec.Spec.OAuth.Endpoint),
			edgeconnectClient.WithOauthScopes(oauthScopes),
			edgeconnectClient.WithContext(ctx),
			edgeconnectClient.WithCustomCA(customCA),
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

func (controller *Controller) getEdgeConnectIDFromClientSecret(ctx context.Context, ec *edgeconnect.EdgeConnect) (string, error) {
	clientSecretName := ec.ClientSecretName()

	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name, "clientSecretName", clientSecretName)

	secrets := k8ssecret.Query(controller.client, controller.apiReader, _log)

	secret, err := secrets.Get(ctx, types.NamespacedName{Name: clientSecretName, Namespace: ec.Namespace})
	if err != nil {
		if k8serrors.IsNotFound(errors.Cause(err)) {
			_log.Debug("EdgeConnect client secret not found")

			return "", nil
		} else {
			_log.Debug("EdgeConnect client secret query failed")

			return "", errors.WithStack(err)
		}
	}

	id, err := k8ssecret.ExtractToken(secret, consts.KeyEdgeConnectID)
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
		consts.KeyEdgeConnectOauthClientID:     []byte(createResponse.OauthClientID),
		consts.KeyEdgeConnectOauthClientSecret: []byte(createResponse.OauthClientSecret),
		consts.KeyEdgeConnectOauthResource:     []byte(createResponse.OauthClientResource),
		consts.KeyEdgeConnectID:                []byte(createResponse.ID)})
	if err != nil {
		_log.Debug("unable to create EdgeConnect secret")

		return errors.WithStack(err)
	}

	secrets := k8ssecret.Query(controller.client, controller.apiReader, _log)

	_, err = secrets.CreateOrUpdate(ctx, ecOAuthSecret)
	if err != nil {
		_log.Debug("could not create or update secret for edge-connect client")

		return errors.WithStack(err)
	}

	_log.Debug("EdgeConnect created")

	return nil
}

func (controller *Controller) updateEdgeConnect(ctx context.Context, edgeConnectClient edgeconnectClient.Client, ec *edgeconnect.EdgeConnect) error {
	_log := log.WithValues("namespace", ec.Namespace, "name", ec.Name)

	secret, err := controller.secrets.Get(ctx, types.NamespacedName{Name: ec.ClientSecretName(), Namespace: ec.Namespace})
	if err != nil {
		_log.Debug("EdgeConnect ID token not found")

		return err
	}

	id, err := k8ssecret.ExtractToken(secret, consts.KeyEdgeConnectID)
	if err != nil {
		_log.Debug("EdgeConnect ID token not found")

		return err
	}

	oauthClientID, err := k8ssecret.ExtractToken(secret, consts.KeyEdgeConnectOauthClientID)
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

	err = edgeConnectClient.UpdateEdgeConnect(id, edgeconnectClient.NewRequest(ec.Name, ec.HostPatterns(), ec.HostMappings(), oauthClientID))
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

	if desiredDeployment.Spec.Template.Annotations == nil {
		desiredDeployment.Spec.Template.Annotations = map[string]string{}
	}

	desiredDeployment.Spec.Template.Annotations[consts.EdgeConnectAnnotationSecretHash] = secretHash
	_log = _log.WithValues("deploymentName", desiredDeployment.Name)

	if err := controllerutil.SetControllerReference(ec, desiredDeployment, scheme.Scheme); err != nil {
		_log.Debug("Could not set controller reference")

		return errors.WithStack(err)
	}

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
				SchemaID: edgeconnectClient.KubernetesConnectionSchemaID,
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
				k8sconditions.SetSecretGenFailed(ec.Conditions(), consts.SecretConfigConditionType, err)

				return "", "", err
			}

			token = newToken.String()
		} else {
			k8sconditions.SetSecretGenFailed(ec.Conditions(), consts.SecretConfigConditionType, err)

			return "", "", err
		}
	}

	configFile, err := ecsecret.PrepareConfigFile(ctx, ec, controller.apiReader, token)
	if err != nil {
		k8sconditions.SetSecretGenFailed(ec.Conditions(), consts.SecretConfigConditionType, err)

		return "", "", err
	}

	secretData := make(map[string][]byte)
	secretData[consts.EdgeConnectConfigFileName] = configFile

	secretConfig, err := k8ssecret.Build(ec,
		ec.Name+"-"+consts.EdgeConnectSecretSuffix,
		secretData,
	)
	if err != nil {
		k8sconditions.SetSecretGenFailed(ec.Conditions(), consts.SecretConfigConditionType, err)

		return "", "", errors.WithStack(err)
	}

	_, err = controller.secrets.CreateOrUpdate(ctx, secretConfig)
	if err != nil {
		log.Info("could not create or update secret for ec.yaml", "name", secretConfig.Name)
		k8sconditions.SetKubeAPIError(ec.Conditions(), consts.SecretConfigConditionType, err)

		return "", "", err
	}

	k8sconditions.SetSecretCreated(ec.Conditions(), consts.SecretConfigConditionType, secretConfig.Name)

	hash, err = hasher.GenerateHash(secretConfig.Data)
	if err != nil {
		k8sconditions.SetSecretGenFailed(ec.Conditions(), consts.SecretConfigConditionType, err)
	}

	return token, hash, err
}

func (controller *Controller) getToken(ctx context.Context, ec *edgeconnect.EdgeConnect) (string, error) {
	secretV, err := controller.secrets.Get(ctx, types.NamespacedName{Name: ec.Name + "-" + consts.EdgeConnectSecretSuffix, Namespace: ec.Namespace})
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
