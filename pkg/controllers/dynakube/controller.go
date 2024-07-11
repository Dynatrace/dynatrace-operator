package dynakube

import (
	"context"
	goerrors "errors"
	"os"
	"time"

	dynatracestatus "github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dynatracev1beta3 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/apimonitoring"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceapi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/injection"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	monitoredentities "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/monitored_entities"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	fastUpdateInterval    = 1 * time.Minute
	changesUpdateInterval = 5 * time.Minute
	defaultUpdateInterval = 30 * time.Minute
)

func Add(mgr manager.Manager, _ string) error {
	kubeSysUID, err := kubesystem.GetUID(context.Background(), mgr.GetAPIReader())
	if err != nil {
		return errors.WithStack(err)
	}

	return NewController(mgr, string(kubeSysUID)).SetupWithManager(mgr)
}

func NewController(mgr manager.Manager, clusterID string) *Controller {
	return NewDynaKubeController(mgr.GetClient(), mgr.GetAPIReader(), mgr.GetConfig(), clusterID)
}

func NewDynaKubeController(kubeClient client.Client, apiReader client.Reader, config *rest.Config, clusterID string) *Controller {
	return &Controller{
		client:                 kubeClient,
		apiReader:              apiReader,
		fs:                     afero.Afero{Fs: afero.NewOsFs()},
		config:                 config,
		operatorNamespace:      os.Getenv(env.PodNamespace),
		clusterID:              clusterID,
		dynatraceClientBuilder: dynatraceclient.NewBuilder(apiReader),
		istioClientBuilder:     istio.NewClient,

		deploymentMetadataReconcilerBuilder: deploymentmetadata.NewReconciler,
		activeGateReconcilerBuilder:         activegate.NewReconciler,
		oneAgentReconcilerBuilder:           oneagent.NewReconciler,
		apiMonitoringReconcilerBuilder:      apimonitoring.NewReconciler,
		injectionReconcilerBuilder:          injection.NewReconciler,
		istioReconcilerBuilder:              istio.NewReconciler,
		extensionBuilder:                    extension.NewReconciler,
	}
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1beta2.DynaKube{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(controller)
}

// Controller reconciles a DynaKube object
type Controller struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the api-server
	client    client.Client
	apiReader client.Reader
	fs        afero.Afero

	dynatraceClientBuilder dynatraceclient.Builder
	config                 *rest.Config
	istioClientBuilder     istio.ClientBuilder

	deploymentMetadataReconcilerBuilder deploymentmetadata.ReconcilerBuilder
	activeGateReconcilerBuilder         activegate.ReconcilerBuilder
	oneAgentReconcilerBuilder           oneagent.ReconcilerBuilder
	apiMonitoringReconcilerBuilder      apimonitoring.ReconcilerBuilder
	injectionReconcilerBuilder          injection.ReconcilerBuilder
	istioReconcilerBuilder              istio.ReconcilerBuilder
	extensionBuilder                    extension.ReconcilerBuilder

	tokens            token.Tokens
	operatorNamespace string
	clusterID         string

	requeueAfter time.Duration
}

// Reconcile reads that state of the cluster for a DynaKube object and makes changes based on the state read
// and what is in the DynaKube.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (controller *Controller) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling DynaKube", "namespace", request.Namespace, "name", request.Name)

	dynaKube, err := controller.getDynakubeOrCleanup(ctx, request.Name, request.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	} else if dynaKube == nil {
		log.Info("reconciling DynaKube finished, no dynakube available", "namespace", request.Namespace, "name", request.Name, "result", "empty")

		return reconcile.Result{}, nil
	}

	oldStatus := *dynaKube.Status.DeepCopy()
	controller.requeueAfter = defaultUpdateInterval
	err = controller.reconcileDynaKube(ctx, dynaKube)
	result, err := controller.handleError(ctx, dynaKube, err, oldStatus)

	log.Info("reconciling DynaKube finished", "namespace", request.Namespace, "name", request.Name, "result", result)

	return result, err
}

func (controller *Controller) getDynakubeOrCleanup(ctx context.Context, dkName, dkNamespace string) (*dynatracev1beta2.DynaKube, error) {
	dynakube := &dynatracev1beta2.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dkName,
			Namespace: dkNamespace,
		},
	}
	err := controller.apiReader.Get(ctx, client.ObjectKey{Name: dynakube.Name, Namespace: dynakube.Namespace}, dynakube)

	if k8serrors.IsNotFound(err) {
		namespaces, err := mapper.GetNamespacesForDynakube(ctx, controller.apiReader, dkName)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to list namespaces for dynakube %s", dkName)
		}

		return nil, controller.createDynakubeMapper(ctx, dynakube).UnmapFromDynaKube(namespaces)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return dynakube, nil
}

func (controller *Controller) handleError(
	ctx context.Context,
	dynaKube *dynatracev1beta2.DynaKube,
	err error,
	oldStatus dynatracev1beta2.DynaKubeStatus,
) (reconcile.Result, error) {
	switch {
	case dynatraceapi.IsUnreachable(err):
		log.Info("the Dynatrace API server is unavailable or request limit reached! trying again in one minute",
			"errorCode", dynatraceapi.StatusCode(err), "errorMessage", dynatraceapi.Message(err))
		// should we set the phase to error ?
		return reconcile.Result{RequeueAfter: fastUpdateInterval}, nil

	case err != nil:
		controller.setRequeueAfterIfNewIsShorter(fastUpdateInterval)
		dynaKube.Status.SetPhase(dynatracestatus.Error)
		log.Error(err, "error reconciling DynaKube", "namespace", dynaKube.Namespace, "name", dynaKube.Name)

	default:
		dynaKube.Status.SetPhase(controller.determineDynaKubePhase(dynaKube))
	}

	if isStatusDifferent, err := hasher.IsDifferent(oldStatus, dynaKube.Status); err != nil {
		log.Error(err, "failed to generate hash for the status section")
	} else if isStatusDifferent {
		log.Info("status changed, updating DynaKube")
		controller.setRequeueAfterIfNewIsShorter(changesUpdateInterval)

		if errClient := dynaKube.UpdateStatus(ctx, controller.client); errClient != nil {
			return reconcile.Result{}, errors.WithMessagef(errClient, "failed to update DynaKube after failure, original error: %s", err)
		}
	}

	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: controller.requeueAfter}, nil
}

func (controller *Controller) setRequeueAfterIfNewIsShorter(requeueAfter time.Duration) {
	if controller.requeueAfter > requeueAfter {
		controller.requeueAfter = requeueAfter
	}
}

func (controller *Controller) reconcileDynaKube(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) error {
	var istioClient *istio.Client

	var err error
	if dynakube.Spec.EnableIstio {
		istioClient, err = controller.setupIstioClient(dynakube)
	}

	if err != nil {
		return err
	}

	if istioClient != nil {
		istioReconciler := controller.istioReconcilerBuilder(istioClient)

		err := istioReconciler.ReconcileAPIUrl(ctx, dynakube)
		if err != nil {
			return errors.WithMessage(err, "failed to reconcile istio objects for API url")
		}
	}

	dynatraceClient, err := controller.setupTokensAndClient(ctx, dynakube)
	if err != nil {
		return err
	}

	dynakube.Status.KubeSystemUUID = controller.clusterID

	log.Info("start reconciling deployment meta data")

	err = controller.deploymentMetadataReconcilerBuilder(controller.client, controller.apiReader, *dynakube, controller.clusterID).Reconcile(ctx)
	if err != nil {
		return err
	}

	log.Info("start reconciling process module config")

	return controller.reconcileComponents(ctx, dynatraceClient, istioClient, dynakube)
}

func (controller *Controller) setupIstioClient(dynakube *dynatracev1beta2.DynaKube) (*istio.Client, error) {
	istioClient, err := controller.istioClientBuilder(controller.config, dynakube)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to initialize istio client")
	}

	isInstalled, err := istioClient.CheckIstioInstalled()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to initialize istio client")
	} else if !isInstalled {
		return nil, errors.New("istio not installed, yet is enabled, aborting reconciliation, check configuration")
	}

	return istioClient, nil
}

func (controller *Controller) setupTokensAndClient(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) (dtclient.Client, error) {
	tokenReader := token.NewReader(controller.apiReader, dynakube)

	tokens, err := tokenReader.ReadTokens(ctx)
	if err != nil {
		controller.setConditionTokenError(dynakube, err)

		return nil, err
	}

	controller.tokens = tokens

	dynatraceClientBuilder := controller.dynatraceClientBuilder.
		SetContext(ctx).
		SetDynakube(*dynakube).
		SetTokens(tokens)

	dynatraceClient, err := dynatraceClientBuilder.BuildWithTokenVerification(&dynakube.Status)
	if err != nil {
		controller.setConditionTokenError(dynakube, err)

		return nil, err
	}

	controller.setConditionTokenReady(dynakube)

	return dynatraceClient, nil
}

func (controller *Controller) reconcileComponents(ctx context.Context, dynatraceClient dtclient.Client, istioClient *istio.Client, dynakube *dynatracev1beta2.DynaKube) error {
	var componentErrors []error

	log.Info("start reconciling ActiveGate")

	err := controller.reconcileActiveGate(ctx, dynakube, dynatraceClient, istioClient)
	if err != nil {
		log.Info("could not reconcile ActiveGate")

		componentErrors = append(componentErrors, err)
	}

	dynakubeV1beta3 := &dynatracev1beta3.DynaKube{}

	err = dynakubeV1beta3.ConvertFrom(dynakube)
	if err != nil {
		return err
	}

	extensionReconciler := extension.NewReconciler(controller.client, controller.apiReader, dynakubeV1beta3)

	err = extensionReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	proxyReconciler := proxy.NewReconciler(controller.client, controller.apiReader, dynakube)

	err = proxyReconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	log.Info("start reconciling monitored entities")

	monitoredEntitiesreconciler := monitoredentities.NewReconciler(dynatraceClient, dynakube, controller.clusterID)

	err = monitoredEntitiesreconciler.Reconcile(ctx)
	if err != nil {
		return err
	}

	log.Info("start reconciling app injection")

	err = controller.injectionReconcilerBuilder(controller.client,
		controller.apiReader,
		dynatraceClient,
		istioClient,
		dynakube).
		Reconcile(ctx)
	if err != nil {
		if errors.Is(err, oaconnectioninfo.NoOneAgentCommunicationHostsError) {
			// missing communication hosts is not an error per se, just make sure next the reconciliation is happening ASAP
			// this situation will clear itself after AG has been started
			controller.setRequeueAfterIfNewIsShorter(fastUpdateInterval)

			return goerrors.Join(componentErrors...)
		}

		log.Info("could not reconcile app injection")

		componentErrors = append(componentErrors, err)
	}

	log.Info("start reconciling OneAgent")

	err = controller.oneAgentReconcilerBuilder(
		controller.client,
		controller.apiReader,
		dynatraceClient,
		dynakube,
		controller.tokens,
		controller.clusterID,
	).
		Reconcile(ctx)
	if err != nil {
		if errors.Is(err, oaconnectioninfo.NoOneAgentCommunicationHostsError) {
			// missing communication hosts is not an error per se, just make sure next the reconciliation is happening ASAP
			// this situation will clear itself after AG has been started
			controller.setRequeueAfterIfNewIsShorter(fastUpdateInterval)

			return goerrors.Join(componentErrors...)
		}

		log.Info("could not reconcile OneAgent")

		componentErrors = append(componentErrors, err)
	}

	return goerrors.Join(componentErrors...)
}

func (controller *Controller) createDynakubeMapper(ctx context.Context, dynakube *dynatracev1beta2.DynaKube) *mapper.DynakubeMapper {
	dkMapper := mapper.NewDynakubeMapper(ctx, controller.client, controller.apiReader, controller.operatorNamespace, dynakube)

	return &dkMapper
}
