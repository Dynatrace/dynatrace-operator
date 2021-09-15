package oneagent

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/controllers/oneagent/daemonset"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultUpdateInterval = 5 * time.Minute
	updateEnvVar          = "ONEAGENT_OPERATOR_UPDATE_INTERVAL"
)

// NewOneAgentReconciler initializes a new ReconcileOneAgent instance
func NewOneAgentReconciler(
	client client.Client,
	apiReader client.Reader,
	scheme *runtime.Scheme,
	logger logr.Logger,
	instance *dynatracev1alpha1.DynaKube,
	fullStack *dynatracev1alpha1.FullStackSpec,
	feature string) *ReconcileOneAgent {
	return &ReconcileOneAgent{
		client:    client,
		apiReader: apiReader,
		scheme:    scheme,
		logger:    logger,
		instance:  instance,
		fullStack: fullStack,
		feature:   feature,
	}
}

type ReconcileOneAgent struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	apiReader client.Reader
	scheme    *runtime.Scheme
	logger    logr.Logger
	instance  *dynatracev1alpha1.DynaKube
	fullStack *dynatracev1alpha1.FullStackSpec
	feature   string
}

// Reconcile reads that state of the cluster for a OneAgent object and makes changes based on the state read
// and what is in the OneAgent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileOneAgent) Reconcile(ctx context.Context, rec *controllers.DynakubeState) (bool, error) {
	r.logger.Info("Reconciling OneAgent")
	rec.Update(setUseImmutableImageStatus(r.instance, r.fullStack), 5*time.Minute, "UseImmutableImage changed")

	upd, err := r.reconcileRollout(rec)
	if err != nil {
		return false, err
	} else if upd {
		r.logger.Info("Rollout reconciled")
	}

	updInterval := defaultUpdateInterval
	if val := os.Getenv(updateEnvVar); val != "" {
		x, err := strconv.Atoi(val)
		if err != nil {
			r.logger.Info("Conversion of ONEAGENT_OPERATOR_UPDATE_INTERVAL failed")
		} else {
			updInterval = time.Duration(x) * time.Minute
		}
	}

	if rec.IsOutdated(r.instance.Status.OneAgent.LastHostsRequestTimestamp, updInterval) || r.oneAgentsSpinningUp() {
		r.instance.Status.OneAgent.LastHostsRequestTimestamp = rec.Now.DeepCopy()
		rec.Update(true, 5*time.Minute, "updated last host request time stamp")

		upd, err = r.reconcileInstanceStatuses(ctx, r.logger, r.instance)
		rec.Update(upd, 5*time.Minute, "Instance statuses reconciled")
		if rec.Error(err) {
			return false, err
		}
	}

	// Finally we have to determine the correct non error phase
	_, err = r.determineDynaKubePhase(r.instance)
	rec.Error(err)

	return upd, nil
}

func (r *ReconcileOneAgent) oneAgentsSpinningUp() bool {
	if !r.instance.Spec.InfraMonitoring.Enabled {
		return false
	}
	if r.instance.Status.OneAgent.Instances == nil {
		return true
	}
	pods, _, err := r.getPods(context.TODO(), r.instance, r.feature)
	if err != nil {
		r.logger.Error(err, "error while checking if onaAgent pods are spinning up")
	} else if len(pods) > len(r.instance.Status.OneAgent.Instances) {
		return true
	}
	return false
}

// validate sanity checks if essential fields in the custom resource are available
//
// Return an error in the following conditions
// - APIURL empty
func validate(cr *dynatracev1alpha1.DynaKube) error {
	var msg []string
	if cr.Spec.APIURL == "" {
		msg = append(msg, ".spec.apiUrl is missing")
	}
	if len(msg) > 0 {
		return errors.New(strings.Join(msg, ", "))
	}
	return nil
}

func (r *ReconcileOneAgent) reconcileRollout(dkState *controllers.DynakubeState) (bool, error) {
	updateCR := false

	// Define a new DaemonSet object
	dsDesired, err := r.getDesiredDaemonSet(dkState)
	if err != nil {
		dkState.Log.Info("Failed to get desired daemonset")
		return false, err
	}

	// Set OneAgent instance as the owner and controller
	if err := controllerutil.SetControllerReference(dkState.Instance, dsDesired, r.scheme); err != nil {
		return false, err
	}

	updateCR, err = kubeobjects.CreateOrUpdateDaemonSet(r.client, r.logger, dsDesired)
	if err != nil {
		return updateCR, err
	}

	if dkState.Instance.Status.Tokens != dkState.Instance.Tokens() {
		dkState.Instance.Status.Tokens = dkState.Instance.Tokens()
		updateCR = true
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) getDesiredDaemonSet(dkState *controllers.DynakubeState) (*appsv1.DaemonSet, error) {
	kubeSysUID, err := kubesystem.GetUID(r.apiReader)
	if err != nil {
		return nil, err
	}

	dsDesired, err := r.newDaemonSetForCR(dkState, string(kubeSysUID))
	if err != nil {
		return nil, err
	}
	return dsDesired, nil
}

func (r *ReconcileOneAgent) getPods(ctx context.Context, instance *dynatracev1alpha1.DynaKube, feature string) ([]corev1.Pod, []client.ListOption, error) {
	podList := &corev1.PodList{}
	listOps := []client.ListOption{
		client.InNamespace((*instance).GetNamespace()),
		client.MatchingLabels(buildLabels(instance.Name, feature)),
	}
	err := r.client.List(ctx, podList, listOps...)
	return podList.Items, listOps, err
}

func (r *ReconcileOneAgent) newDaemonSetForCR(dkState *controllers.DynakubeState, clusterID string) (*appsv1.DaemonSet, error) {
	var ds *appsv1.DaemonSet
	var err error

	if r.feature == daemonset.ClassicFeature {
		ds, err = daemonset.NewClassicFullStack(dkState.Instance, dkState.Log, clusterID).BuildDaemonSet()
	} else {
		ds, err = daemonset.NewInfraMonitoring(dkState.Instance, dkState.Log, clusterID).BuildDaemonSet()
	}
	if err != nil {
		return nil, err
	}

	dsHash, err := kubeobjects.GenerateHash(ds)
	if err != nil {
		return nil, err
	}
	ds.Annotations[kubeobjects.AnnotationHash] = dsHash

	return ds, nil
}

func (r *ReconcileOneAgent) reconcileInstanceStatuses(ctx context.Context, logger logr.Logger, instance *dynatracev1alpha1.DynaKube) (bool, error) {
	pods, listOpts, err := r.getPods(ctx, instance, r.feature)
	if err != nil {
		handlePodListError(logger, err, listOpts)
	}

	instanceStatuses, err := getInstanceStatuses(pods)
	if err != nil {
		if instanceStatuses == nil || len(instanceStatuses) <= 0 {
			return false, err
		}
	}

	if instance.Status.OneAgent.Instances == nil || !reflect.DeepEqual(instance.Status.OneAgent.Instances, instanceStatuses) {
		instance.Status.OneAgent.Instances = instanceStatuses
		return true, err
	}

	return false, err
}
func getInstanceStatuses(pods []corev1.Pod) (map[string]dynatracev1alpha1.OneAgentInstance, error) {
	instanceStatuses := make(map[string]dynatracev1alpha1.OneAgentInstance)

	for _, pod := range pods {
		instanceStatuses[pod.Spec.NodeName] = dynatracev1alpha1.OneAgentInstance{
			PodName:   pod.Name,
			IPAddress: pod.Status.HostIP,
		}
	}

	return instanceStatuses, nil
}

// SetUseImmutableImageStatus updates the status' UseImmutableImage field to indicate whether the Operator should use
// immutable images or not.
func setUseImmutableImageStatus(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec) bool {
	if fs.UseImmutableImage == instance.Status.OneAgent.UseImmutableImage {
		return false
	}

	instance.Status.OneAgent.UseImmutableImage = fs.UseImmutableImage
	return true
}

func (r *ReconcileOneAgent) determineDynaKubePhase(instance *dynatracev1alpha1.DynaKube) (bool, error) {
	var phaseChanged bool
	dsActual := &appsv1.DaemonSet{}
	instanceName := fmt.Sprintf("%s-%s", instance.Name, r.feature)
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instanceName, Namespace: instance.Namespace}, dsActual)

	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		phaseChanged = instance.Status.Phase != dynatracev1alpha1.Error
		instance.Status.Phase = dynatracev1alpha1.Error
		return phaseChanged, err
	}

	if dsActual.Status.NumberReady == dsActual.Status.CurrentNumberScheduled {
		phaseChanged = instance.Status.Phase != dynatracev1alpha1.Running
		instance.Status.Phase = dynatracev1alpha1.Running
	} else {
		phaseChanged = instance.Status.Phase != dynatracev1alpha1.Deploying
		instance.Status.Phase = dynatracev1alpha1.Deploying
	}

	return phaseChanged, nil
}
