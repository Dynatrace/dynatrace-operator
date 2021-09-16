package dynakube

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/capability"
	rcap "github.com/Dynatrace/dynatrace-operator/controllers/activegate/reconciler/capability"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube/status"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube/updates"
	"github.com/Dynatrace/dynatrace-operator/controllers/istio"
	"github.com/Dynatrace/dynatrace-operator/controllers/oneagent"
	"github.com/Dynatrace/dynatrace-operator/controllers/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/mapper"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultUpdateInterval = 5 * time.Minute
)

var log = logf.Log.WithName("controller_dynakube")

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
		logger:            logger.NewDTLogger(),
	}
}

func (r *ReconcileDynaKube) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1alpha1.DynaKube{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Complete(r)
}

func NewDynaKubeReconciler(c client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtcBuildFunc DynatraceClientFunc, logger logr.Logger, config *rest.Config) *ReconcileDynaKube {
	return &ReconcileDynaKube{
		client:            c,
		apiReader:         apiReader,
		scheme:            scheme,
		dtcBuildFunc:      dtcBuildFunc,
		logger:            logger,
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
	logger            logr.Logger
	config            *rest.Config
	operatorPodName   string
	operatorNamespace string
}

type DynatraceClientFunc func(rtc client.Client, instance *dynatracev1alpha1.DynaKube, secret *corev1.Secret) (dtclient.Client, error)

// Reconcile reads that state of the cluster for a DynaKube object and makes changes based on the state read
// and what is in the DynaKube.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDynaKube) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("namespace", request.Namespace, "name", request.Name)
	reqLogger.Info("Reconciling DynaKube")

	// Fetch the DynaKube instance
	instance := &dynatracev1alpha1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: request.NamespacedName.Name}}
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
	dkState := controllers.NewDynakubeState(reqLogger, instance)
	r.reconcileDynaKube(ctx, dkState, &dkMapper)

	if dkState.Err != nil {
		if dkState.Updated || instance.Status.SetPhaseOnError(dkState.Err) {
			if errClient := r.updateCR(ctx, reqLogger, instance); errClient != nil {
				return reconcile.Result{}, fmt.Errorf("failed to update CR after failure, original, %s, then: %w", dkState.Err, errClient)
			}
		}

		var serr dtclient.ServerError
		if ok := errors.As(dkState.Err, &serr); ok && serr.Code == http.StatusTooManyRequests {
			dkState.Log.Info("Request limit for Dynatrace API reached! Next reconcile in one minute")
			return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
		}

		return reconcile.Result{}, dkState.Err
	}

	if dkState.Updated {
		if err := r.updateCR(ctx, reqLogger, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{RequeueAfter: dkState.RequeueAfter}, nil
}

func (r *ReconcileDynaKube) reconcileDynaKube(ctx context.Context, dkState *controllers.DynakubeState, dkMapper *mapper.DynakubeMapper) {
	dtcReconciler := DynatraceClientReconciler{
		Client:              r.client,
		DynatraceClientFunc: r.dtcBuildFunc,
		UpdateAPIToken:      true,
		UpdatePaaSToken:     true,
	}
	dtc, upd, err := dtcReconciler.Reconcile(ctx, dkState.Instance)

	dkState.Update(upd, defaultUpdateInterval, "Token conditions updated")
	if dkState.Error(err) {
		return
	}

	err = status.SetDynakubeStatus(dkState.Instance, status.Options{
		Dtc:       dtc,
		ApiClient: r.apiReader,
	})
	if dkState.Error(err) {
		dkState.Log.Error(err, "could not set Dynakube status")
		return
	}

	// Fetch api token secret
	secret, err := r.getTokenSecret(ctx, dkState.Instance)
	if err != nil {
		dkState.Log.Error(err, "could not find token secret")
		return
	}

	if dkState.Instance.Spec.EnableIstio {
		if upd, err = istio.NewController(r.config, r.scheme).ReconcileIstio(dkState.Instance); err != nil {
			// If there are errors log them, but move on.
			dkState.Log.Info("Istio: failed to reconcile objects", "error", err)
		} else if upd {
			dkState.Update(true, 30*time.Second, "Istio: objects updated")
		}
	}

	err = dtpullsecret.
		NewReconciler(r.client, r.apiReader, r.scheme, dkState.Instance, dkState.Log, secret).
		Reconcile()
	if dkState.Error(err) {
		dkState.Log.Error(err, "could not reconcile Dynatrace pull secret")
		return
	}

	upd, err = updates.ReconcileVersions(ctx, dkState, r.client, dtversion.GetImageVersion)
	dkState.Update(upd, defaultUpdateInterval, "Found updates")
	dkState.Error(err)

	if !r.reconcileActiveGateCapabilities(dkState) {
		return
	}

	// Check Code Modules if CSI driver is needed
	err = dtcsi.ConfigureCSIDriver(
		r.client, r.scheme, r.operatorPodName, r.operatorNamespace, dkState, defaultUpdateInterval)
	if err != nil {
		dkState.Log.Error(err, "could not check code modules")
		return
	}

	if dkState.Instance.Spec.InfraMonitoring.Enabled {
		upd, err = oneagent.NewOneAgentReconciler(
			r.client, r.apiReader, r.scheme, dkState.Log, dkState.Instance, &dkState.Instance.Spec.InfraMonitoring.FullStackSpec, daemonset.InframonFeature,
		).Reconcile(ctx, dkState)
		if dkState.Error(err) || dkState.Update(upd, defaultUpdateInterval, "infra monitoring reconciled") {
			return
		}
	} else {
		ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dkState.Instance.Name + "-inframon", Namespace: dkState.Instance.Namespace}}
		if err := r.ensureDeleted(&ds); dkState.Error(err) {
			return
		}
	}

	if dkState.Instance.Spec.ClassicFullStack.Enabled {
		upd, err = oneagent.NewOneAgentReconciler(
			r.client, r.apiReader, r.scheme, dkState.Log, dkState.Instance, &dkState.Instance.Spec.ClassicFullStack, daemonset.ClassicFeature,
		).Reconcile(ctx, dkState)
		if dkState.Error(err) || dkState.Update(upd, defaultUpdateInterval, "classic fullstack reconciled") {
			return
		}
	} else {
		ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dkState.Instance.Name + "-classic", Namespace: dkState.Instance.Namespace}}
		if err := r.ensureDeleted(&ds); dkState.Error(err) {
			return
		}
	}

	if dkState.Instance.Spec.CodeModules.Enabled {
		if err := dkMapper.MapFromDynakube(); err != nil {
			dkState.Log.Error(err, "update of a map of namespaces failed")
			return
		}
		if dkState.Instance.Spec.InfraMonitoring.Enabled {
			upd, err := initgeneration.NewInitGenerator(r.client, r.apiReader, dkState.Instance.Namespace, r.logger).GenerateForDynakube(ctx, dkState.Instance)
			dkState.Update(upd, defaultUpdateInterval, "new init script created")
			if dkState.Error(err) {
				return
			}
		}
	}
}

func (r *ReconcileDynaKube) ensureDeleted(obj client.Object) error {
	if err := r.client.Delete(context.TODO(), obj); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *ReconcileDynaKube) reconcileActiveGateCapabilities(dkState *controllers.DynakubeState) bool {
	var caps = []capability.Capability{
		capability.NewKubeMonCapability(&dkState.Instance.Spec.KubernetesMonitoringSpec.CapabilityProperties, &dkState.Instance.Spec.ActiveGate),
		capability.NewRoutingCapability(&dkState.Instance.Spec.RoutingSpec.CapabilityProperties, &dkState.Instance.Spec.ActiveGate),
		capability.NewDataIngestCapability(&dkState.Instance.Spec.DataIngestSpec.CapabilityProperties, &dkState.Instance.Spec.ActiveGate),
	}

	for _, c := range caps {
		if c.GetProperties().Enabled {
			upd, err := rcap.NewReconciler(
				c, r.client, r.apiReader, r.scheme, dkState.Log, dkState.Instance, dtversion.GetImageVersion,
			).Reconcile()
			if dkState.Error(err) || dkState.Update(upd, defaultUpdateInterval, c.GetModuleName()+" reconciled") {
				return false
			}
		} else {
			sts := appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      capability.CalculateStatefulSetName(c, dkState.Instance.Name),
					Namespace: dkState.Instance.Namespace,
				},
			}
			if err := r.ensureDeleted(&sts); dkState.Error(err) {
				return false
			}

			if c.GetConfiguration().CreateService {
				svc := corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      rcap.BuildServiceName(dkState.Instance.Name, c.GetModuleName()),
						Namespace: dkState.Instance.Namespace,
					},
				}
				if err := r.ensureDeleted(&svc); dkState.Error(err) {
					return false
				}
			}
		}
	}

	return true
}

func (r *ReconcileDynaKube) getTokenSecret(ctx context.Context, instance *dynatracev1alpha1.DynaKube) (*corev1.Secret, error) {
	var secret corev1.Secret
	err := r.client.Get(ctx, client.ObjectKey{Name: instance.Tokens(), Namespace: instance.Namespace}, &secret)
	return &secret, errors.WithStack(err)
}

func (r *ReconcileDynaKube) updateCR(ctx context.Context, log logr.Logger, instance *dynatracev1alpha1.DynaKube) error {
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
