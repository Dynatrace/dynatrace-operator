package dynakube

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/apimonitoring"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/status"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
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

const shortUpdateInterval = 30 * time.Second

func Add(mgr manager.Manager, _ string) error {
	return NewController(mgr).SetupWithManager(mgr)
}

// NewController returns a new ReconcileDynaKube
func NewController(mgr manager.Manager) *DynakubeController {
	return NewDynaKubeController(mgr.GetClient(), mgr.GetAPIReader(), mgr.GetScheme(), BuildDynatraceClient, mgr.GetConfig())
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

func (controller *DynakubeController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1beta1.DynaKube{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Complete(controller)
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
	instance, err := controller.getDynakubeOrUnmap(ctx, request.Name, request.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}
	if instance == nil {
		return reconcile.Result{}, nil
	}

	// A new mapper is initialized here as well as in getDynakubeOrUnmap because to solve these dependencies
	// the whole dynakube controller would have to be bulldozed
	dkMapper := mapper.NewDynakubeMapper(ctx, controller.client, controller.apiReader, controller.operatorNamespace, instance)
	dkState := status.NewDynakubeState(instance)

	updated := controller.reconcileIstio(instance)
	if updated {
		dkState.Update(true, "Istio: objects updated")
		dkState.RequeueAfter = shortUpdateInterval
	}

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

func (controller *DynakubeController) getDynakubeOrUnmap(ctx context.Context, name string, namespace string) (*dynatracev1beta1.DynaKube, error) {
	dynakube := dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	dkMapper := mapper.NewDynakubeMapper(ctx, controller.client, controller.apiReader, controller.operatorNamespace, &dynakube)
	err := errors.WithStack(controller.client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, &dynakube))

	if k8serrors.IsNotFound(err) {
		err = dkMapper.UnmapFromDynaKube()
		return nil, err
	} else if err != nil {
		return nil, err
	}

	return &dynakube, nil
}

func (controller *DynakubeController) reconcileIstio(dynakube *dynatracev1beta1.DynaKube) bool {
	var err error
	updated := false

	if dynakube.Spec.EnableIstio {
		updated, err = istio.NewIstioReconciler(controller.config, controller.scheme).ReconcileIstio(dynakube)
		if err != nil {
			// If there are errors log them, but move on.
			log.Info("Istio: failed to reconcile objects", "error", err)
		}
	}

	return updated
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

	err = dtpullsecret.
		NewReconciler(controller.client, controller.apiReader, controller.scheme, dkState.Instance, dtcReconciler.ApiToken, dtcReconciler.PaasToken).
		Reconcile()
	if dkState.Error(err) {
		log.Error(err, "could not reconcile Dynatrace pull secret")
		return
	}

	upd, err = version.ReconcileVersions(ctx, dkState, controller.apiReader, controller.fs, version.GetImageVersion)
	dkState.Update(upd, "Found updates")
	dkState.Error(err)

	err = controller.reconcileActiveGate(ctx, dkState, dtc)
	if dkState.Error(err) {
		return
	}

	err = controller.reconcileOneAgent(ctx, dkState)
	if err != nil {
		return
	}

	endpointSecretGenerator := dtingestendpoint.NewEndpointSecretGenerator(controller.client, controller.apiReader, dkState.Instance.Namespace)
	if dkState.Instance.NeedAppInjection() {
		if err = dkMapper.MapFromDynakube(); err != nil {
			log.Error(err, "update of a map of namespaces failed")
		}

		err = initgeneration.NewInitGenerator(controller.client, controller.apiReader, dkState.Instance.Namespace).GenerateForDynakube(ctx, dkState.Instance)
		if dkState.Error(err) {
			return
		}

		err = endpointSecretGenerator.GenerateForDynakube(ctx, dkState.Instance)
		if dkState.Error(err) {
			return
		}

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

func (controller *DynakubeController) reconcileOneAgent(ctx context.Context, dkState *status.DynakubeState) (err error) {
	if dkState.Instance.HostMonitoringMode() {
		upd, err := oneagent.NewOneAgentReconciler(
			controller.client, controller.apiReader, controller.scheme, dkState.Instance, daemonset.DeploymentTypeHostMonitoring,
		).Reconcile(ctx, dkState)
		if dkState.Error(err) {
			return nil
		}
		dkState.Update(upd, "host monitoring reconciled")
	} else if dkState.Instance.CloudNativeFullstackMode() {
		upd, err := oneagent.NewOneAgentReconciler(
			controller.client, controller.apiReader, controller.scheme, dkState.Instance, daemonset.DeploymentTypeCloudNative,
		).Reconcile(ctx, dkState)
		if dkState.Error(err) {
			return nil
		}
		dkState.Update(upd, "cloud native fullstack monitoring reconciled")
	} else if dkState.Instance.ClassicFullStackMode() {
		upd, err := oneagent.NewOneAgentReconciler(
			controller.client, controller.apiReader, controller.scheme, dkState.Instance, daemonset.DeploymentTypeFullStack,
		).Reconcile(ctx, dkState)
		if dkState.Error(err) {
			return nil
		}
		dkState.Update(upd, "classic fullstack reconciled")
	} else {
		controller.removeOneAgentDaemonSet(dkState)
	}
	return err
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
	if err := kubeobjects.Delete(context.TODO(), controller.client, &ds); dkState.Error(err) {
		return
	}
}

func (controller *DynakubeController) reconcileActiveGate(ctx context.Context, dynakubeState *status.DynakubeState, dtc dtclient.Client) error {
	reconciler := activegate.NewReconciler(ctx, controller.client, controller.apiReader, controller.scheme, dynakubeState.Instance, dtc)
	upd, err := reconciler.Reconcile()

	if err != nil {
		return errors.WithMessage(err, "failed to reconcile ActiveGate")
	}

	dynakubeState.Update(upd, "ActiveGate reconciled")
	controller.startApiMonitoring(dynakubeState, dtc)

	return nil
}

func (controller *DynakubeController) startApiMonitoring(dynakubeState *status.DynakubeState, dtc dtclient.Client) {
	if dynakubeState.Instance.Status.KubeSystemUUID != "" &&
		dynakubeState.Instance.FeatureAutomaticKubernetesApiMonitoring() &&
		dynakubeState.Instance.IsKubernetesMonitoringCapabilityEnabled() {

		clusterLabel := dynakubeState.Instance.FeatureAutomaticKubernetesApiMonitoringClusterName()
		if clusterLabel == "" {
			clusterLabel = dynakubeState.Instance.Name
		}

		err := apimonitoring.NewReconciler(dtc, clusterLabel, dynakubeState.Instance.Status.KubeSystemUUID).
			Reconcile()
		if err != nil {
			log.Error(err, "could not create setting")
		}
	}
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
