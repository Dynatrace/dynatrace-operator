package edgeconnect

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/deployment"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	k8sdeployment "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/deployment"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

type oauthCredentialsType struct {
	clientId     string
	clientSecret string
}

type edgeConnectClientBuilderType func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error)

// Controller reconciles an EdgeConnect object
type Controller struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the api-server
	client                   client.Client
	apiReader                client.Reader
	registryClientBuilder    registry.ClientBuilder
	scheme                   *runtime.Scheme
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
		scheme:                   mgr.GetScheme(),
		registryClientBuilder:    registry.NewClient,
		config:                   mgr.GetConfig(),
		timeProvider:             timeprovider.New(),
		edgeConnectClientBuilder: newEdgeConnectClient(),
	}
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgeconnectv1alpha1.EdgeConnect{}).
		Owns(&appsv1.Deployment{}).
		Complete(controller)
}

func (controller *Controller) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	_log := log.WithValues("namespace", request.Namespace, "name", request.Name)

	_log.Info("reconciling EdgeConnect")

	edgeConnect, err := controller.getEdgeConnect(ctx, request.Name, request.Namespace)
	if err != nil {
		_log.Debug("reconciliation of EdgeConnect failed")

		return reconcile.Result{}, err
	} else if edgeConnect == nil {
		_log.Debug("EdgeConnect object does not exist")

		return reconcile.Result{}, nil
	}

	if deletionTimestamp := edgeConnect.GetDeletionTimestamp(); deletionTimestamp != nil {
		_log.Debug("EdgeConnect object shall be deleted", "timestamp", deletionTimestamp.String())

		return reconcile.Result{}, controller.reconcileEdgeConnectDeletion(ctx, edgeConnect)
	}

	_log.Debug("EdgeConnect object needs reconcile")

	return controller.reconcileEdgeConnect(ctx, edgeConnect)
}

func (controller *Controller) reconcileEdgeConnectDeletion(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name)

	_log.Info("reconciling EdgeConnect deletion", "name", edgeConnect.Name, "namespace", edgeConnect.Namespace)

	edgeConnectIdFromSecret, err := controller.getEdgeConnectIdFromClientSecret(ctx, edgeConnect)
	if err != nil {
		return err
	}

	edgeConnect.ObjectMeta.Finalizers = nil
	if err := controller.client.Update(ctx, edgeConnect); err != nil {
		_log.Debug("updating the EdgeConnect object failed")

		return errors.WithStack(err)
	}

	edgeConnectClient, err := controller.buildEdgeConnectClient(ctx, edgeConnect)
	if err != nil {
		_log.Debug("building EdgeConnect client failed")

		return err
	}

	tenantEdgeConnect, err := getEdgeConnectByName(edgeConnectClient, edgeConnect.Name)
	if err != nil {
		return err
	}

	switch {
	case tenantEdgeConnect.ID == "":
		_log.Info("EdgeConnect not found on the tenant")
	case !tenantEdgeConnect.ManagedByDynatraceOperator:
		_log.Info("can't delete EdgeConnect configuration from the tenant because it has been created manually by a user")
	case edgeConnectIdFromSecret == "":
		_log.Info("EdgeConnect client secret is missing")

		return edgeConnectClient.DeleteEdgeConnect(tenantEdgeConnect.ID)
	default:
		if tenantEdgeConnect.ID != edgeConnectIdFromSecret {
			_log.Info("EdgeConnect client secret contains invalid Id")
		}

		return edgeConnectClient.DeleteEdgeConnect(tenantEdgeConnect.ID)
	}

	return nil
}

func (controller *Controller) reconcileEdgeConnect(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) (reconcile.Result, error) {
	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name)

	oldStatus := *edgeConnect.Status.DeepCopy()

	err := controller.reconcileEdgeConnectCR(ctx, edgeConnect)
	if err != nil {
		edgeConnect.Status.SetPhase(status.Error)
		_log.Debug("error reconciling EdgeConnect, setting phase 'Error'")
	} else {
		_log.Debug("moving EdgeConnect to phase 'Running'")
		edgeConnect.Status.SetPhase(status.Running)
	}

	if isDifferentStatus, err := hasher.IsDifferent(oldStatus, edgeConnect.Status); err != nil {
		_log.Error(errors.WithStack(err), "failed to generate hash for the status section")
	} else if isDifferentStatus {
		_log.Info("status changed, updating EdgeConnect")

		if errClient := controller.updateEdgeConnectStatus(ctx, edgeConnect); errClient != nil {
			retErr := errors.WithMessagef(errClient, "failed to update EdgeConnect after failure, original error: %s", err)

			_log.Debug("reconcileEdgeConnect error")

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

func (controller *Controller) reconcileEdgeConnectCR(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name)

	if err := controller.updateFinalizers(ctx, edgeConnect); err != nil {
		_log.Debug("updating finalizers failed")

		return err
	}

	if err := controller.updateVersionInfo(ctx, edgeConnect); err != nil {
		_log.Debug("updating version info failed")

		return err
	}

	if edgeConnect.Spec.OAuth.Provisioner {
		_log.Debug("reconcile EdgeConnect provisioner")

		return controller.reconcileEdgeConnectProvisioner(ctx, edgeConnect)
	}

	_log.Debug("reconcile regular EdgeConnect")

	return controller.reconcileEdgeConnectRegular(ctx, edgeConnect)
}

func (controller *Controller) getEdgeConnect(ctx context.Context, name, namespace string) (*edgeconnectv1alpha1.EdgeConnect, error) {
	edgeConnect := &edgeconnectv1alpha1.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := controller.apiReader.Get(ctx, client.ObjectKey{Name: edgeConnect.Name, Namespace: edgeConnect.Namespace}, edgeConnect)
	if k8serrors.IsNotFound(err) {
		log.Debug("EdgeConnect object not found", "name", edgeConnect.Name, "namespace", edgeConnect.Namespace)

		return nil, nil //nolint: nilnil
	} else if err != nil {
		log.Debug("Unable to get EdgeConnect object ",
			"name", edgeConnect.Name, "namespace", edgeConnect.Namespace)

		return nil, errors.WithStack(err)
	}

	return edgeConnect, nil
}

func (controller *Controller) updateFinalizers(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name)

	if edgeConnect.Spec.OAuth.Provisioner && len(edgeConnect.ObjectMeta.Finalizers) == 0 {
		_log.Info("updating finalizers")

		edgeConnect.ObjectMeta.Finalizers = []string{finalizerName}
		if err := controller.client.Update(ctx, edgeConnect); err != nil {
			_log.Debug("updating finalizers failed")

			return errors.WithStack(err)
		}
	}

	return nil
}

func (controller *Controller) updateVersionInfo(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name)

	_log.Info("updating version info")

	transport := http.DefaultTransport.(*http.Transport).Clone()
	keyChainSecret := edgeConnect.EmptyPullSecret()

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

	versionReconciler := version.NewReconciler(controller.apiReader, registryClient, timeprovider.New(), edgeConnect)
	if err = versionReconciler.Reconcile(ctx); err != nil {
		_log.Debug("reconciliation of EdgeConnect version failed")

		return err
	}

	_log.Debug("EdgeConnect version info updated")

	return nil
}

func (controller *Controller) updateEdgeConnectStatus(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name)

	edgeConnect.Status.UpdatedTimestamp = *controller.timeProvider.Now()

	err := controller.client.Status().Update(ctx, edgeConnect)
	if k8serrors.IsConflict(err) {
		_log.Info("could not update EdgeConnect status due to conflict")

		return errors.WithStack(err)
	} else if err != nil {
		return errors.WithStack(err)
	}

	_log.Info("EdgeConnect status updated", "timestamp", edgeConnect.Status.UpdatedTimestamp)

	return nil
}

func (controller *Controller) reconcileEdgeConnectRegular(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	desiredDeployment := deployment.New(edgeConnect)

	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name, "deploymentName", desiredDeployment.Name)

	if err := controllerutil.SetControllerReference(edgeConnect, desiredDeployment, controller.scheme); err != nil {
		return errors.WithStack(err)
	}

	ddHash, err := hasher.GenerateHash(desiredDeployment)
	if err != nil {
		_log.Debug("Unable to generate hash for EdgeConnect deployment")

		return err
	}

	desiredDeployment.Annotations[hasher.AnnotationHash] = ddHash

	if err := controller.createOrUpdateEdgeConnectConfigSecret(ctx, edgeConnect); err != nil {
		return err
	}

	_, err = k8sdeployment.CreateOrUpdateDeployment(controller.client, log, desiredDeployment)
	if err != nil {
		_log.Info("could not create or update deployment for EdgeConnect")

		return err
	}

	return nil
}

func (controller *Controller) reconcileEdgeConnectProvisioner(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error { //nolint: revive
	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name)

	_log.Info("reconcileEdgeConnectProvisioner")

	edgeConnectClient, err := controller.buildEdgeConnectClient(ctx, edgeConnect)
	if err != nil {
		_log.Debug("unable to build EdgeConnect client")

		return err
	}

	tenantEdgeConnect, err := getEdgeConnectByName(edgeConnectClient, edgeConnect.Name)
	if err != nil {
		return err
	}

	edgeConnectIdFromSecret, err := controller.getEdgeConnectIdFromClientSecret(ctx, edgeConnect)
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
		err := controller.createEdgeConnect(ctx, edgeConnectClient, edgeConnect)
		if err != nil {
			return err
		}

		return controller.createOrUpdateEdgeConnectDeployment(ctx, edgeConnect)
	}

	err = controller.updateEdgeConnect(ctx, edgeConnectClient, edgeConnect)
	if err != nil {
		return err
	}

	return controller.createOrUpdateEdgeConnectDeployment(ctx, edgeConnect)
}

func (controller *Controller) buildEdgeConnectClient(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) (edgeconnect.Client, error) {
	oauthCredentials, err := controller.getOauthCredentials(ctx, edgeConnect)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return controller.edgeConnectClientBuilder(ctx, edgeConnect, oauthCredentials)
}

func (controller *Controller) getOauthCredentials(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) (oauthCredentialsType, error) {
	query := k8ssecret.NewQuery(ctx, controller.client, controller.apiReader, log)

	secret, err := query.Get(types.NamespacedName{
		Name:      edgeConnect.Spec.OAuth.ClientSecret,
		Namespace: edgeConnect.Namespace,
	})
	if err != nil {
		return oauthCredentialsType{}, errors.WithStack(err)
	}

	oauthClientId, err := k8ssecret.ExtractToken(&secret, consts.KeyEdgeConnectOauthClientID)
	if err != nil {
		return oauthCredentialsType{}, errors.WithStack(err)
	}

	oauthClientSecret, err := k8ssecret.ExtractToken(&secret, consts.KeyEdgeConnectOauthClientSecret)
	if err != nil {
		return oauthCredentialsType{}, errors.WithStack(err)
	}

	return oauthCredentialsType{clientId: oauthClientId, clientSecret: oauthClientSecret}, nil
}

func newEdgeConnectClient() func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
	return func(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect, oauthCredentials oauthCredentialsType) (edgeconnect.Client, error) {
		edgeConnectClient, err := edgeconnect.NewClient(
			oauthCredentials.clientId,
			oauthCredentials.clientSecret,
			edgeconnect.WithBaseURL("https://"+edgeConnect.Spec.ApiServer+"/platform/app-engine/edge-connect/v1"),
			edgeconnect.WithTokenURL(edgeConnect.Spec.OAuth.Endpoint),
			edgeconnect.WithOauthScopes([]string{
				"app-engine:edge-connects:read",
				"app-engine:edge-connects:write",
				"app-engine:edge-connects:delete",
				"oauth2:clients:manage",
			}),
			edgeconnect.WithContext(ctx),
		)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		return edgeConnectClient, nil
	}
}

func getEdgeConnectByName(edgeConnectClient edgeconnect.Client, name string) (edgeconnect.GetResponse, error) {
	_log := log.WithValues("name", name)

	ecs, err := edgeConnectClient.GetEdgeConnects(name)
	if err != nil {
		log.Debug("Unable to get EdgeConnect object")

		return edgeconnect.GetResponse{}, errors.WithStack(err)
	}

	if len(ecs.EdgeConnects) > 1 {
		_log.Debug("Found multiple EdgeConnect objects with the same name", "count", ecs.EdgeConnects)

		return edgeconnect.GetResponse{}, errors.New("many EdgeConnects have the same name")
	}

	if len(ecs.EdgeConnects) == 1 {
		_log.Debug("Found one EdgeConnect objects with matching name", "count", ecs.EdgeConnects)

		return ecs.EdgeConnects[0], nil
	}

	_log.Debug("No EdgeConnect object found with matching name", "count", ecs.EdgeConnects)

	return edgeconnect.GetResponse{}, nil
}

func (controller *Controller) getEdgeConnectIdFromClientSecret(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) (string, error) {
	clientSecretName := edgeConnect.ClientSecretName()

	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name, "clientSecretName", clientSecretName)

	query := k8ssecret.NewQuery(ctx, controller.client, controller.apiReader, log)

	secret, err := query.Get(types.NamespacedName{Name: clientSecretName, Namespace: edgeConnect.Namespace})
	if err != nil {
		if k8serrors.IsNotFound(errors.Cause(err)) {
			_log.Debug("EdgeConnect client secret not found")

			return "", nil
		} else {
			_log.Debug("EdgeConnect client secret query failed")

			return "", errors.WithStack(err)
		}
	}

	id, err := k8ssecret.ExtractToken(&secret, consts.KeyEdgeConnectId)
	if err != nil {
		log.Debug("unable to extract EdgeConnect tokens")

		return "", errors.WithStack(err)
	}

	log.Debug("successfully read EdgeConnect id from client secret", "id", "***")

	return id, nil
}

func (controller *Controller) createEdgeConnect(ctx context.Context, edgeConnectClient edgeconnect.Client, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name)

	createResponse, err := edgeConnectClient.CreateEdgeConnect(edgeConnect.Name, edgeConnect.Spec.HostPatterns, "")
	if err != nil {
		_log.Debug("creating EdgeConnect failed")

		return errors.WithStack(err)
	}

	_log.Debug("createResponse", "id", createResponse.ID)

	ecOAuthSecret, err := k8ssecret.Create(controller.scheme, edgeConnect,
		k8ssecret.NewNameModifier(edgeConnect.ClientSecretName()),
		k8ssecret.NewNamespaceModifier(edgeConnect.Namespace),
		k8ssecret.NewDataModifier(map[string][]byte{
			consts.KeyEdgeConnectOauthClientID:     []byte(createResponse.OauthClientId),
			consts.KeyEdgeConnectOauthClientSecret: []byte(createResponse.OauthClientSecret),
			consts.KeyEdgeConnectOauthResource:     []byte(createResponse.OauthClientResource),
			consts.KeyEdgeConnectId:                []byte(createResponse.ID),
		}))

	if err != nil {
		_log.Debug("unable to create EdgeConnect secret")

		return errors.WithStack(err)
	}

	query := k8ssecret.NewQuery(ctx, controller.client, controller.apiReader, _log)

	err = query.CreateOrUpdate(*ecOAuthSecret)
	if err != nil {
		_log.Debug("could not create or update secret for edge-connect client")

		return errors.WithStack(err)
	}

	_log.Debug("EdgeConnect created")

	return nil
}

func (controller *Controller) updateEdgeConnect(ctx context.Context, edgeConnectClient edgeconnect.Client, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name)

	secretQuery := k8ssecret.NewQuery(ctx, controller.client, controller.apiReader, log)

	secret, err := secretQuery.Get(types.NamespacedName{Name: edgeConnect.ClientSecretName(), Namespace: edgeConnect.Namespace})
	if err != nil {
		_log.Debug("EdgeConnect ID token not found")

		return err
	}

	id, err := k8ssecret.ExtractToken(&secret, consts.KeyEdgeConnectId)
	if err != nil {
		_log.Debug("EdgeConnect ID token not found")

		return err
	}

	oauthClientId, err := k8ssecret.ExtractToken(&secret, consts.KeyEdgeConnectOauthClientID)
	if err != nil {
		_log.Debug("EdgeConnect OAuth client token not found")

		return err
	}

	edgeConnectResponse, err := edgeConnectClient.GetEdgeConnect(id)
	if err != nil {
		_log.Debug("EdgeConnect object not found")

		return errors.WithStack(err)
	}

	if slices.Equal(edgeConnect.Spec.HostPatterns, edgeConnectResponse.HostPatterns) {
		_log.Debug("EdgeConnect host patterns in response match", "patterns", edgeConnect.Spec.HostPatterns)

		return nil
	}

	log.Debug("updating EdgeConnect", "name", edgeConnect.Name)

	err = edgeConnectClient.UpdateEdgeConnect(id, edgeConnect.Name, edgeConnect.Spec.HostPatterns, oauthClientId)
	if err != nil {
		_log.Debug("updating EdgeConnect failed")

		return errors.WithStack(err)
	}

	_log.Debug("EdgeConnect updated")

	return nil
}

func (controller *Controller) createOrUpdateEdgeConnectDeployment(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	clientSecretName := edgeConnect.ClientSecretName()

	_log := log.WithValues("namespace", edgeConnect.Namespace, "name", edgeConnect.Name, "clientSecretName", clientSecretName)

	err := controller.createOrUpdateEdgeConnectConfigSecret(ctx, edgeConnect)
	if err != nil {
		return err
	}

	desiredDeployment := deployment.New(edgeConnect)
	_log = _log.WithValues("deploymentName", desiredDeployment.Name)

	if err := controllerutil.SetControllerReference(edgeConnect, desiredDeployment, controller.scheme); err != nil {
		_log.Debug("Could not set controller reference")

		return errors.WithStack(err)
	}

	ddHash, err := hasher.GenerateHash(desiredDeployment)
	if err != nil {
		_log.Debug("EdgeConnect hash generation failed")

		return err
	}

	desiredDeployment.Annotations[hasher.AnnotationHash] = ddHash

	_, err = k8sdeployment.CreateOrUpdateDeployment(controller.client, _log, desiredDeployment)
	if err != nil {
		_log.Debug("could not create or update deployment for EdgeConnect")

		return err
	}

	_log.Debug("EdgeConnect deployment created/updated successfully")

	return nil
}

func (controller *Controller) createOrUpdateEdgeConnectConfigSecret(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	configFile, err := secret.PrepareConfigFile(ctx, edgeConnect, controller.apiReader)
	if err != nil {
		return err
	}

	secretData := make(map[string][]byte)
	secretData[consts.EdgeConnectConfigFileName] = configFile

	secretConfig, err := k8ssecret.Create(controller.scheme, edgeConnect,
		k8ssecret.NewNameModifier(consts.EdgeConnectConfigVolumeMountName),
		k8ssecret.NewNamespaceModifier(edgeConnect.Namespace),
		k8ssecret.NewDataModifier(secretData))

	if err != nil {
		return errors.WithStack(err)
	}

	query := k8ssecret.NewQuery(ctx, controller.client, controller.apiReader, log)

	err = query.CreateOrUpdate(*secretConfig)
	if err != nil {
		log.Info("could not create or update secret for edgeConnect.yaml", "name", secretConfig.Name)

		return err
	}

	return nil
}
