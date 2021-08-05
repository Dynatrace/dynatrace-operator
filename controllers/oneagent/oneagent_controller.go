package oneagent

import (
	"context"
	"os"
	"reflect"
	"strconv"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/reconciler/statefulset"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// time between consecutive queries for a new pod to get ready
	splayTimeSeconds                      = uint16(10)
	defaultUpdateInterval                 = 15 * time.Minute
	updateEnvVar                          = "ONEAGENT_OPERATOR_UPDATE_INTERVAL"
	ClassicFeature                        = "classic"
	InframonFeature                       = "inframon"
	defaultOneAgentImage                  = "docker.io/dynatrace/oneagent:latest"
	defaultServiceAccountName             = "dynatrace-dynakube-oneagent"
	defaultUnprivilegedServiceAccountName = "dynatrace-dynakube-oneagent-unprivileged"
)

// NewOneAgentReconciler initializes a new ReconcileOneAgent instance
func NewOneAgentReconciler(client client.Client, apiReader client.Reader, scheme *runtime.Scheme, logger logr.Logger, instance *dynatracev1alpha1.DynaKube, fullStack *dynatracev1alpha1.FullStackSpec, feature string) *ReconcileOneAgent {
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

// ReconcileOneAgent reconciles a OneAgent object
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
func (r *ReconcileOneAgent) Reconcile(ctx context.Context, rec *utils.Reconciliation) (bool, error) {
	r.logger.Info("Reconciling OneAgent")
	if err := validate(r.instance); err != nil {
		return false, err
	}

	rec.Update(utils.SetUseImmutableImageStatus(r.instance, r.fullStack), 5*time.Minute, "UseImmutableImage changed")

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

	if rec.IsOutdated(r.instance.Status.OneAgent.LastHostsRequestTimestamp, updInterval) {
		r.instance.Status.OneAgent.LastHostsRequestTimestamp = rec.Now.DeepCopy()
		rec.Update(true, 5*time.Minute, "updated last host request time stamp")

		upd, err = r.reconcileInstanceStatuses(ctx, r.logger, r.instance)
		rec.Update(upd, 5*time.Minute, "Instance statuses reconciled")
		if rec.Error(err) {
			return false, err
		}
	}

	// Finally we have to determine the correct non error phase
	_, err = r.determineOneAgentPhase(r.instance)
	rec.Error(err)

	return upd, nil
}

func (r *ReconcileOneAgent) reconcileRollout(rec *utils.Reconciliation) (bool, error) {
	updateCR := false

	// Define a new DaemonSet object
	dsDesired, err := r.getDesiredDaemonSet(rec)
	if err != nil {
		rec.Log.Info("Failed to get desired daemonset")
		return false, err
	}

	// Set OneAgent instance as the owner and controller
	if err := controllerutil.SetControllerReference(rec.Instance, dsDesired, r.scheme); err != nil {
		return false, err
	}

	// Check if this DaemonSet already exists
	updateCR, err = kubeobjects.CreateOrUpdateDaemonSet(r.client, r.logger, dsDesired)
	if err != nil {
		return updateCR, err
	}

	if rec.Instance.Status.Tokens != rec.Instance.Tokens() {
		rec.Instance.Status.Tokens = rec.Instance.Tokens()
		updateCR = true
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) getDesiredDaemonSet(rec *utils.Reconciliation) (*appsv1.DaemonSet, error) {
	kubeSysUID, err := kubesystem.GetUID(r.apiReader)
	if err != nil {
		return nil, err
	}

	dsDesired, err := newDaemonSetForCR(rec.Log, rec.Instance, r.fullStack, string(kubeSysUID), r.feature)
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

func newDaemonSetForCR(logger logr.Logger, instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, clusterID string, feature string) (*appsv1.DaemonSet, error) {
	unprivileged := true
	if ptr := fs.UseUnprivilegedMode; ptr != nil {
		unprivileged = *ptr
	}

	name := instance.GetName() + "-" + feature
	podSpec := newPodSpecForCR(instance, fs, feature, unprivileged, logger, clusterID)
	selectorLabels := buildLabels(instance.GetName(), feature)
	mergedLabels := mergeLabels(fs.Labels, selectorLabels)

	maxUnavailable := intstr.FromInt(instance.FeatureOneAgentMaxUnavailable())

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   instance.GetNamespace(),
			Labels:      mergedLabels,
			Annotations: map[string]string{},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: mergedLabels,
					Annotations: map[string]string{
						statefulset.AnnotationVersion: instance.Status.OneAgent.Version,
					},
				},
				Spec: podSpec,
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
				},
			},
		},
	}

	if unprivileged {
		ds.Spec.Template.ObjectMeta.Annotations["container.apparmor.security.beta.kubernetes.io/dynatrace-oneagent"] = "unconfined"
	}

	dsHash, err := kubeobjects.GenerateDaemonSetHash(ds)
	if err != nil {
		return nil, err
	}
	ds.Annotations[statefulset.AnnotationTemplateHash] = dsHash

	return ds, nil
}

func newPodSpecForCR(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, feature string, unprivileged bool, logger logr.Logger, clusterID string) corev1.PodSpec {
	p := corev1.PodSpec{}

	sa := "dynatrace-dynakube-oneagent"
	if fs.ServiceAccountName != "" {
		sa = fs.ServiceAccountName
	} else if unprivileged {
		sa = "dynatrace-dynakube-oneagent-unprivileged"
	}

	resources := fs.Resources
	if resources.Requests == nil {
		resources.Requests = corev1.ResourceList{}
	}
	if _, hasCPUResource := resources.Requests[corev1.ResourceCPU]; !hasCPUResource {
		// Set CPU resource to 1 * 10**(-1) Cores, e.g. 100mC
		resources.Requests[corev1.ResourceCPU] = *resource.NewScaledQuantity(1, -1)
	}

	dnsPolicy := fs.DNSPolicy
	if dnsPolicy == "" {
		dnsPolicy = corev1.DNSClusterFirstWithHostNet
	}

	secCtx := prepareSecurityContext(unprivileged, fs)

	p = corev1.PodSpec{
		Containers: []corev1.Container{{
			Args:            prepareArgs(instance, fs, feature, clusterID),
			Env:             prepareEnvVars(instance, fs, feature, clusterID),
			Image:           "",
			ImagePullPolicy: corev1.PullAlways,
			Name:            "dynatrace-oneagent",
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"/bin/sh", "-c", "grep -q oneagentwatchdo /proc/[0-9]*/stat",
						},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       30,
				TimeoutSeconds:      1,
			},
			Resources:       resources,
			SecurityContext: secCtx,
			VolumeMounts:    prepareVolumeMounts(instance, fs),
		}},
		HostNetwork:        true,
		HostPID:            true,
		HostIPC:            false,
		NodeSelector:       fs.NodeSelector,
		PriorityClassName:  fs.PriorityClassName,
		ServiceAccountName: sa,
		Tolerations:        fs.Tolerations,
		DNSPolicy:          dnsPolicy,
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "beta.kubernetes.io/arch",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"amd64", "arm64"},
								},
								{
									Key:      "beta.kubernetes.io/os",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"linux"},
								},
							},
						},
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/arch",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"amd64", "arm64"},
								},
								{
									Key:      "kubernetes.io/os",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"linux"},
								},
							},
						},
					},
				},
			},
		},
		Volumes: prepareVolumes(instance, fs),
	}

	if instance.Status.OneAgent.UseImmutableImage {
		err := preparePodSpecImmutableImage(&p, instance)
		if err != nil {
			logger.Error(err, "failed to prepare pod spec v2")
		}
	} else {
		err := preparePodSpecInstaller(&p, instance)
		if err != nil {
			logger.Error(err, "failed to prepare pod spec v1")
		}
	}

	return p
}

func preparePodSpecInstaller(p *corev1.PodSpec, instance *dynatracev1alpha1.DynaKube) error {
	img := "docker.io/dynatrace/oneagent:latest"
	envVarImg := os.Getenv("RELATED_IMAGE_DYNATRACE_ONEAGENT")

	if instance.Spec.OneAgent.Image != "" {
		img = instance.Spec.OneAgent.Image
	} else if envVarImg != "" {
		img = envVarImg
	}

	p.Containers[0].Image = img
	return nil
}

func preparePodSpecImmutableImage(p *corev1.PodSpec, instance *dynatracev1alpha1.DynaKube) error {
	pullSecretName := instance.PullSecret()

	p.ImagePullSecrets = append(p.ImagePullSecrets, corev1.LocalObjectReference{
		Name: pullSecretName,
	})

	if instance.Spec.OneAgent.Image != "" {
		p.Containers[0].Image = instance.Spec.OneAgent.Image
		return nil
	}

	p.Containers[0].Image = instance.ImmutableOneAgentImage()
	return nil
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
