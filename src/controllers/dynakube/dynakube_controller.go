package dynakube

import (
	"context"
	"net/http"
	"os"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/apimonitoring"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/status"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	errorUpdateInterval   = 1 * time.Minute
	changesUpdateInterval = 5 * time.Minute
	defaultUpdateInterval = 30 * time.Minute
)

func Add(mgr manager.Manager, _ string) error {
	kubeSysUID, err := kubesystem.GetUID(mgr.GetAPIReader())
	if err != nil {
		return errors.WithStack(err)
	}
	return NewController(mgr, string(kubeSysUID)).SetupWithManager(mgr)
}

// NewController returns a new ReconcileDynaKube
func NewController(mgr manager.Manager, clusterID string) *Controller {
	return NewDynaKubeController(mgr.GetClient(), mgr.GetAPIReader(), mgr.GetScheme(), mgr.GetConfig(), clusterID)
}

func NewDynaKubeController(kubeClient client.Client, apiReader client.Reader, scheme *runtime.Scheme, config *rest.Config, clusterID string) *Controller { //nolint:revive // maximum number of return results per function exceeded; max 3 but got 4
	return &Controller{
		client:                 kubeClient,
		apiReader:              apiReader,
		scheme:                 scheme,
		fs:                     afero.Afero{Fs: afero.NewOsFs()},
		dynatraceClientBuilder: dynatraceclient.NewBuilder(apiReader),
		config:                 config,
		operatorNamespace:      os.Getenv("POD_NAMESPACE"),
		clusterID:              clusterID,
	}
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1beta1.DynaKube{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Complete(controller)
}

// Controller reconciles a DynaKube object
type Controller struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the api-server
	client                 client.Client
	apiReader              client.Reader
	scheme                 *runtime.Scheme
	fs                     afero.Afero
	dynatraceClientBuilder dynatraceclient.Builder
	config                 *rest.Config
	operatorNamespace      string
	clusterID              string
}

// Reconcile reads that state of the cluster for a DynaKube object and makes changes based on the state read
// and what is in the DynaKube.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (controller *Controller) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling DynaKube", "namespace", request.Namespace, "name", request.Name)
	requeueAfter := defaultUpdateInterval

	dynakube, err := controller.getDynakubeOrUnmap(ctx, request.Name, request.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	} else if dynakube == nil {
		return reconcile.Result{}, nil
	}

	oldStatus := *dynakube.Status.DeepCopy()
	updated := controller.reconcileIstio(dynakube)
	if updated {
		log.Info("istio: objects updated")
	}

	err = controller.reconcileDynaKube(ctx, dynakube)

	if err != nil {
		requeueAfter = errorUpdateInterval

		var serverErr dtclient.ServerError
		isServerError := errors.As(err, &serverErr)
		if isServerError && serverErr.Code == http.StatusTooManyRequests {
			// should we set the phase to error ?
			log.Info("request limit for Dynatrace API reached! Next reconcile in one minute")
			return reconcile.Result{RequeueAfter: requeueAfter}, nil
		}
		dynakube.Status.SetPhase(dynatracev1beta1.Error)
	} else {
		dynakube.Status.SetPhase(controller.determineDynaKubePhase(dynakube))
	}

	isStatusDifferent, err := kubeobjects.IsDifferent(oldStatus, dynakube.Status)
	if err != nil {
		log.Error(err, "failed to generate hash for the status section")
	}
	if isStatusDifferent {
		log.Info("status changed, updating DynaKube")
		requeueAfter = changesUpdateInterval
		if errClient := controller.updateDynakubeStatus(ctx, dynakube); errClient != nil {
			return reconcile.Result{}, errors.WithMessagef(errClient, "failed to update DynaKube after failure, original error: %s", err)
		}
	}

	return reconcile.Result{RequeueAfter: requeueAfter}, err
}

func (controller *Controller) getDynakubeOrUnmap(ctx context.Context, dkName, dkNamespace string) (*dynatracev1beta1.DynaKube, error) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dkName,
			Namespace: dkNamespace,
		},
	}
	err := controller.apiReader.Get(ctx, client.ObjectKey{Name: dynakube.Name, Namespace: dynakube.Namespace}, dynakube)
	if k8serrors.IsNotFound(err) {
		return nil, controller.createDynakubeMapper(ctx, dynakube).UnmapFromDynaKube()
	} else if err != nil {
		return nil, errors.WithStack(err)
	}
	return dynakube, nil
}

func (controller *Controller) createDynakubeMapper(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) *mapper.DynakubeMapper {
	dkMapper := mapper.NewDynakubeMapper(ctx, controller.client, controller.apiReader, controller.operatorNamespace, dynakube)
	return &dkMapper
}

func (controller *Controller) reconcileIstio(dynakube *dynatracev1beta1.DynaKube) bool {
	var err error
	updated := false

	if dynakube.Spec.EnableIstio {
		updated, err = istio.NewReconciler(controller.config, controller.scheme).Reconcile(dynakube)
		if err != nil {
			// If there are errors log them, but move on.
			log.Info("istio: failed to reconcile objects", "error", err)
		}
	}

	return updated
}

func (controller *Controller) reconcileDynaKube(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	tokenReader := token.NewReader(controller.apiReader, dynakube)
	tokens, err := tokenReader.ReadTokens(ctx)

	if err != nil {
		controller.setConditionTokenError(dynakube, err)
		return err
	}

	dynatraceClientBuilder := controller.dynatraceClientBuilder.
		SetContext(ctx).
		SetDynakube(*dynakube).
		SetTokens(tokens)
	dynatraceClient, err := dynatraceClientBuilder.BuildWithTokenVerification(&dynakube.Status)

	if err != nil {
		controller.setConditionTokenError(dynakube, err)
		return err
	}

	controller.setConditionTokenReady(dynakube)
	err = status.SetDynakubeStatus(dynakube, status.Options{
		DtClient:  dynatraceClient,
		ApiReader: controller.apiReader,
	})
	if err != nil {
		log.Info("could not update Dynakube status")
		return err
	}

	err = dtpullsecret.
		NewReconciler(ctx, controller.client, controller.apiReader, controller.scheme, dynakube, tokens).
		Reconcile()
	if err != nil {
		log.Info("could not reconcile Dynatrace pull secret")
		return err
	}

	err = connectioninfo.NewReconciler(ctx, controller.client, controller.apiReader, dynakube, dynatraceClient).Reconcile()
	if err != nil {
		return err
	}

	err = deploymentmetadata.NewReconciler(ctx, controller.client, controller.apiReader, *dynakube, controller.clusterID).Reconcile()
	if err != nil {
		return err
	}

	versionReconciler := version.Reconciler{
		Dynakube:        dynakube,
		ApiReader:       controller.apiReader,
		Fs:              controller.fs,
		VersionProvider: version.GetImageVersion,
		TimeProvider:    kubeobjects.NewTimeProvider(),
	}
	err = versionReconciler.Reconcile(ctx)
	if err != nil {
		log.Info("could not reconcile component versions")
		return err
	}

	err = controller.reconcileActiveGate(ctx, dynakube, dynatraceClient)
	if err != nil {
		log.Info("could not reconcile ActiveGate")
		return err
	}

	err = controller.reconcileOneAgent(ctx, dynakube)
	if err != nil {
		log.Info("could not reconcile OneAgent")
		return err
	}

	err = controller.reconcileAppInjection(ctx, dynakube)
	if err != nil {
		log.Info("could not reconcile app injection")
		return err
	}

	return nil
}

func (controller *Controller) reconcileAppInjection(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	if dynakube.NeedAppInjection() {
		return controller.setupAppInjection(ctx, dynakube)
	}

	return controller.removeAppInjection(ctx, dynakube)
}

func (controller *Controller) setupAppInjection(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (err error) {
	endpointSecretGenerator := dtingestendpoint.NewEndpointSecretGenerator(controller.client, controller.apiReader, dynakube.Namespace)
	dkMapper := controller.createDynakubeMapper(ctx, dynakube)

	if err = dkMapper.MapFromDynakube(); err != nil {
		log.Info("update of a map of namespaces failed")
		return err
	}

	err = initgeneration.NewInitGenerator(controller.client, controller.apiReader, dynakube.Namespace).GenerateForDynakube(ctx, dynakube)
	if err != nil {
		log.Info("failed to generate init secret")
		return err
	}

	err = endpointSecretGenerator.GenerateForDynakube(ctx, dynakube)
	if err != nil {
		log.Info("failed to generate data-ingest secret")
		return err
	}

	if dynakube.ApplicationMonitoringMode() {
		dynakube.Status.SetPhase(dynatracev1beta1.Running)
	}

	log.Info("app injection reconciled")
	return nil
}

func (controller *Controller) removeAppInjection(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (err error) {
	endpointSecretGenerator := dtingestendpoint.NewEndpointSecretGenerator(controller.client, controller.apiReader, dynakube.Namespace)
	dkMapper := controller.createDynakubeMapper(ctx, dynakube)

	if err := dkMapper.UnmapFromDynaKube(); err != nil {
		log.Info("could not unmap dynakube from namespace")
		return err
	}
	err = endpointSecretGenerator.RemoveEndpointSecrets(ctx, dynakube)
	if err != nil {
		log.Info("could not remove data-ingest secret")
		return err
	}
	return nil
}

func (controller *Controller) reconcileOneAgent(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	if !dynakube.NeedsOneAgent() {
		return controller.removeOneAgentDaemonSet(ctx, dynakube)
	}

	return oneagent.NewOneAgentReconciler(
		controller.client, controller.apiReader, controller.scheme, controller.clusterID,
	).Reconcile(ctx, dynakube)
}

func (controller *Controller) removeOneAgentDaemonSet(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	oneAgentDaemonSet := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dynakube.OneAgentDaemonsetName(), Namespace: dynakube.Namespace}}
	return kubeobjects.Delete(ctx, controller.client, &oneAgentDaemonSet)
}

func (controller *Controller) reconcileActiveGate(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) error {
	reconciler := activegate.NewReconciler(ctx, controller.client, controller.apiReader, controller.scheme, dynakube, dtc)
	err := reconciler.Reconcile()

	if err != nil {
		return errors.WithMessage(err, "failed to reconcile ActiveGate")
	}
	controller.setupAutomaticApiMonitoring(dynakube, dtc)

	return nil
}

func (controller *Controller) setupAutomaticApiMonitoring(dynakube *dynatracev1beta1.DynaKube, dtc dtclient.Client) {
	if dynakube.Status.KubeSystemUUID != "" &&
		dynakube.FeatureAutomaticKubernetesApiMonitoring() &&
		dynakube.IsKubernetesMonitoringActiveGateEnabled() {
		clusterLabel := dynakube.FeatureAutomaticKubernetesApiMonitoringClusterName()
		if clusterLabel == "" {
			clusterLabel = dynakube.Name
		}

		err := apimonitoring.NewReconciler(dtc, clusterLabel, dynakube.Status.KubeSystemUUID).
			Reconcile()
		if err != nil {
			log.Error(err, "could not create setting")
		}
	}
}

func (controller *Controller) updateDynakubeStatus(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	dynakube.Status.UpdatedTimestamp = metav1.Now()
	err := controller.client.Status().Update(ctx, dynakube)
	if err != nil && k8serrors.IsConflict(err) {
		log.Info("could not update dynakube due to conflict", "name", dynakube.Name)
		return nil
	}
	return errors.WithStack(err)
}
