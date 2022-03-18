package oneagent

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/status"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	instance *dynatracev1beta1.DynaKube,
	feature string) *OneAgentReconciler {
	return &OneAgentReconciler{
		client:    client,
		apiReader: apiReader,
		scheme:    scheme,
		instance:  instance,
		feature:   feature,
	}
}

type OneAgentReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	apiReader client.Reader
	scheme    *runtime.Scheme
	instance  *dynatracev1beta1.DynaKube
	feature   string
}

// Reconcile reads that state of the cluster for a OneAgent object and makes changes based on the state read
// and what is in the OneAgent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *OneAgentReconciler) Reconcile(ctx context.Context, rec *status.DynakubeState) (bool, error) {
	log.Info("reconciling OneAgent")

	upd, err := r.reconcileRollout(rec)
	if err != nil {
		return false, err
	} else if upd {
		log.Info("rollout reconciled")
	}

	updInterval := defaultUpdateInterval
	if val := os.Getenv(updateEnvVar); val != "" {
		x, err := strconv.Atoi(val)
		if err != nil {
			log.Info("conversion of ONEAGENT_OPERATOR_UPDATE_INTERVAL failed")
		} else {
			updInterval = time.Duration(x) * time.Minute
		}
	}

	if rec.IsOutdated(r.instance.Status.OneAgent.LastHostsRequestTimestamp, updInterval) {
		r.instance.Status.OneAgent.LastHostsRequestTimestamp = rec.Now.DeepCopy()
		rec.Update(true, 5*time.Minute, "updated last host request time stamp")

		upd, err = r.reconcileInstanceStatuses(ctx, r.instance)
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

func (r *OneAgentReconciler) reconcileRollout(dkState *status.DynakubeState) (bool, error) {
	updateCR := false

	// Define a new DaemonSet object
	dsDesired, err := r.getDesiredDaemonSet(dkState)
	if err != nil {
		log.Info("failed to get desired daemonset")
		return false, err
	}

	// Set OneAgent instance as the owner and controller
	if err := controllerutil.SetControllerReference(dkState.Instance, dsDesired, r.scheme); err != nil {
		return false, err
	}

	updateCR, err = kubeobjects.CreateOrUpdateDaemonSet(r.client, log, dsDesired)
	if err != nil {
		return updateCR, err
	}
	if updateCR {
		// remove old daemonset with feature in name
		oldClassicDaemonset := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", r.instance.Name, daemonset.ClassicFeature),
				Namespace: r.instance.Namespace,
			},
		}
		err = r.client.Delete(context.TODO(), oldClassicDaemonset)
		if err == nil {
			log.Info("removed oneagent daemonset with feature in name")
		} else if !k8serrors.IsNotFound(err) {
			return false, err
		}
	}

	if dkState.Instance.Status.Tokens != dkState.Instance.Tokens() {
		dkState.Instance.Status.Tokens = dkState.Instance.Tokens()
		updateCR = true
	}

	return updateCR, nil
}

func (r *OneAgentReconciler) getDesiredDaemonSet(dkState *status.DynakubeState) (*appsv1.DaemonSet, error) {
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

func (r *OneAgentReconciler) getPods(ctx context.Context, instance *dynatracev1beta1.DynaKube, feature string) ([]corev1.Pod, []client.ListOption, error) {
	podList := &corev1.PodList{}
	listOps := []client.ListOption{
		client.InNamespace((*instance).GetNamespace()),
		client.MatchingLabels(daemonset.BuildLabels(instance.Name, feature)),
	}
	err := r.client.List(ctx, podList, listOps...)
	return podList.Items, listOps, err
}

func (r *OneAgentReconciler) newDaemonSetForCR(dkState *status.DynakubeState, clusterID string) (*appsv1.DaemonSet, error) {
	var ds *appsv1.DaemonSet
	var err error

	if r.feature == daemonset.ClassicFeature {
		ds, err = daemonset.NewClassicFullStack(dkState.Instance, clusterID).BuildDaemonSet()
	} else if r.feature == daemonset.HostMonitoringFeature {
		ds, err = daemonset.NewHostMonitoring(dkState.Instance, clusterID).BuildDaemonSet()
	} else if r.feature == daemonset.CloudNativeFeature {
		ds, err = daemonset.NewCloudNativeFullStack(dkState.Instance, clusterID).BuildDaemonSet()
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

func (r *OneAgentReconciler) reconcileInstanceStatuses(ctx context.Context, instance *dynatracev1beta1.DynaKube) (bool, error) {
	pods, listOpts, err := r.getPods(ctx, instance, r.feature)
	if err != nil {
		handlePodListError(err, listOpts)
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

func getInstanceStatuses(pods []corev1.Pod) (map[string]dynatracev1beta1.OneAgentInstance, error) {
	instanceStatuses := make(map[string]dynatracev1beta1.OneAgentInstance)

	for _, pod := range pods {
		instanceStatuses[pod.Spec.NodeName] = dynatracev1beta1.OneAgentInstance{
			PodName:   pod.Name,
			IPAddress: pod.Status.HostIP,
		}
	}

	return instanceStatuses, nil
}

func (r *OneAgentReconciler) determineDynaKubePhase(instance *dynatracev1beta1.DynaKube) (bool, error) {
	var phaseChanged bool
	dsActual := &appsv1.DaemonSet{}
	instanceName := instance.OneAgentDaemonsetName()
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: instanceName, Namespace: instance.Namespace}, dsActual)

	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		phaseChanged = instance.Status.Phase != dynatracev1beta1.Error
		instance.Status.Phase = dynatracev1beta1.Error
		return phaseChanged, err
	}

	if dsActual.Status.NumberReady == dsActual.Status.CurrentNumberScheduled {
		phaseChanged = instance.Status.Phase != dynatracev1beta1.Running
		instance.Status.Phase = dynatracev1beta1.Running
	} else {
		phaseChanged = instance.Status.Phase != dynatracev1beta1.Deploying
		instance.Status.Phase = dynatracev1beta1.Deploying
	}

	return phaseChanged, nil
}
