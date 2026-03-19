package dynakube

import (
	"context"
	goerrors "errors"
	"os"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dynatracestatus "github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	tokenclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/apimonitoring"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceapi"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/injection"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/k8sentity"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring"
	logmondaemonset "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sevent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/system"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	fastRequeueInterval    = 1 * time.Minute
	defaultRequeueInterval = 15 * time.Minute

	controllerName = "dynakube-controller"
)

func Add(mgr manager.Manager, _ string) error {
	kubeSysUID, err := system.GetUID(context.Background(), mgr.GetAPIReader())
	if err != nil {
		return errors.WithStack(err)
	}

	return NewController(mgr, string(kubeSysUID)).SetupWithManager(mgr)
}

func NewController(mgr manager.Manager, clusterID string) *Controller {
	return NewDynaKubeController(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		mgr.GetEventRecorder(controllerName),
		mgr.GetConfig(),
		clusterID)
}

func NewDynaKubeController(kubeClient client.Client, apiReader client.Reader, eventRecorder events.EventRecorder, config *rest.Config, clusterID string) *Controller {
	return &Controller{
		client:                 kubeClient,
		apiReader:              apiReader,
		eventRecorder:          eventRecorder,
		config:                 config,
		operatorNamespace:      os.Getenv(k8senv.PodNamespace),
		clusterID:              clusterID,
		dynatraceClientBuilder: dynatraceclient.NewBuilder(apiReader),

		activeGateReconcilerBuilder: activegate.NewReconciler,
		oneAgentReconcilerBuilder:   oneagent.NewReconciler,
		injectionReconcilerBuilder:  injection.NewReconciler,

		apiMonitoringReconciler:      apimonitoring.NewReconciler(),
		extensionReconciler:          extension.NewReconciler(kubeClient, apiReader),
		kspmReconciler:               kspm.NewReconciler(kubeClient, apiReader),
		k8sEntityReconciler:          k8sentity.NewReconciler(),
		otelcReconciler:              otelc.NewReconciler(kubeClient, apiReader),
		proxyReconciler:              proxy.NewReconciler(kubeClient, apiReader),
		deploymentMetadataReconciler: deploymentmetadata.NewReconciler(kubeClient, apiReader, clusterID),
		istioReconciler:              istio.NewReconciler(kubeClient, apiReader),
		logMonitoringReconciler:      logmonitoring.NewReconciler(kubeClient, apiReader),
	}
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynakube.DynaKube{}).
		Named(controllerName).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(controller)
}

type apiMonitoringReconciler interface {
	Reconcile(ctx context.Context, dtc settings.APIClient, clusterLabel string, dk *dynakube.DynaKube) error
}

type istioReconciler interface {
	ReconcileAPIURL(ctx context.Context, dk *dynakube.DynaKube) error
}

type dynakubeReconciler interface {
	Reconcile(ctx context.Context, dk *dynakube.DynaKube) error
}

// dtSettingReconciler is a reconciler that uses the Dynatrace's Settings API during its reconcile.
type dtSettingReconciler interface {
	Reconcile(ctx context.Context, dtclient settings.APIClient, dk *dynakube.DynaKube) error
}

type logMonitoringReconciler interface {
	Reconcile(ctx context.Context, dtc dtclient.Client, dk *dynakube.DynaKube) error
}

// Controller reconciles a DynaKube object
type Controller struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the api-server
	client        client.Client
	apiReader     client.Reader
	eventRecorder events.EventRecorder

	apiMonitoringReconciler      apiMonitoringReconciler
	extensionReconciler          dynakubeReconciler
	k8sEntityReconciler          dtSettingReconciler
	kspmReconciler               dtSettingReconciler
	otelcReconciler              dynakubeReconciler
	proxyReconciler              dynakubeReconciler
	deploymentMetadataReconciler dynakubeReconciler
	istioReconciler              istioReconciler
	logMonitoringReconciler      logMonitoringReconciler

	dynatraceClientBuilder dynatraceclient.Builder
	config                 *rest.Config

	activeGateReconcilerBuilder activegate.ReconcilerBuilder
	oneAgentReconcilerBuilder   oneagent.ReconcilerBuilder
	injectionReconcilerBuilder  injection.ReconcilerBuilder

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

	dk, err := controller.getDynakubeOrCleanup(ctx, request.Name, request.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	} else if dk == nil {
		log.Info("reconciling DynaKube finished, no dynakube available", "namespace", request.Namespace, "name", request.Name, "result", "empty")

		return reconcile.Result{}, nil
	}

	isCrdLatestVersion, err := k8scrd.IsLatestVersion(ctx, controller.apiReader, k8scrd.DynaKubeName)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !isCrdLatestVersion {
		log.Debug("sending k8s event about CRD version mismatch")
		k8sevent.SendCRDVersionMismatch(controller.eventRecorder, dk)
	}

	oldStatus := *dk.Status.DeepCopy()
	controller.requeueAfter = defaultRequeueInterval
	err = controller.reconcileDynaKube(ctx, dk)
	result, err := controller.handleError(ctx, dk, err, oldStatus)

	return result, err
}

func (controller *Controller) getDynakubeOrCleanup(ctx context.Context, dkName, dkNamespace string) (*dynakube.DynaKube, error) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dkName,
			Namespace: dkNamespace,
		},
	}

	err := controller.apiReader.Get(ctx, client.ObjectKey{Name: dk.Name, Namespace: dk.Namespace}, dk)
	if k8serrors.IsNotFound(err) {
		namespaces, err := mapper.GetNamespacesForDynakube(ctx, controller.apiReader, dkName)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to list namespaces for dynakube %s", dkName)
		}

		return nil, controller.createDynakubeMapper(ctx, dk).UnmapFromDynaKube(namespaces)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return dk, nil
}

func (controller *Controller) handleError(
	ctx context.Context,
	dk *dynakube.DynaKube,
	reconcileErr error,
	oldStatus dynakube.DynaKubeStatus,
) (reconcile.Result, error) {
	switch {
	case dynatraceapi.IsUnreachable(reconcileErr):
		log.Info("the Dynatrace API server is unavailable or request limit reached! trying again in one minute",
			"errorCode", dynatraceapi.StatusCode(reconcileErr), "errorMessage", dynatraceapi.Message(reconcileErr))
		// should we set the phase to error ?
		return reconcile.Result{RequeueAfter: fastRequeueInterval}, nil

	case reconcileErr != nil:
		dk.Status.SetPhase(dynatracestatus.Error)

	default:
		dk.Status.SetPhase(controller.determineDynaKubePhase(ctx, dk))
	}

	isStatusDifferent, hashErr := hasher.IsDifferent(oldStatus, dk.Status)
	if hashErr != nil {
		reconcileErr = goerrors.Join(
			reconcileErr,
			errors.WithMessagef(hashErr, "failed to generate a hash for the DynaKube's status %s/%s", dk.Namespace, dk.Name),
		)
	} else if isStatusDifferent {
		log.Info("status changed, updating the DynaKube", "namespace", dk.Namespace, "name", dk.Name)

		if updateErr := dk.UpdateStatus(ctx, controller.client); updateErr != nil {
			reconcileErr = goerrors.Join(
				reconcileErr,
				errors.WithMessagef(updateErr, "failed to update the DynaKube's status %s/%s", dk.Namespace, dk.Name),
			)
		}
	}

	// needed so you don't see warning logs such as: "Warning: Reconciler returned both a non-zero result and a non-nil error. The result will always be ignored if the error is non-nil and the non-nil error causes requeuing with exponential backoff"
	if reconcileErr != nil {
		return reconcile.Result{}, reconcileErr
	}

	log.Info("finished DynaKube reconcile", "namespace", dk.Namespace, "name", dk.Name, "requeueAfter", controller.requeueAfter.String())

	return reconcile.Result{RequeueAfter: controller.requeueAfter}, nil
}

func (controller *Controller) setRequeueAfterIfNewIsShorter(requeueAfter time.Duration) {
	if controller.requeueAfter > requeueAfter {
		controller.requeueAfter = requeueAfter
	}
}

func (controller *Controller) reconcileDynaKube(ctx context.Context, dk *dynakube.DynaKube) error {
	err := controller.istioReconciler.ReconcileAPIURL(ctx, dk)
	if err != nil {
		return errors.WithMessage(err, "failed to reconcile istio objects for API url")
	}

	dynatraceClient, err := controller.setupTokensAndClient(ctx, dk)
	if err != nil {
		return err
	}

	dk.Status.KubeSystemUUID = controller.clusterID

	log.Info("start reconciling deployment meta data")

	err = controller.deploymentMetadataReconciler.Reconcile(ctx, dk)
	if err != nil {
		return err
	}

	if err := controller.proxyReconciler.Reconcile(ctx, dk); err != nil {
		log.Info("could not reconcile proxy resources")

		return err
	}

	return controller.reconcileComponents(ctx, dynatraceClient, dk)
}

func (controller *Controller) setupTokensAndClient(ctx context.Context, dk *dynakube.DynaKube) (dtclient.Client, error) {
	tokenReader := token.NewReader(controller.apiReader, dk)

	tokens, err := tokenReader.ReadTokens(ctx)
	if err != nil {
		controller.setConditionTokenError(dk, err)

		return nil, err
	}

	controller.tokens = tokens

	dynatraceClientBuilder := controller.dynatraceClientBuilder.
		SetDynakube(*dk).
		SetTokens(tokens)

	dynatraceClient, err := dynatraceClientBuilder.Build(ctx)
	if err != nil {
		controller.setConditionTokenError(dk, err)

		return nil, err
	}

	err = controller.verifyTokens(ctx, dynatraceClient, dk)
	if err != nil {
		controller.setConditionTokenError(dk, err)

		return nil, err
	}

	controller.setConditionTokenReady(dk, token.CheckForDataIngestToken(tokens))

	return dynatraceClient, nil
}

func (controller *Controller) reconcileComponents(ctx context.Context, dynatraceClient dtclient.Client, dk *dynakube.DynaKube) error {
	var componentErrors []error

	log.Info("start reconciling ActiveGate")

	err := controller.reconcileActiveGate(ctx, dk, dynatraceClient)
	if err != nil {
		log.Info("could not reconcile ActiveGate")

		componentErrors = append(componentErrors, err)
	}

	if err := controller.k8sEntityReconciler.Reconcile(ctx, dynatraceClient.AsV2().Settings, dk); err != nil {
		componentErrors = append(componentErrors, err)
	}

	if err := controller.extensionReconciler.Reconcile(ctx, dk); err != nil {
		log.Info("could not reconcile Extensions")

		componentErrors = append(componentErrors, err)
	}

	log.Info("start reconciling otel-collector")

	if err := controller.otelcReconciler.Reconcile(ctx, dk); err != nil {
		log.Info("could not reconcile otelc")

		componentErrors = append(componentErrors, err)
	}

	log.Info("start reconciling LogMonitoring")

	err = controller.logMonitoringReconciler.Reconcile(ctx, dynatraceClient, dk)
	if err != nil {
		if errors.Is(err, oaconnectioninfo.NoOneAgentCommunicationEndpointsError) || errors.Is(err, logmondaemonset.KubernetesSettingsNotAvailableError) {
			controller.setRequeueAfterIfNewIsShorter(fastRequeueInterval)

			return goerrors.Join(componentErrors...)
		}

		log.Info("could not reconcile LogMonitoring")

		componentErrors = append(componentErrors, err)
	}

	log.Info("start reconciling app injection")

	err = controller.injectionReconcilerBuilder(
		controller.client,
		controller.apiReader,
		dynatraceClient,
		dk,
	).Reconcile(ctx)
	if err != nil {
		if errors.Is(err, oaconnectioninfo.NoOneAgentCommunicationEndpointsError) {
			// missing communication endpoints is not an error per se, just make sure next the reconciliation is happening ASAP
			// this situation will clear itself after AG has been started
			controller.setRequeueAfterIfNewIsShorter(fastRequeueInterval)

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
		dk,
		controller.tokens,
		controller.clusterID,
	).
		Reconcile(ctx)
	if err != nil {
		if errors.Is(err, oaconnectioninfo.NoOneAgentCommunicationEndpointsError) {
			// missing communication endpoints is not an error per se, just make sure next the reconciliation is happening ASAP
			// this situation will clear itself after AG has been started
			controller.setRequeueAfterIfNewIsShorter(fastRequeueInterval)

			return goerrors.Join(componentErrors...)
		}

		log.Info("could not reconcile OneAgent")

		componentErrors = append(componentErrors, err)
	}

	if err := controller.kspmReconciler.Reconcile(ctx, dynatraceClient.AsV2().Settings, dk); err != nil {
		log.Info("could not reconcile kspm")

		componentErrors = append(componentErrors, err)
	}

	return goerrors.Join(componentErrors...)
}

func (controller *Controller) createDynakubeMapper(ctx context.Context, dk *dynakube.DynaKube) *mapper.DynakubeMapper {
	dkMapper := mapper.NewDynakubeMapper(ctx, controller.client, controller.apiReader, controller.operatorNamespace, dk)

	return &dkMapper
}

func (controller *Controller) verifyTokens(ctx context.Context, dynatraceClient dtclient.Client, dk *dynakube.DynaKube) error {
	err := controller.tokens.VerifyValues()
	if err != nil {
		return err
	}

	err = controller.verifyTokenScopes(ctx, dynatraceClient, dk)
	if err != nil {
		return err
	}

	return nil
}

func (controller *Controller) verifyTokenScopes(ctx context.Context, dynatraceClient dtclient.Client, dk *dynakube.DynaKube) error {
	if !dk.IsTokenScopeVerificationAllowed(timeprovider.New()) {
		log.Info(dynakube.GetCacheValidMessage(
			"token verification",
			dk.Status.DynatraceAPI.LastTokenScopeRequest,
			dk.APIRequestThreshold()))

		return lastErrorFromCondition(&dk.Status)
	}

	tokens := controller.tokens.AddFeatureScopesToTokens()

	optionalScopes, err := tokens.VerifyScopes(ctx, dynatraceClient, *dk)
	if err != nil {
		return err
	}

	log.Info("token verified")

	dk.Status.DynatraceAPI.LastTokenScopeRequest = metav1.Now()

	controller.updateOptionalScopesConditions(&dk.Status, optionalScopes)

	return nil
}

func (controller *Controller) updateOptionalScopesConditions(dkStatus *dynakube.DynaKubeStatus, optionalScopes map[string]bool) {
	for scope, conditionType := range tokenclient.OptionalScopes {
		available, ok := optionalScopes[scope]
		switch {
		case !ok: // no enabled feature uses the `scope` -> doesn't need to be in the status
			_ = meta.RemoveStatusCondition(&dkStatus.Conditions, conditionType)
		case available:
			k8sconditions.SetOptionalScopeAvailable(&dkStatus.Conditions, conditionType, scope+" optional scope available")
		case !available:
			k8sconditions.SetOptionalScopeMissing(&dkStatus.Conditions, conditionType, scope+" optional scope not available, some features may not work")
		}
	}
}

func lastErrorFromCondition(dkStatus *dynakube.DynaKubeStatus) error {
	oldCondition := meta.FindStatusCondition(dkStatus.Conditions, dynakube.TokenConditionType)
	if oldCondition != nil && oldCondition.Reason != dynakube.ReasonTokenReady {
		return errors.New(oldCondition.Message)
	}

	return nil
}
