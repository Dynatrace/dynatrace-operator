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
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtversion"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/status"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/updates"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/pkg/errors"
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

const (
	defaultUpdateInterval = 5 * time.Minute
)

func Add(mgr manager.Manager, _ string) error {
	return NewReconciler(mgr).SetupWithManager(mgr)
}

// NewReconciler returns a new ReconcileDynaKube
func NewReconciler(mgr manager.Manager) *ReconcileDynaKube {
	return &ReconcileDynaKube{
		client:            mgr.GetClient(),
		apiReader:         mgr.GetAPIReader(),
		scheme:            mgr.GetScheme(),
		dtcBuildFunc:      BuildDynatraceClient,
		config:            mgr.GetConfig(),
		operatorPodName:   os.Getenv("POD_NAME"),
		operatorNamespace: os.Getenv("POD_NAMESPACE"),
	}
}

func (r *ReconcileDynaKube) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1beta1.DynaKube{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Complete(r)
}

func NewDynaKubeReconciler(c client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtcBuildFunc DynatraceClientFunc, config *rest.Config) *ReconcileDynaKube {
	return &ReconcileDynaKube{
		client:            c,
		apiReader:         apiReader,
		scheme:            scheme,
		dtcBuildFunc:      dtcBuildFunc,
		config:            config,
		operatorPodName:   os.Getenv("POD_NAME"),
		operatorNamespace: os.Getenv("POD_NAMESPACE"),
	}
}

// blank assignment to verify that ReconcileDynaKube implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDynaKube{}

// ReconcileDynaKube reconciles a DynaKube object
type ReconcileDynaKube struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client            client.Client
	apiReader         client.Reader
	scheme            *runtime.Scheme
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
func (r *ReconcileDynaKube) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling DynaKube", "namespace", request.Namespace, "name", request.Name)

	// Fetch the DynaKube instance
	instance := &dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: request.NamespacedName.Name}}
	dkMapper := mapper.NewDynakubeMapper(ctx, r.client, r.apiReader, r.operatorNamespace, instance)
	err := r.client.Get(ctx, request.NamespacedName, instance)
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
	r.reconcileDynaKube(ctx, dkState, &dkMapper)

	if dkState.Err != nil {
		if !dkState.ValidTokens {
			return reconcile.Result{}, fmt.Errorf("paas or api token not valid")
		}
		if dkState.Updated || instance.Status.SetPhaseOnError(dkState.Err) {
			if errClient := r.updateCR(ctx, instance); errClient != nil {
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
		if err := r.updateCR(ctx, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{RequeueAfter: dkState.RequeueAfter}, nil
}

func (r *ReconcileDynaKube) reconcileDynaKube(ctx context.Context, dkState *status.DynakubeState, dkMapper *mapper.DynakubeMapper) {
	dtcReconciler := DynatraceClientReconciler{
		Client:              r.client,
		DynatraceClientFunc: r.dtcBuildFunc,
	}
	dtc, upd, err := dtcReconciler.Reconcile(ctx, dkState.Instance)

	dkState.Update(upd, defaultUpdateInterval, "Token conditions updated")
	if dkState.Error(err) {
		log.Error(err, "failed to check tokens")
		return
	}
	dkState.ValidTokens = true
	if !dtcReconciler.ValidTokens {
		dkState.ValidTokens = false
		log.Info("paas or api token not valid", "name", dkState.Instance.GetName())
		return
	}

	err = status.SetDynakubeStatus(dkState.Instance, status.Options{
		Dtc:       dtc,
		ApiClient: r.apiReader,
	})
	if dkState.Error(err) {
		log.Error(err, "could not set Dynakube status")
		return
	}

	if dkState.Instance.Spec.EnableIstio {
		if upd, err = istio.NewController(r.config, r.scheme).ReconcileIstio(dkState.Instance); err != nil {
			// If there are errors log them, but move on.
			log.Info("Istio: failed to reconcile objects", "error", err)
		} else if upd {
			dkState.Update(true, 30*time.Second, "Istio: objects updated")
		}
	}

	err = dtpullsecret.
		NewReconciler(r.client, r.apiReader, r.scheme, dkState.Instance, dtcReconciler.ApiToken, dtcReconciler.PaasToken).
		Reconcile()
	if dkState.Error(err) {
		log.Error(err, "could not reconcile Dynatrace pull secret")
		return
	}

	upd, err = updates.ReconcileVersions(ctx, dkState, r.client, dtversion.GetImageVersion)
	dkState.Update(upd, defaultUpdateInterval, "Found updates")
	dkState.Error(err)

	if !r.reconcileActiveGate(ctx, dkState, dtc) {
		return
	}
	if dkState.Instance.HostMonitoringMode() {
		upd, err = oneagent.NewOneAgentReconciler(
			r.client, r.apiReader, r.scheme, dkState.Instance, daemonset.HostMonitoringFeature,
		).Reconcile(ctx, dkState)
		if dkState.Error(err) || dkState.Update(upd, defaultUpdateInterval, "infra monitoring reconciled") {
			return
		}
	} else {
		ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dkState.Instance.Name + "-" + daemonset.HostMonitoringFeature, Namespace: dkState.Instance.Namespace}}
		if err := r.ensureDeleted(&ds); dkState.Error(err) {
			return
		}
	}

	if dkState.Instance.CloudNativeFullstackMode() {
		upd, err = oneagent.NewOneAgentReconciler(
			r.client, r.apiReader, r.scheme, dkState.Instance, daemonset.CloudNativeFeature,
		).Reconcile(ctx, dkState)
		if dkState.Error(err) || dkState.Update(upd, defaultUpdateInterval, "cloud native infra monitoring reconciled") {
			return
		}
	} else {
		ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dkState.Instance.Name + "-" + daemonset.CloudNativeFeature, Namespace: dkState.Instance.Namespace}}
		if err := r.ensureDeleted(&ds); dkState.Error(err) {
			return
		}
	}

	if dkState.Instance.ClassicFullStackMode() {
		upd, err = oneagent.NewOneAgentReconciler(
			r.client, r.apiReader, r.scheme, dkState.Instance, daemonset.ClassicFeature,
		).Reconcile(ctx, dkState)
		if dkState.Error(err) || dkState.Update(upd, defaultUpdateInterval, "classic fullstack reconciled") {
			return
		}
	} else {
		ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dkState.Instance.Name + "-" + daemonset.ClassicFeature, Namespace: dkState.Instance.Namespace}}
		if err := r.ensureDeleted(&ds); dkState.Error(err) {
			return
		}
	}

	endpointSecretGenerator := dtingestendpoint.NewEndpointSecretGenerator(r.client, r.apiReader, dkState.Instance.Namespace)
	if dkState.Instance.NeedAppInjection() {
		if err := dkMapper.MapFromDynakube(); err != nil {
			log.Error(err, "update of a map of namespaces failed")
		}
		upd, err := initgeneration.NewInitGenerator(r.client, r.apiReader, dkState.Instance.Namespace).GenerateForDynakube(ctx, dkState.Instance)
		if dkState.Error(err) || dkState.Update(upd, defaultUpdateInterval, "new init script created") {
			return
		}

		if !dkState.Instance.FeatureDisableMetadataEnrichment() {
			upd, err = endpointSecretGenerator.GenerateForDynakube(ctx, dkState.Instance)
			if dkState.Error(err) || dkState.Update(upd, defaultUpdateInterval, "new data-ingest endpoint secret created") {
				return
			}
		} else {
			err = endpointSecretGenerator.RemoveEndpointSecrets(ctx, dkState.Instance)
			if dkState.Error(err) {
				return
			}
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
}

func (r *ReconcileDynaKube) ensureDeleted(obj client.Object) error {
	if err := r.client.Delete(context.TODO(), obj); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *ReconcileDynaKube) reconcileActiveGate(ctx context.Context, dynakubeState *status.DynakubeState, dtc dtclient.Client) bool {
	if !r.reconcileActiveGateProxySecret(ctx, dynakubeState) {
		return false
	}
	return r.reconcileActiveGateCapabilities(dynakubeState, dtc)
}

func (r *ReconcileDynaKube) reconcileActiveGateProxySecret(ctx context.Context, dynakubeState *status.DynakubeState) bool {
	gen := agproxysecret.NewActiveGateProxySecretGenerator(r.client, r.apiReader, dynakubeState.Instance.Namespace, log)
	if dynakubeState.Instance.HasProxy() {
		upd, err := gen.GenerateForDynakube(ctx, dynakubeState.Instance)
		if dynakubeState.Error(err) || dynakubeState.Update(upd, defaultUpdateInterval, "new ActiveGate proxy secret created") {
			return false
		}
	} else {
		if err := gen.EnsureDeleted(ctx, dynakubeState.Instance); dynakubeState.Error(err) {
			return false
		}
	}
	return true
}

func (r *ReconcileDynaKube) reconcileActiveGateCapabilities(dynakubeState *status.DynakubeState, dtc dtclient.Client) bool {
	var caps = []capability.Capability{
		capability.NewKubeMonCapability(dynakubeState.Instance),
		capability.NewRoutingCapability(dynakubeState.Instance),
		capability.NewMultiCapability(dynakubeState.Instance),
	}

	for _, c := range caps {
		if c.Enabled() {
			upd, err := rcap.NewReconciler(
				c, r.client, r.apiReader, r.scheme, dynakubeState.Instance).Reconcile()
			if dynakubeState.Error(err) || dynakubeState.Update(upd, defaultUpdateInterval, c.ShortName()+" reconciled") {
				return false
			}
		} else {
			sts := appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      capability.CalculateStatefulSetName(c, dynakubeState.Instance.Name),
					Namespace: dynakubeState.Instance.Namespace,
				},
			}
			if err := r.ensureDeleted(&sts); dynakubeState.Error(err) {
				return false
			}

			if c.Config().CreateService {
				svc := corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      rcap.BuildServiceName(dynakubeState.Instance.Name, c.ShortName()),
						Namespace: dynakubeState.Instance.Namespace,
					},
				}
				if err := r.ensureDeleted(&svc); dynakubeState.Error(err) {
					return false
				}
			}
		}
	}

	//start automatic config creation
	if dynakubeState.Instance.Status.KubeSystemUUID != "" &&
		dynakubeState.Instance.FeatureAutomaticKubernetesApiMonitoring() &&
		dynakubeState.Instance.KubernetesMonitoringMode() {
		err := automaticapimonitoring.NewReconciler(dtc, dynakubeState.Instance.Name, dynakubeState.Instance.Status.KubeSystemUUID).
			Reconcile()
		if err != nil {
			log.Error(err, "could not create setting")
		}
	}

	return true
}

func (r *ReconcileDynaKube) updateCR(ctx context.Context, instance *dynatracev1beta1.DynaKube) error {
	instance.Status.UpdatedTimestamp = metav1.Now()
	err := r.client.Status().Update(ctx, instance)
	if err != nil && k8serrors.IsConflict(err) {
		// OneAgent reconciler already updates instance which leads to conflict here
		// Only print info in that event
		log.Info("could not update instance due to conflict")
		return nil
	}
	return errors.WithStack(err)
}
