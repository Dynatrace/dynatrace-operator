package dynakube

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/agproxysecret"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/reconciler/automaticapimonitoring"
	rcap "github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/reconciler/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/secrets"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/status"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const shortUpdateInterval = 30 * time.Second

func Add(mgr manager.Manager, _ string) error {
	return NewController(mgr).SetupWithManager(mgr)
}

// NewController returns a new ReconcileDynaKube
func NewController(mgr manager.Manager) *DynakubeController {
	return &DynakubeController{
		client:            mgr.GetClient(),
		apiReader:         mgr.GetAPIReader(),
		scheme:            mgr.GetScheme(),
		dtcBuildFunc:      BuildDynatraceClient,
		config:            mgr.GetConfig(),
		operatorPodName:   os.Getenv("POD_NAME"),
		operatorNamespace: os.Getenv("POD_NAMESPACE"),
	}
}

func (controller *DynakubeController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1beta1.DynaKube{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Complete(controller)
}

func NewDynaKubeController(c client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtcBuildFunc DynatraceClientFunc, config *rest.Config) *DynakubeController {
	return &DynakubeController{
		client:            c,
		apiReader:         apiReader,
		scheme:            scheme,
		fs:                afero.Afero{Fs: afero.NewOsFs()},
		dtcBuildFunc:      dtcBuildFunc,
		config:            config,
		operatorPodName:   os.Getenv("POD_NAME"),
		operatorNamespace: os.Getenv("POD_NAMESPACE"),
	}
}

// DynakubeController reconciles a DynaKube object
type DynakubeController struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client            client.Client
	apiReader         client.Reader
	scheme            *runtime.Scheme
	fs                afero.Afero
	dtcBuildFunc      DynatraceClientFunc
	config            *rest.Config
	operatorPodName   string
	operatorNamespace string
}

type DynatraceClientFunc func(properties DynatraceClientProperties) (dtclient.Client, error)

// Reconcile reads that state of the cluster for a DynaKube object and makes changes based on the state read
// and what is in the DynaKube.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (controller *DynakubeController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling DynaKube", "namespace", request.Namespace, "name", request.Name)

	// Fetch the DynaKube instance
	instance := &dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: request.NamespacedName.Name}}
	dkMapper := mapper.NewDynakubeMapper(ctx, controller.client, controller.apiReader, controller.operatorNamespace, instance)
	err := controller.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			if err := dkMapper.UnmapFromDynaKube(); err != nil {
				return reconcile.Result{}, err
			}
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	dkState := status.NewDynakubeState(instance)
	controller.reconcileDynaKube(ctx, dkState, &dkMapper)

	if dkState.Err != nil {
		if !dkState.ValidTokens {
			instance.Status.SetPhase(dynatracev1beta1.Error)
			_ = controller.updateCR(ctx, instance)
			return reconcile.Result{RequeueAfter: dkState.RequeueAfter}, nil
		}
		if dkState.Updated || instance.Status.SetPhaseOnError(dkState.Err) {
			if errClient := controller.updateCR(ctx, instance); errClient != nil {
				return reconcile.Result{}, fmt.Errorf("failed to update CR after failure, original, %s, then: %w", dkState.Err, errClient)
			}
		}

		var serr dtclient.ServerError
		if ok := errors.As(dkState.Err, &serr); ok && serr.Code == http.StatusTooManyRequests {
			log.Info("request limit for Dynatrace API reached! Next reconcile in one minute")
			return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
		}

		return reconcile.Result{}, dkState.Err
	}

	if dkState.Updated {
		if err := controller.updateCR(ctx, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{RequeueAfter: dkState.RequeueAfter}, nil
}

func (controller *DynakubeController) reconcileDynaKube(ctx context.Context, dkState *status.DynakubeState, dkMapper *mapper.DynakubeMapper) {
	dtcReconciler := DynatraceClientReconciler{
		Client:              controller.client,
		DynatraceClientFunc: controller.dtcBuildFunc,
	}
	dtc, upd, err := dtcReconciler.Reconcile(ctx, dkState.Instance)

	dkState.Update(upd, "token conditions updated")
	if dkState.Error(err) {
		log.Error(err, "failed to check tokens")
		return
	}
	dkState.ValidTokens = true
	if !dtcReconciler.ValidTokens {
		dkState.ValidTokens = false
		dkState.Update(true, "tokens not valid")
		return
	}

	err = status.SetDynakubeStatus(dkState.Instance, status.Options{
		Dtc:       dtc,
		ApiClient: controller.apiReader,
	})
	if dkState.Error(err) {
		log.Error(err, "could not set Dynakube status")
		return
	}

	if dkState.Instance.Spec.EnableIstio {
		if upd, err = istio.NewIstioReconciler(controller.config, controller.scheme).ReconcileIstio(dkState.Instance); err != nil {
			// If there are errors log them, but move on.
			log.Info("Istio: failed to reconcile objects", "error", err)
		} else if upd {
			dkState.Update(true, "Istio: objects updated")
			dkState.RequeueAfter = shortUpdateInterval
			return
		}
	}

	err = dtpullsecret.
		NewReconciler(controller.client, controller.apiReader, controller.scheme, dkState.Instance, dtcReconciler.ApiToken, dtcReconciler.PaasToken).
		Reconcile()
	if dkState.Error(err) {
		log.Error(err, "could not reconcile Dynatrace pull secret")
		return
	}

	if !dkState.Instance.FeatureDisableActivegateRawImage() && dkState.Instance.NeedsActiveGate() {
		err = secrets.NewTenantSecretReconciler(controller.client, controller.apiReader, controller.scheme, dkState.Instance, dtcReconciler.ApiToken, dtc).
			Reconcile()
		if dkState.Error(err) {
			log.Error(err, "could not reconcile Dynatrace ActiveGate Tenant secrets")
			return
		}
	}

	if dkState.Instance.UseActiveGateAuthToken() {
		err = secrets.NewAuthTokenReconciler(controller.client, controller.apiReader, controller.scheme, dkState.Instance, dtcReconciler.ApiToken, dtc).
			Reconcile()
		if dkState.Error(err) {
			log.Error(err, "could not reconcile Dynatrace ActiveGateAuthToken secrets")
			return
		}
	}

	upd, err = version.ReconcileVersions(ctx, dkState, controller.apiReader, controller.fs, version.GetImageVersion)
	dkState.Update(upd, "Found updates")
	dkState.Error(err)

	if !controller.reconcileActiveGate(ctx, dkState, dtc) {
		return
	}

	if dkState.Instance.HostMonitoringMode() {
		upd, err = oneagent.NewOneAgentReconciler(
			controller.client, controller.apiReader, controller.scheme, dkState.Instance, daemonset.DeploymentTypeHostMonitoring,
		).Reconcile(ctx, dkState)
		if dkState.Error(err) {
			return
		}
		dkState.Update(upd, "host monitoring reconciled")
	} else if dkState.Instance.CloudNativeFullstackMode() {
		upd, err = oneagent.NewOneAgentReconciler(
			controller.client, controller.apiReader, controller.scheme, dkState.Instance, daemonset.DeploymentTypeCloudNative,
		).Reconcile(ctx, dkState)
		if dkState.Error(err) {
			return
		}
		dkState.Update(upd, "cloud native fullstack monitoring reconciled")
	} else if dkState.Instance.ClassicFullStackMode() {
		upd, err = oneagent.NewOneAgentReconciler(
			controller.client, controller.apiReader, controller.scheme, dkState.Instance, daemonset.DeploymentTypeFullStack,
		).Reconcile(ctx, dkState)
		if dkState.Error(err) {
			return
		}
		dkState.Update(upd, "classic fullstack reconciled")
	} else {
		controller.removeOneAgentDaemonSet(dkState)
	}

	endpointSecretGenerator := dtingestendpoint.NewEndpointSecretGenerator(controller.client, controller.apiReader, dkState.Instance.Namespace)
	if dkState.Instance.NeedAppInjection() {
		if err := dkMapper.MapFromDynakube(); err != nil {
			log.Error(err, "update of a map of namespaces failed")
		}
		upd, err := initgeneration.NewInitGenerator(controller.client, controller.apiReader, dkState.Instance.Namespace).GenerateForDynakube(ctx, dkState.Instance)
		if dkState.Error(err) {
			return
		}
		dkState.Update(upd, "new init secret created")

		upd, err = endpointSecretGenerator.GenerateForDynakube(ctx, dkState.Instance)
		if dkState.Error(err) {
			return
		}
		dkState.Update(upd, "new data-ingest endpoint secret created")

		if dkState.Instance.ApplicationMonitoringMode() {
			dkState.Instance.Status.SetPhase(dynatracev1beta1.Running)
			dkState.Update(upd, "application monitoring reconciled")
		}
	} else {
		if err := dkMapper.UnmapFromDynaKube(); err != nil {
			log.Error(err, "could not unmap dynakube from namespace")
			return
		}
		err = endpointSecretGenerator.RemoveEndpointSecrets(ctx, dkState.Instance)
		if dkState.Error(err) {
			return
		}
	}

	upd = controller.determineDynaKubePhase(dkState.Instance)
	dkState.Update(upd, "dynakube phase changed")
}

func updatePhaseIfChanged(instance *dynatracev1beta1.DynaKube, newPhase dynatracev1beta1.DynaKubePhaseType) bool {
	if instance.Status.Phase == newPhase {
		return false
	}
	instance.Status.Phase = newPhase
	return true
}

func (controller *DynakubeController) removeOneAgentDaemonSet(dkState *status.DynakubeState) {
	ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dkState.Instance.OneAgentDaemonsetName(), Namespace: dkState.Instance.Namespace}}
	if err := controller.ensureDeleted(&ds); dkState.Error(err) {
		return
	}
}

func (controller *DynakubeController) ensureDeleted(obj client.Object) error {
	if err := controller.client.Delete(context.TODO(), obj); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (controller *DynakubeController) reconcileActiveGate(ctx context.Context, dynakubeState *status.DynakubeState, dtc dtclient.Client) bool {
	if !controller.reconcileActiveGateProxySecret(ctx, dynakubeState) {
		return false
	}
	return controller.reconcileActiveGateCapabilities(dynakubeState, dtc)
}

func (controller *DynakubeController) reconcileActiveGateProxySecret(ctx context.Context, dynakubeState *status.DynakubeState) bool {
	gen := agproxysecret.NewActiveGateProxySecretGenerator(controller.client, controller.apiReader, dynakubeState.Instance.Namespace, log)
	if dynakubeState.Instance.NeedsActiveGateProxy() {
		upd, err := gen.GenerateForDynakube(ctx, dynakubeState.Instance)
		if dynakubeState.Error(err) || dynakubeState.Update(upd, "new ActiveGate proxy secret created") {
			return false
		}
	} else {
		if err := gen.EnsureDeleted(ctx, dynakubeState.Instance); dynakubeState.Error(err) {
			return false
		}
	}
	return true
}

func generateActiveGateCapabilities(instance *dynatracev1beta1.DynaKube) []capability.Capability {
	return []capability.Capability{
		capability.NewKubeMonCapability(instance),
		capability.NewRoutingCapability(instance),
		capability.NewMultiCapability(instance),
	}
}

func (controller *DynakubeController) reconcileActiveGateCapabilities(dynakubeState *status.DynakubeState, dtc dtclient.Client) bool {
	var caps = generateActiveGateCapabilities(dynakubeState.Instance)

	for _, c := range caps {
		if c.Enabled() {
			upd, err := rcap.NewReconciler(
				c, controller.client, controller.apiReader, controller.scheme, dynakubeState.Instance).Reconcile()
			if dynakubeState.Error(err) {
				return false
			}
			dynakubeState.Update(upd, c.ShortName()+" reconciled")
		} else {
			sts := appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      capability.CalculateStatefulSetName(c, dynakubeState.Instance.Name),
					Namespace: dynakubeState.Instance.Namespace,
				},
			}
			if err := controller.ensureDeleted(&sts); dynakubeState.Error(err) {
				return false
			}

			if c.ShouldCreateService() {
				svc := corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      rcap.BuildServiceName(dynakubeState.Instance.Name, c.ShortName()),
						Namespace: dynakubeState.Instance.Namespace,
					},
				}
				if err := controller.ensureDeleted(&svc); dynakubeState.Error(err) {
					return false
				}
			}
		}
	}

	//start automatic config creation
	if dynakubeState.Instance.Status.KubeSystemUUID != "" &&
		dynakubeState.Instance.FeatureAutomaticKubernetesApiMonitoring() &&
		dynakubeState.Instance.KubernetesMonitoringMode() {

		clusterLabel := dynakubeState.Instance.FeatureAutomaticKubernetesApiMonitoringClusterName()
		if clusterLabel == "" {
			clusterLabel = dynakubeState.Instance.Name
		}

		err := automaticapimonitoring.NewReconciler(dtc, clusterLabel, dynakubeState.Instance.Status.KubeSystemUUID).
			Reconcile()
		if err != nil {
			log.Error(err, "could not create setting")
		}
	}

	return true
}

func (controller *DynakubeController) updateCR(ctx context.Context, instance *dynatracev1beta1.DynaKube) error {
	instance.Status.UpdatedTimestamp = metav1.Now()
	err := controller.client.Status().Update(ctx, instance)
	if err != nil && k8serrors.IsConflict(err) {
		// OneAgent reconciler already updates instance which leads to conflict here
		// Only print info in that event
		log.Info("could not update instance due to conflict")
		return nil
	}
	return errors.WithStack(err)
}
