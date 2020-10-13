package activegate

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/dao"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
	agerrors "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/errors"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/parser"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_activegate")

// Add creates a new ActiveGate Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileActiveGate{
		client:        mgr.GetClient(),
		scheme:        mgr.GetScheme(),
		dtcBuildFunc:  builder.BuildDynatraceClient,
		updateService: &activeGateUpdateService{},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("activegate-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ActiveGate
	err = c.Watch(&source.Kind{Type: &dynatracev1alpha1.ActiveGate{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner ActiveGate
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dynatracev1alpha1.ActiveGate{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileActiveGate implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileActiveGate{}

// ReconcileActiveGate reconciles a ActiveGate object
type ReconcileActiveGate struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client        client.Client
	scheme        *runtime.Scheme
	dtcBuildFunc  DynatraceClientFunc
	updateService updateService
}

type DynatraceClientFunc func(rtc client.Client, instance *dynatracev1alpha1.ActiveGate, secret *corev1.Secret) (dtclient.Client, error)

// Reconcile reads that state of the cluster for a ActiveGate object and makes changes based on the state read
// and what is in the ActiveGate.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileActiveGate) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ActiveGate")

	// Fetch the ActiveGate instance
	instance := &dynatracev1alpha1.ActiveGate{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
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
	if err != nil || secret == nil {
		return agerrors.HandleSecretError(secret, err, reqLogger)
	}

	// Define a new Pod object
	log.Info("creating new pod definition from custom resource")

	uid, err := dao.FindKubeSystemUID(r.client)
	if err != nil {
		log.Error(err, "error getting uid from kube-system namespace")
		return reconcile.Result{}, err
	}
	pod := r.newPodForCR(instance, secret, uid)

	// Set ActiveGate instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		return r.createPod(pod)
	} else if err != nil {
		return reconcile.Result{}, err
	}

	reconcileResult, err := r.updateService.UpdatePods(r, found, instance, secret)
	if err != nil {
		log.Error(err, "could not update pods")
	}
	if reconcileResult != nil {
		return *reconcileResult, err
	}

	//Set version and last updated timestamp
	// Nothing to do - requeue after five minutes
	reqLogger.Info("Nothing to do: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return builder.ReconcileAfterFiveMinutes(), nil
}

func (r *ReconcileActiveGate) getTokenSecret(instance *dynatracev1alpha1.ActiveGate) (*corev1.Secret, error) {
	namespace := instance.GetNamespace()
	secret := &corev1.Secret{}
	err := r.client.Get(context.TODO(), client.ObjectKey{Name: parser.GetTokensName(instance), Namespace: namespace}, secret)
	if err != nil {
		log.Error(err, err.Error())
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return secret, nil
}

func (r *ReconcileActiveGate) updateInstanceStatus(pod *corev1.Pod, instance *dynatracev1alpha1.ActiveGate, secret *corev1.Secret) {
	dtc, err := r.dtcBuildFunc(r.client, instance, secret)
	if err != nil {
		log.Error(err, err.Error())
	}

	query := builder.BuildActiveGateQuery(instance, pod)
	activegates, err := dtc.QueryActiveGates(query)
	if err != nil {
		log.Error(err, "failed to query activegates")
	}
	if len(activegates) > 0 {
		log.Info(fmt.Sprintf("found %d activegate(s)", len(activegates)))
		log.Info("setting activegate version", "version", activegates[0].Version)
		instance.Status.Version = activegates[0].Version
		instance.Status.UpdatedTimestamp = metav1.Now()
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			log.Info("failed to updated instance status", "message", err.Error())
		}
	}

}

// newPodForCR returns a pod with the same name/namespace as the cr
func (r *ReconcileActiveGate) newPodForCR(instance *dynatracev1alpha1.ActiveGate, secret *corev1.Secret, kubeSystemUID types.UID) *corev1.Pod {
	dtc, err := r.dtcBuildFunc(r.client, instance, secret)
	if err != nil {
		log.Error(err, err.Error())
	}

	tenantInfo, err := dtc.GetTenantInfo()
	if err != nil {
		log.Error(err, err.Error())
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-pod",
			Namespace: instance.Namespace,
			Labels:    builder.BuildLabels(instance.GetName(), instance.Spec.Labels),
		},
		Spec: builder.BuildActiveGatePodSpecs(instance, tenantInfo, kubeSystemUID),
	}
}

func (r *ReconcileActiveGate) findPods(instance *dynatracev1alpha1.ActiveGate) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(builder.BuildLabelsForQuery(instance.Name)),
	}
	err := r.client.List(context.TODO(), podList, listOptions...)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

const (
	TimeUntilActive = 10 * time.Second
	UpdateInterval  = 5 * time.Minute
)
