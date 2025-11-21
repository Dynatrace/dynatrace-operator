package oneagent

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	oaconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/configmap"
	k8sdaemonset "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultUpdateInterval = 5 * time.Minute
	updateEnvVar          = "ONEAGENT_OPERATOR_UPDATE_INTERVAL"
	oldDsName             = "classic"
)

var _ ReconcilerBuilder = NewReconciler

type ReconcilerBuilder func(
	client client.Client,
	apiReader client.Reader,
	dtClient dynatrace.Client,
	dynakube *dynakube.DynaKube,
	tokens token.Tokens,
	clusterID string,
	operatorImageName string,
) controllers.Reconciler

// NewReconciler initializes a new ReconcileOneAgent instance
func NewReconciler( //nolint
	client client.Client,
	apiReader client.Reader,
	dtClient dynatrace.Client,
	dk *dynakube.DynaKube,
	tokens token.Tokens,
	clusterID string,
	operatorImageName string,
) controllers.Reconciler {
	return &Reconciler{
		client:                   client,
		apiReader:                apiReader,
		clusterID:                clusterID,
		dk:                       dk,
		connectionInfoReconciler: oaconnectioninfo.NewReconciler(client, apiReader, dtClient, dk),
		versionReconciler:        version.NewReconciler(apiReader, dtClient, timeprovider.New().Freeze()),
		tokens:                   tokens,
		operatorImageName:        operatorImageName,
	}
}

type Reconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client                   client.Client
	apiReader                client.Reader
	connectionInfoReconciler controllers.Reconciler
	versionReconciler        version.Reconciler
	dk                       *dynakube.DynaKube
	tokens                   token.Tokens
	clusterID                string
	operatorImageName        string
}

// Reconcile reads that state of the cluster for a OneAgent object and makes changes based on the state read
// and what is in the OneAgent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *Reconciler) Reconcile(ctx context.Context) error {
	log.Info("reconciling OneAgent")

	err := r.versionReconciler.ReconcileOneAgent(ctx, r.dk)
	if err != nil {
		return err
	}

	err = r.connectionInfoReconciler.Reconcile(ctx)
	if errors.Is(err, oaconnectioninfo.NoOneAgentCommunicationHostsError) { // This only informational
		log.Info("OneAgent were not yet able to communicate with tenant, no direct route or ready ActiveGate available, postponing OneAgent deployment")

		if r.dk.Spec.NetworkZone != "" {
			log.Info("A network zone has been configured for DynaKube, check that there a working ActiveGate ready for that network zone", "network zone", r.dk.Spec.NetworkZone, "dynakube", r.dk.Name)
		}
	}

	if err != nil {
		return err
	}

	if !r.dk.OneAgent().IsDaemonsetRequired() {
		return r.cleanUp(ctx)
	}

	err = dtpullsecret.NewReconciler(r.client, r.apiReader, r.dk, r.tokens).Reconcile(ctx)
	if err != nil {
		return err
	}

	log.Info("At least one communication host is provided, deploying OneAgent")

	err = r.createOneAgentTenantConnectionInfoConfigMap(ctx)
	if err != nil {
		return err
	}

	err = r.reconcileRollout(ctx)
	if err != nil {
		return err
	}

	err = r.updateInstancesStatus(ctx)
	if err != nil {
		return err
	}

	log.Info("reconciled " + deploymentmetadata.GetOneAgentDeploymentType(*r.dk))

	return nil
}

func (r *Reconciler) cleanUp(ctx context.Context) error {
	log.Info("removing OneAgent daemonSet")

	if meta.FindStatusCondition(*r.dk.Conditions(), oaConditionType) == nil {
		return nil // no condition == nothing is there to clean up
	}

	err := r.deleteOneAgentTenantConnectionInfoConfigMap(ctx)
	if err != nil {
		log.Error(err, "failed to cleanup oneagent connection-info configmap") // error shouldn't block another cleanup
	}

	meta.RemoveStatusCondition(r.dk.Conditions(), oaConditionType)

	// be careful with OneAgent Status cleanup, as some things (ConnectionInfo) are shared with injection.
	// only cleanup things that are directly set in THIS reconciler
	r.dk.Status.OneAgent.Instances = nil
	r.dk.Status.OneAgent.LastInstanceStatusUpdate = nil

	return r.removeOneAgentDaemonSet(ctx, r.dk)
}

func (r *Reconciler) updateInstancesStatus(ctx context.Context) error {
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
	if timeprovider.TimeoutReached(r.dk.Status.OneAgent.LastInstanceStatusUpdate, &now, updInterval) {
		err := r.reconcileInstanceStatuses(ctx, r.dk)
		if err != nil {
			return err
		}

		r.dk.Status.OneAgent.LastInstanceStatusUpdate = &now

		log.Info("oneagent instance statuses reconciled")
	}

	return nil
}

func (r *Reconciler) createOneAgentTenantConnectionInfoConfigMap(ctx context.Context) error {
	configMapData := extractPublicData(r.dk)

	configMap, err := configmap.Build(r.dk,
		r.dk.OneAgent().GetConnectionInfoConfigMapName(),
		configMapData,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	query := configmap.Query(r.client, r.apiReader, log)

	_, err = query.CreateOrUpdate(ctx, configMap)
	if err != nil {
		log.Info("could not create or update configMap for connection info", "name", configMap.Name)
		conditions.SetKubeAPIError(r.dk.Conditions(), oaConditionType, err)

		return err
	}

	return nil
}

func (r *Reconciler) deleteOneAgentTenantConnectionInfoConfigMap(ctx context.Context) error {
	cm, _ := configmap.Build(r.dk,
		r.dk.OneAgent().GetConnectionInfoConfigMapName(),
		nil,
	)
	query := configmap.Query(r.client, r.apiReader, log)

	return query.Delete(ctx, cm)
}

func extractPublicData(dk *dynakube.DynaKube) map[string]string {
	data := map[string]string{}

	if dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID != "" {
		data[connectioninfo.TenantUUIDKey] = dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID
	}

	if dk.Status.OneAgent.ConnectionInfoStatus.Endpoints != "" {
		data[connectioninfo.CommunicationEndpointsKey] = dk.Status.OneAgent.ConnectionInfoStatus.Endpoints
	}

	return data
}

func (r *Reconciler) reconcileRollout(ctx context.Context) error {
	// Define a new DaemonSet object
	dsDesired, err := r.buildDesiredDaemonSet(r.dk)
	if err != nil {
		log.Info("failed to get desired daemonset")
		setDaemonSetGenerationFailedCondition(r.dk.Conditions(), err)

		return err
	}

	// Set OneAgent instance as the owner and controller
	if err := controllerutil.SetControllerReference(r.dk, dsDesired, scheme.Scheme); err != nil {
		return err
	}

	updated, err := k8sdaemonset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, dsDesired)
	if err != nil {
		log.Info("failed to roll out new OneAgent DaemonSet")
		conditions.SetKubeAPIError(r.dk.Conditions(), oaConditionType, err)

		return err
	}

	if updated {
		log.Info("rolled out new OneAgent DaemonSet")
		setDaemonSetCreatedCondition(r.dk.Conditions())

		// remove old daemonset with feature in name
		oldClassicDaemonset := &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", r.dk.Name, oldDsName),
				Namespace: r.dk.Namespace,
			},
		}

		err = r.client.Delete(ctx, oldClassicDaemonset)
		if err == nil {
			log.Info("removed oneagent daemonset with feature in name")
		} else if !k8serrors.IsNotFound(err) {
			log.Info("failed to remove oneagent daemonset with feature in name")
			conditions.SetKubeAPIError(r.dk.Conditions(), oaConditionType, err)

			return err
		}
	}

	return nil
}

func (r *Reconciler) getOneagentPods(ctx context.Context, dk *dynakube.DynaKube, feature string) ([]corev1.Pod, []client.ListOption, error) {
	agentVersion := dk.OneAgent().GetVersion()
	appLabels := labels.NewAppLabels(labels.OneAgentComponentLabel, dk.Name,
		feature, agentVersion)
	podList := &corev1.PodList{}
	listOps := []client.ListOption{
		client.InNamespace((*dk).GetNamespace()),
		client.MatchingLabels(appLabels.BuildLabels()),
	}
	err := r.client.List(ctx, podList, listOps...)

	return podList.Items, listOps, err
}

func (r *Reconciler) buildDesiredDaemonSet(dk *dynakube.DynaKube) (*appsv1.DaemonSet, error) {
	var ds *appsv1.DaemonSet

	var err error

	switch {
	case dk.OneAgent().IsClassicFullStackMode():
		ds, err = daemonset.NewClassicFullStack(dk, r.clusterID, r.operatorImageName).BuildDaemonSet()
	case dk.OneAgent().IsHostMonitoringMode():
		ds, err = daemonset.NewHostMonitoring(dk, r.clusterID, r.operatorImageName).BuildDaemonSet()
	case dk.OneAgent().IsCloudNativeFullstackMode():
		ds, err = daemonset.NewCloudNativeFullStack(dk, r.clusterID, r.operatorImageName).BuildDaemonSet()
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

func (r *Reconciler) reconcileInstanceStatuses(ctx context.Context, dk *dynakube.DynaKube) error {
	pods, listOpts, err := r.getOneagentPods(ctx, dk, deploymentmetadata.GetOneAgentDeploymentType(*dk))
	if err != nil {
		handlePodListError(err, listOpts)
	}

	instanceStatuses := getInstanceStatuses(pods)
	if err != nil {
		if len(instanceStatuses) == 0 {
			return err
		}
	}

	if dk.Status.OneAgent.Instances == nil || !reflect.DeepEqual(dk.Status.OneAgent.Instances, instanceStatuses) {
		dk.Status.OneAgent.Instances = instanceStatuses

		return err
	}

	return err
}

func (r *Reconciler) removeOneAgentDaemonSet(ctx context.Context, dk *dynakube.DynaKube) error {
	oneAgentDaemonSet := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dk.OneAgent().GetDaemonsetName(), Namespace: dk.Namespace}}

	return client.IgnoreNotFound(r.client.Delete(ctx, &oneAgentDaemonSet))
}

func getInstanceStatuses(pods []corev1.Pod) map[string]oneagent.Instance {
	instanceStatuses := make(map[string]oneagent.Instance)

	for _, pod := range pods {
		instanceStatuses[pod.Spec.NodeName] = oneagent.Instance{
			PodName:   pod.Name,
			IPAddress: pod.Status.HostIP,
		}
	}

	return instanceStatuses
}
