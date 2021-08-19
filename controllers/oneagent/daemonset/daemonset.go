package daemonset

import (
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/reconciler/statefulset"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

const (
	labelFeature = "operator.dynatrace.com/feature"

	annotationUnprivileged      = "container.apparmor.security.beta.kubernetes.io/dynatrace-oneagent"
	annotationUnprivilegedValue = "unconfined"

	defaultServiceAccountName             = "dynatrace-dynakube-oneagent"
	defaultUnprivilegedServiceAccountName = "dynatrace-dynakube-oneagent-unprivileged"
	defaultOneAgentImage                  = "docker.io/dynatrace/oneagent:latest"

	hostRootMount = "host-root"

	oneagentInstallationMountName = "oneagent-installation"
	oneagentInstallationMountPath = "/mnt/volume_storage_mount"

	relatedImageEnvVar = "RELATED_IMAGE_DYNATRACE_ONEAGENT"

	podName = "dynatrace-oneagent"

	defaultUserId  = 1001
	defaultGroupId = 1001

	inframonHostIdSource = "--set-host-id-source=k8s-node-name"
	classicHostIdSource  = "--set-host-id-source=auto"

	ClassicFeature  = "classic"
	InframonFeature = "inframon"
)

type InfraMonitoring struct {
	builderInfo
}

type ClassicFullStack struct {
	builderInfo
}

type builderInfo struct {
	instance      *v1alpha1.DynaKube
	fullstackSpec *v1alpha1.FullStackSpec
	logger        logr.Logger
	clusterId     string
	relatedImage  string
}

type Builder interface {
	BuildDaemonSet() (*appsv1.DaemonSet, error)
}

func NewInfraMonitoring(instance *v1alpha1.DynaKube, logger logr.Logger, clusterId string) Builder {
	return &InfraMonitoring{
		builderInfo{
			instance:      instance,
			fullstackSpec: &instance.Spec.InfraMonitoring.FullStackSpec,
			logger:        logger,
			clusterId:     clusterId,
			relatedImage:  os.Getenv(relatedImageEnvVar),
		},
	}
}

func NewClassicFullStack(instance *v1alpha1.DynaKube, logger logr.Logger, clusterId string) Builder {
	return &ClassicFullStack{
		builderInfo{
			instance:      instance,
			fullstackSpec: &instance.Spec.ClassicFullStack,
			logger:        logger,
			clusterId:     clusterId,
			relatedImage:  os.Getenv(relatedImageEnvVar),
		},
	}
}

func (dsInfo *InfraMonitoring) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	result, err := dsInfo.builderInfo.BuildDaemonSet()
	if err != nil {
		return nil, err
	}

	result.Name = dsInfo.instance.Name + fmt.Sprintf("-%s", InframonFeature)
	result.Labels[labelFeature] = InframonFeature
	result.Spec.Selector.MatchLabels[labelFeature] = InframonFeature
	result.Spec.Template.Labels[labelFeature] = InframonFeature

	if len(result.Spec.Template.Spec.Containers) > 0 {
		appendHostIdArgument(result, inframonHostIdSource)
		dsInfo.setSecurityContextOptions(result)
		dsInfo.appendInfraMonEnvVars(result)
		dsInfo.appendReadOnlyVolume(result)
		dsInfo.appendReadOnlyVolumeMount(result)
		dsInfo.setRootMountReadability(result)
	}

	return result, nil
}

func (dsInfo *ClassicFullStack) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	result, err := dsInfo.builderInfo.BuildDaemonSet()
	if err != nil {
		return nil, err
	}

	result.Name = dsInfo.instance.Name + fmt.Sprintf("-%s", ClassicFeature)
	result.Labels[labelFeature] = ClassicFeature
	result.Spec.Selector.MatchLabels[labelFeature] = ClassicFeature
	result.Spec.Template.Labels[labelFeature] = ClassicFeature

	if len(result.Spec.Template.Spec.Containers) > 0 {
		appendHostIdArgument(result, classicHostIdSource)
	}

	return result, nil
}

func (dsInfo *InfraMonitoring) setSecurityContextOptions(daemonset *appsv1.DaemonSet) {
	securityContext := daemonset.Spec.Template.Spec.Containers[0].SecurityContext
	securityContext.RunAsUser = pointer.Int64Ptr(defaultUserId)
	securityContext.RunAsGroup = pointer.Int64Ptr(defaultGroupId)
}

func appendHostIdArgument(result *appsv1.DaemonSet, source string) {
	result.Spec.Template.Spec.Containers[0].Args = append(result.Spec.Template.Spec.Containers[0].Args, source)
}

func (dsInfo *builderInfo) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	instance := dsInfo.instance
	podSpec := dsInfo.podSpec()
	labels := kubeobjects.MergeLabels(dsInfo.buildLabels(), dsInfo.fullstackSpec.Labels)
	maxUnavailable := intstr.FromInt(instance.FeatureOneAgentMaxUnavailable())
	annotations := map[string]string{
		statefulset.AnnotationVersion: instance.Status.OneAgent.Version,
	}

	if dsInfo.unprivileged() {
		annotations[annotationUnprivileged] = annotationUnprivilegedValue
	}

	result := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: map[string]string{},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: dsInfo.buildLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
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

	return result, nil
}

func (dsInfo *builderInfo) podSpec() corev1.PodSpec {
	serviceAccount := dsInfo.serviceAccountName()
	resources := dsInfo.resources()
	dnsPolicy := dsInfo.dnsPolicy()
	securityContext := dsInfo.securityContext()
	arguments := dsInfo.arguments()
	environmentVariables := dsInfo.environmentVariables()
	volumeMounts := dsInfo.volumeMounts()
	volumes := dsInfo.volumes()

	image := dsInfo.image()
	imagePullSecrets := dsInfo.imagePullSecrets()

	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Args:            arguments,
			Env:             environmentVariables,
			Image:           image,
			ImagePullPolicy: corev1.PullAlways,
			Name:            podName,
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
			SecurityContext: securityContext,
			VolumeMounts:    volumeMounts,
		}},
		ImagePullSecrets:   imagePullSecrets,
		HostNetwork:        true,
		HostPID:            true,
		HostIPC:            false,
		NodeSelector:       dsInfo.fullstackSpec.NodeSelector,
		PriorityClassName:  dsInfo.fullstackSpec.PriorityClassName,
		ServiceAccountName: serviceAccount,
		Tolerations:        dsInfo.fullstackSpec.Tolerations,
		DNSPolicy:          dnsPolicy,
		Volumes:            volumes,
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
	}
}

func (dsInfo *builderInfo) useImmutableImage() bool {
	return dsInfo.instance.Status.OneAgent.UseImmutableImage
}

func (dsInfo *builderInfo) buildLabels() map[string]string {
	return map[string]string{
		"dynatrace.com/component":         "operator",
		"operator.dynatrace.com/instance": dsInfo.instance.Name,
	}
}

func (dsInfo *builderInfo) serviceAccountName() string {
	if dsInfo.fullstackSpec.ServiceAccountName != "" {
		return dsInfo.fullstackSpec.ServiceAccountName
	} else if dsInfo.unprivileged() {
		return defaultUnprivilegedServiceAccountName
	}
	return defaultServiceAccountName
}

func (dsInfo *builderInfo) unprivileged() bool {
	return dsInfo.fullstackSpec.UseUnprivilegedMode == nil || *dsInfo.fullstackSpec.UseUnprivilegedMode
}

func (dsInfo *builderInfo) resources() corev1.ResourceRequirements {
	resources := dsInfo.fullstackSpec.Resources
	if resources.Requests == nil {
		resources.Requests = corev1.ResourceList{}
	}
	if _, hasCPUResource := resources.Requests[corev1.ResourceCPU]; !hasCPUResource {
		// Set CPU resource to 1 * 10**(-1) Cores, e.g. 100mC
		resources.Requests[corev1.ResourceCPU] = *resource.NewScaledQuantity(1, -1)
	}
	return resources
}

func (dsInfo *builderInfo) dnsPolicy() corev1.DNSPolicy {
	if dsInfo.fullstackSpec.DNSPolicy != "" {
		return dsInfo.fullstackSpec.DNSPolicy
	}
	return corev1.DNSClusterFirstWithHostNet
}

func (dsInfo *builderInfo) securityContext() *corev1.SecurityContext {
	if dsInfo.unprivileged() {
		return unprivilegedSecurityContext()
	}
	return privilegedSecurityContext()
}

func (dsInfo *builderInfo) volumeMounts() []corev1.VolumeMount {
	return prepareVolumeMounts(dsInfo.instance)
}

func (dsInfo *builderInfo) volumes() []corev1.Volume {
	return prepareVolumes(dsInfo.instance, dsInfo.fullstackSpec)
}

func (dsInfo *builderInfo) image() string {
	if dsInfo.instance.Spec.OneAgent.Image != "" {
		return dsInfo.instance.Spec.OneAgent.Image
	}

	if dsInfo.useImmutableImage() {
		return dsInfo.instance.ImmutableOneAgentImage()
	} else if dsInfo.relatedImage != "" {
		return dsInfo.relatedImage
	}

	return defaultOneAgentImage
}

func (dsInfo *builderInfo) imagePullSecrets() []corev1.LocalObjectReference {
	pullSecretName := dsInfo.instance.PullSecret()
	pullSecrets := make([]corev1.LocalObjectReference, 0)

	if !dsInfo.useImmutableImage() {
		return pullSecrets
	}

	pullSecrets = append(pullSecrets, corev1.LocalObjectReference{
		Name: pullSecretName,
	})
	return pullSecrets
}

func privilegedSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged: pointer.BoolPtr(true),
	}
}

func unprivilegedSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
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
}
