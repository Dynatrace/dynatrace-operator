package activegate

import (
	"context"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_dynakube")

// Add creates a new DynaKube Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileActiveGate{
		client:       mgr.GetClient(),
		apiReader:    mgr.GetAPIReader(),
		scheme:       mgr.GetScheme(),
		dtcBuildFunc: BuildDynatraceClient,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("activegate-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource DynaKube
	err = c.Watch(&source.Kind{Type: &dynatracev1alpha1.DynaKube{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner DynaKube
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dynatracev1alpha1.DynaKube{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileActiveGate implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileActiveGate{}

// ReconcileActiveGate reconciles a DynaKube object
type ReconcileActiveGate struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client       client.Client
	apiReader    client.Reader
	scheme       *runtime.Scheme
	dtcBuildFunc DynatraceClientFunc
}

type DynatraceClientFunc func(rtc client.Client, instance *dynatracev1alpha1.DynaKube, secret *corev1.Secret) (dtclient.Client, error)

// Reconcile reads that state of the cluster for a DynaKube object and makes changes based on the state read
// and what is in the DynaKube.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileActiveGate) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling DynaKube")

	// Fetch the DynaKube instance
	instance := &dynatracev1alpha1.DynaKube{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Fetch api token secret
	secret, err := r.getTokenSecret(instance)
	if err != nil {
		reqLogger.Error(err, "could not find token secret")
		return reconcile.Result{}, err
	}

	dtc, err := r.dtcBuildFunc(r.client, instance, secret)
	if err != nil {
		return reconcile.Result{}, err
	}

	//if !instance.Spec.KubernetesMonitoringSpec.Enabled {
	//	return builder.ReconcileAfterFiveMinutes(), nil
	//}
	if instance.Spec.KubernetesMonitoringSpec.Enabled {
		return kubemon.NewReconciler(
			r.client, r.apiReader, r.scheme, dtc, reqLogger, secret, instance,
		).Reconcile(request)
	}

	//if instance.Spec.KubernetesMonitoringSpec.Image == "" && instance.Spec.CustomPullSecret == "" {
	//	err = r.reconcilePullSecret(instance, reqLogger, dtc)
	//	if err != nil {
	//		return reconcile.Result{}, err
	//	}
	//}

	//customPropertiesConfigMap, err := r.manageCustomProperties(instance.Name, &instance.Spec.KubernetesMonitoringSpec)
	//if err != nil {
	//	reqLogger.Error(err, "error creating config map for custom properties")
	//	return reconcile.Result{}, err
	//}
	//if customPropertiesConfigMap != nil {
	//	if err := controllerutil.SetControllerReference(instance, customPropertiesConfigMap, r.scheme); err != nil {
	//		reqLogger.Error(err, "error setting controller reference for custom properties secret")
	//		return reconcile.Result{}, err
	//	}
	//}

	//desiredStatefulSet, err := r.createDesiredStatefulSet(instance, dtc)
	//if err != nil {
	//	reqLogger.Error(err, "error when creating desired stateful set")
	//	return reconcile.Result{}, err
	//}
	//
	//// Set DynaKube instance as the owner and controller
	//if err := controllerutil.SetControllerReference(instance, desiredStatefulSet, r.scheme); err != nil {
	//	reqLogger.Error(err, "error setting controller reference")
	//	return reconcile.Result{}, err
	//}
	//
	//actualStatefulSet := &appsv1.StatefulSet{}
	//reconcileResult, err := r.manageStatefulSet(reqLogger, instance, desiredStatefulSet, actualStatefulSet)
	//if reconcileResult != nil {
	//	return *reconcileResult, err
	//}

	//pods, err := r.findPods(instance)
	//if err != nil {
	//	return reconcile.Result{}, err
	//}
	//
	//err = r.setVersionLabel(pods)
	//if err != nil {
	//
	//	var statusError *k8serrors.StatusError
	//	if errors.As(err, &statusError) {
	//		// Since this happens early during deployment, pods might have been modified
	//		// In this case, retry silently
	//		return builder.ReconcileAfter(5 * time.Second), nil
	//	}
	//	// Otherwise, retry loudly
	//	return builder.ReconcileAfterFiveMinutes(), err
	//}

	//reconcileResult, err = r.updateService.UpdatePods(r, instance)
	//if err != nil {
	//	log.Error(err, "could not update pods")
	//}
	//if reconcileResult != nil {
	//	return *reconcileResult, err
	//}

	//if instance.Spec.KubernetesMonitoringSpec.KubernetesAPIEndpoint != "" {
	//	id, err := r.addToDashboard(dtc, instance)
	//	r.handleAddToDashboardResult(id, err, log)
	//}

	// Set version and last updated timestamp
	// Nothing to do - requeue after five minutes
	reqLogger.Info("Nothing to do: Instance is ready", "Namespace", instance.Namespace, "Name", instance.Name)
	return reconcile.Result{Requeue: true, RequeueAfter: 5 * time.Minute}, nil
}

func (r *ReconcileActiveGate) getTokenSecret(instance *dynatracev1alpha1.DynaKube) (*corev1.Secret, error) {
	var secret *corev1.Secret
	err := r.client.Get(context.TODO(), client.ObjectKey{Name: GetTokensName(instance), Namespace: instance.Namespace}, secret)
	return secret, err
}

//func (r *ReconcileActiveGate) findPods(instance *dynatracev1alpha1.DynaKube) ([]corev1.Pod, error) {
//	podList := &corev1.PodList{}
//	listOptions := []client.ListOption{
//		client.InNamespace(instance.GetNamespace()),
//		client.MatchingLabels(builder.BuildLabelsForQuery(instance.Name)),
//	}
//	err := r.client.List(context.TODO(), podList, listOptions...)
//	if err != nil {
//		return nil, err
//	}
//	return podList.Items, nil
//}

const (
	annotationTemplateHash = "internal.activegate.dynatrace.com/template-hash"
)
