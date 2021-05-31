package oneagent

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"reflect"
	"strconv"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
func NewOneAgentReconciler(client client.Client, apiReader client.Reader, scheme *runtime.Scheme, logger logr.Logger,
	dtc dtclient.Client, instance *dynatracev1alpha1.DynaKube, fullStack *dynatracev1alpha1.FullStackSpec, feature string) *ReconcileOneAgent {
	return &ReconcileOneAgent{
		client:    client,
		apiReader: apiReader,
		scheme:    scheme,
		logger:    logger,
		dtc:       dtc,
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
	dtc       dtclient.Client
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

	upd, err := r.reconcileRollout(ctx, rec)
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

		upd, err := r.reconcileInstanceStatuses(ctx, r.logger, r.instance, r.dtc)
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

func (r *ReconcileOneAgent) reconcileRollout(ctx context.Context, rec *utils.Reconciliation) (bool, error) {
	updateCR := false

	// Define a new DaemonSet object
	dsDesired, err := r.getDesiredDaemonSet(rec)
	if err != nil {
		rec.Log.Info("Failed to get desired daemonset")
		return updateCR, err
	}

	// Set OneAgent instance as the owner and controller
	if err := controllerutil.SetControllerReference(rec.Instance, dsDesired, r.scheme); err != nil {
		return false, err
	}

	// Check if this DaemonSet already exists
	dsActual := &appsv1.DaemonSet{}
	err = r.client.Get(ctx, types.NamespacedName{Name: dsDesired.Name, Namespace: dsDesired.Namespace}, dsActual)
	if err != nil && k8serrors.IsNotFound(err) {
		rec.Log.Info("Creating new daemonset")
		if err = r.client.Create(ctx, dsDesired); err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	} else if hasDaemonSetChanged(dsDesired, dsActual) {
		rec.Log.Info("Updating existing daemonset")
		if err = r.client.Update(ctx, dsDesired); err != nil {
			return false, err
		}
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
						activegate.AnnotationVersion: instance.Status.OneAgent.Version,
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

	dsHash, err := generateDaemonSetHash(ds)
	if err != nil {
		return nil, err
	}
	ds.Annotations[activegate.AnnotationTemplateHash] = dsHash

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

	// K8s 1.18+ is expected to drop the "beta.kubernetes.io" labels in favor of "kubernetes.io" which was added on K8s 1.14.
	// To support both older and newer K8s versions we use node affinity.

	var secCtx *corev1.SecurityContext
	if unprivileged {
		secCtx = &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"ALL",
				},
				Add: []corev1.Capability{
					"CHOWN",
					"DAC_OVERRIDE",
					"DAC_READ_SEARCH",
					"FOWNER",
					"FSETID",
					"KILL",
					"NET_ADMIN",
					"NET_RAW",
					"SETFCAP",
					"SETGID",
					"SETUID",
					"SYS_ADMIN",
					"SYS_CHROOT",
					"SYS_PTRACE",
					"SYS_RESOURCE",
				},
			},
		}
	} else {
		trueVar := true
		secCtx = &corev1.SecurityContext{
			Privileged: &trueVar,
		}
	}

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
			VolumeMounts:    prepareVolumeMounts(instance),
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
		Volumes: prepareVolumes(instance),
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
	pullSecretName := instance.GetName() + "-pull-secret"
	if instance.Spec.CustomPullSecret != "" {
		pullSecretName = instance.Spec.CustomPullSecret
	}

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

func prepareVolumes(instance *dynatracev1alpha1.DynaKube) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "host-root",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/",
				},
			},
		},
	}

	if instance.Spec.TrustedCAs != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "certs",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: instance.Spec.TrustedCAs,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "certs",
							Path: "certs.pem",
						},
					},
				},
			},
		})
	}

	return volumes
}

func prepareVolumeMounts(instance *dynatracev1alpha1.DynaKube) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "host-root",
			MountPath: "/mnt/root",
		},
	}

	if instance.Spec.TrustedCAs != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "certs",
			MountPath: "/mnt/dynatrace/certs",
		})
	}

	return volumeMounts
}

func prepareEnvVars(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec, feature string, clusterID string) []corev1.EnvVar {
	type reservedEnvVar struct {
		Name    string
		Default func(ev *corev1.EnvVar)
		Value   *corev1.EnvVar
	}

	reserved := []reservedEnvVar{
		{
			Name: "DT_K8S_NODE_NAME",
			Default: func(ev *corev1.EnvVar) {
				ev.ValueFrom = &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}
			},
		},
		{
			Name: "DT_K8S_CLUSTER_ID",
			Default: func(ev *corev1.EnvVar) {
				ev.Value = clusterID
			},
		},
	}

	if feature == InframonFeature {
		reserved = append(reserved,
			reservedEnvVar{
				Name: "ONEAGENT_DISABLE_CONTAINER_INJECTION",
				Default: func(ev *corev1.EnvVar) {
					ev.Value = "true"
				},
			})
	}

	if !instance.Status.OneAgent.UseImmutableImage {
		reserved = append(reserved,
			reservedEnvVar{
				Name: "ONEAGENT_INSTALLER_DOWNLOAD_TOKEN",
				Default: func(ev *corev1.EnvVar) {
					ev.ValueFrom = &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: instance.Tokens()},
							Key:                  utils.DynatracePaasToken,
						},
					}
				},
			},
			reservedEnvVar{
				Name: "ONEAGENT_INSTALLER_SCRIPT_URL",
				Default: func(ev *corev1.EnvVar) {
					ev.Value = fmt.Sprintf("%s/v1/deployment/installer/agent/unix/default/latest?arch=x86&flavor=default", instance.Spec.APIURL)
				},
			},
			reservedEnvVar{
				Name: "ONEAGENT_INSTALLER_SKIP_CERT_CHECK",
				Default: func(ev *corev1.EnvVar) {
					ev.Value = strconv.FormatBool(instance.Spec.SkipCertCheck)
				},
			})

		if p := instance.Spec.Proxy; p != nil && (p.Value != "" || p.ValueFrom != "") {
			reserved = append(reserved, reservedEnvVar{
				Name: "https_proxy",
				Default: func(ev *corev1.EnvVar) {
					if p.ValueFrom != "" {
						ev.ValueFrom = &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: instance.Spec.Proxy.ValueFrom},
								Key:                  "proxy",
							},
						}
					} else {
						p.Value = instance.Spec.Proxy.Value
					}
				},
			})
		}
	}

	reservedMap := map[string]*reservedEnvVar{}
	for i := range reserved {
		reservedMap[reserved[i].Name] = &reserved[i]
	}

	// Split defined environment variables between those reserved and the rest

	instanceEnv := fs.Env

	var remaining []corev1.EnvVar
	for i := range instanceEnv {
		if p := reservedMap[instanceEnv[i].Name]; p != nil {
			p.Value = &instanceEnv[i]
			continue
		}
		remaining = append(remaining, instanceEnv[i])
	}

	// Add reserved environment variables in that order, and generate a default if unset.

	var env []corev1.EnvVar
	for i := range reserved {
		ev := reserved[i].Value
		if ev == nil {
			ev = &corev1.EnvVar{Name: reserved[i].Name}
			reserved[i].Default(ev)
		}
		env = append(env, *ev)
	}

	return append(env, remaining...)
}

func hasDaemonSetChanged(a, b *appsv1.DaemonSet) bool {
	return getTemplateHash(a) != getTemplateHash(b)
}

func generateDaemonSetHash(ds *appsv1.DaemonSet) (string, error) {
	data, err := json.Marshal(ds)
	if err != nil {
		return "", err
	}

	hasher := fnv.New32()
	_, err = hasher.Write(data)
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(uint64(hasher.Sum32()), 10), nil
}

func getTemplateHash(a metav1.Object) string {
	if annotations := a.GetAnnotations(); annotations != nil {
		return annotations[activegate.AnnotationTemplateHash]
	}
	return ""
}

func (r *ReconcileOneAgent) reconcileInstanceStatuses(ctx context.Context, logger logr.Logger, instance *dynatracev1alpha1.DynaKube, dtc dtclient.Client) (bool, error) {
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
