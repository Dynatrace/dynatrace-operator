package oneagent

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/configmap"
	k8sdaemonset "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/object"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultUpdateInterval = 5 * time.Minute
	updateEnvVar          = "ONEAGENT_OPERATOR_UPDATE_INTERVAL"
	oldDsName             = "classic"
)

type ReconcilerBuilder func(
	client client.Client,
	apiReader client.Reader,
	scheme *runtime.Scheme,
	clusterID string) *Reconciler

// NewOneAgentReconciler initializes a new ReconcileOneAgent instance
func NewOneAgentReconciler(
	client client.Client,
	apiReader client.Reader,
	scheme *runtime.Scheme,
	clusterID string) *Reconciler {
	return &Reconciler{
		client:    client,
		apiReader: apiReader,
		scheme:    scheme,
		clusterID: clusterID,
	}
}

type Reconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	apiReader client.Reader
	scheme    *runtime.Scheme
	clusterID string
}

// Reconcile reads that state of the cluster for a OneAgent object and makes changes based on the state read
// and what is in the OneAgent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *Reconciler) Reconcile(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	if !dynakube.NeedsOneAgent() {
		log.Info("removing OneAgent daemonSet")

		return r.removeOneAgentDaemonSet(ctx, dynakube)
	}

	log.Info("reconciling OneAgent")

	if !dynakube.IsOneAgentCommunicationRouteClear() {
		log.Info("OneAgent were not yet able to communicate with tenant, no direct route or ready ActiveGate available, postponing OneAgent deployment")

		if len(dynakube.Spec.NetworkZone) > 0 {
			log.Info("A network zone has been configured for DynaKube, check that there a working ActiveGate ready for that network zone", "network zone", dynakube.Spec.NetworkZone, "dynakube", dynakube.Name)
		}

		return nil
	}

	log.Info("At least one ActiveGate is operational, deploying OneAgent")

	err := r.createOneAgentTenantConnectionInfoConfigMap(ctx, dynakube)
	if err != nil {
		return err
	}

	err = r.reconcileRollout(ctx, dynakube)
	if err != nil {
		return err
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

	now := metav1.Now()
	if timeprovider.TimeoutReached(dynakube.Status.OneAgent.LastInstanceStatusUpdate, &now, updInterval) {
		err = r.reconcileInstanceStatuses(ctx, dynakube)
		if err != nil {
			return err
		}

		dynakube.Status.OneAgent.LastInstanceStatusUpdate = &now

		log.Info("oneagent instance statuses reconciled")
	}

	log.Info("reconciled " + deploymentmetadata.GetOneAgentDeploymentType(*dynakube))

	return nil
}

func (r *Reconciler) createOneAgentTenantConnectionInfoConfigMap(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	configMapData := extractPublicData(dynakube)

	configMap, err := configmap.CreateConfigMap(r.scheme, dynakube,
		configmap.NewModifier(dynakube.OneAgentConnectionInfoConfigMapName()),
		configmap.NewNamespaceModifier(dynakube.Namespace),
		configmap.NewConfigMapDataModifier(configMapData))
	if err != nil {
		return errors.WithStack(err)
	}

	query := configmap.NewQuery(ctx, r.client, r.apiReader, log)

	err = query.CreateOrUpdate(*configMap)
	if err != nil {
		log.Info("could not create or update configMap for connection info", "name", configMap.Name)

		return err
	}

	return nil
}

func extractPublicData(dynakube *dynatracev1beta1.DynaKube) map[string]string {
	data := map[string]string{}

	if dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID != "" {
		data[connectioninfo.TenantUUIDName] = dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID
	}

	if dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints != "" {
		data[connectioninfo.CommunicationEndpointsName] = dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints
	}

	return data
}

func (r *Reconciler) reconcileRollout(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	// Define a new DaemonSet object
	dsDesired, err := r.buildDesiredDaemonSet(dynakube)
	if err != nil {
		log.Info("failed to get desired daemonset")

		return err
	}

	// Set OneAgent instance as the owner and controller
	if err := controllerutil.SetControllerReference(dynakube, dsDesired, r.scheme); err != nil {
		return err
	}

	updated, err := k8sdaemonset.CreateOrUpdateDaemonSet(r.client, log, dsDesired)
	if err != nil {
		log.Info("failed to roll out new OneAgent DaemonSet")

		return err
	}

	if updated {
		log.Info("rolled out new OneAgent DaemonSet")
		// remove old daemonset with feature in name
		oldClassicDaemonset := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", dynakube.Name, oldDsName),
				Namespace: dynakube.Namespace,
			},
		}

		err = r.client.Delete(ctx, oldClassicDaemonset)
		if err == nil {
			log.Info("removed oneagent daemonset with feature in name")
		} else if !k8serrors.IsNotFound(err) {
			log.Info("failed to remove oneagent daemonset with feature in name")

			return err
		}
	}

	return nil
}

func (r *Reconciler) getOneagentPods(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, feature string) ([]corev1.Pod, []client.ListOption, error) {
	agentVersion := dynakube.OneAgentVersion()
	appLabels := labels.NewAppLabels(labels.OneAgentComponentLabel, dynakube.Name,
		feature, agentVersion)
	podList := &corev1.PodList{}
	listOps := []client.ListOption{
		client.InNamespace((*dynakube).GetNamespace()),
		client.MatchingLabels(appLabels.BuildLabels()),
	}
	err := r.client.List(ctx, podList, listOps...)

	return podList.Items, listOps, err
}

func (r *Reconciler) buildDesiredDaemonSet(dynakube *dynatracev1beta1.DynaKube) (*appsv1.DaemonSet, error) {
	var ds *appsv1.DaemonSet

	var err error

	switch {
	case dynakube.ClassicFullStackMode():
		ds, err = daemonset.NewClassicFullStack(dynakube, r.clusterID).BuildDaemonSet()
	case dynakube.HostMonitoringMode():
		ds, err = daemonset.NewHostMonitoring(dynakube, r.clusterID).BuildDaemonSet()
	case dynakube.CloudNativeFullstackMode():
		ds, err = daemonset.NewCloudNativeFullStack(dynakube, r.clusterID).BuildDaemonSet()
	}

	if err != nil {
		return nil, err
	}

	dsHash, err := hasher.GenerateHash(ds)
	if err != nil {
		return nil, err
	}

	ds.Annotations[hasher.AnnotationHash] = dsHash

	return ds, nil
}

func (r *Reconciler) reconcileInstanceStatuses(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	pods, listOpts, err := r.getOneagentPods(ctx, dynakube, deploymentmetadata.GetOneAgentDeploymentType(*dynakube))
	if err != nil {
		handlePodListError(err, listOpts)
	}

	instanceStatuses := getInstanceStatuses(pods)
	if err != nil {
		if len(instanceStatuses) == 0 {
			return err
		}
	}

	if dynakube.Status.OneAgent.Instances == nil || !reflect.DeepEqual(dynakube.Status.OneAgent.Instances, instanceStatuses) {
		dynakube.Status.OneAgent.Instances = instanceStatuses

		return err
	}

	return err
}

func (r *Reconciler) removeOneAgentDaemonSet(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	oneAgentDaemonSet := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dynakube.OneAgentDaemonsetName(), Namespace: dynakube.Namespace}}

	return object.Delete(ctx, r.client, &oneAgentDaemonSet)
}

func getInstanceStatuses(pods []corev1.Pod) map[string]dynatracev1beta1.OneAgentInstance {
	instanceStatuses := make(map[string]dynatracev1beta1.OneAgentInstance)

	for _, pod := range pods {
		instanceStatuses[pod.Spec.NodeName] = dynatracev1beta1.OneAgentInstance{
			PodName:   pod.Name,
			IPAddress: pod.Status.HostIP,
		}
	}

	return instanceStatuses
}
