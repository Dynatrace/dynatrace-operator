package daemonset

import (
	"fmt"
	"os"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
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
	annotationVersion           = "internal.operator.dynatrace.com/version"

	defaultUnprivilegedServiceAccountName = "dynatrace-dynakube-oneagent-unprivileged"

	hostRootMount = "host-root"

	oneagentInstallationMountName = "oneagent-installation"
	oneagentInstallationMountPath = "/mnt/volume_storage_mount"

	relatedImageEnvVar = "RELATED_IMAGE_DYNATRACE_ONEAGENT"

	podName = "dynatrace-oneagent"

	defaultUserId  = 1001
	defaultGroupId = 1001

	inframonHostIdSource = "--set-host-id-source=k8s-node-name"
	classicHostIdSource  = "--set-host-id-source=auto"

	ClassicFeature        = "classic"
	HostMonitoringFeature = "inframon"
	CloudNativeFeature    = "cloud-native"
)

type HostMonitoring struct {
	builderInfo
	feature string
}

type ClassicFullStack struct {
	builderInfo
}

type builderInfo struct {
	instance               *dynatracev1beta1.DynaKube
	hostInjectSpec         *dynatracev1beta1.HostInjectSpec
	logger                 logr.Logger
	clusterId              string
	relatedImage           string
	deploymentType         string
	minorKubernetesVersion string
	majorKubernetesVersion string
}

type Builder interface {
	BuildDaemonSet() (*appsv1.DaemonSet, error)
}

func NewHostMonitoring(instance *dynatracev1beta1.DynaKube, logger logr.Logger, clusterId string, majorKubernetesVersion string, minorKubernetesVersion string) Builder {
	return &HostMonitoring{
		builderInfo{
			instance:               instance,
			hostInjectSpec:         &instance.Spec.OneAgent.HostMonitoring.HostInjectSpec,
			logger:                 logger,
			clusterId:              clusterId,
			relatedImage:           os.Getenv(relatedImageEnvVar),
			deploymentType:         deploymentmetadata.DeploymentTypeHM,
			majorKubernetesVersion: majorKubernetesVersion,
			minorKubernetesVersion: minorKubernetesVersion,
		},
		HostMonitoringFeature,
	}
}

func NewCloudNativeFullStack(instance *dynatracev1beta1.DynaKube, logger logr.Logger, clusterId string, majorKubernetesVersion string, minorKubernetesVersion string) Builder {
	return &HostMonitoring{
		builderInfo{
			instance:               instance,
			hostInjectSpec:         &instance.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec,
			logger:                 logger,
			clusterId:              clusterId,
			relatedImage:           os.Getenv(relatedImageEnvVar),
			deploymentType:         deploymentmetadata.DeploymentTypeHM,
			majorKubernetesVersion: majorKubernetesVersion,
			minorKubernetesVersion: minorKubernetesVersion,
		},
		CloudNativeFeature,
	}
}

func NewClassicFullStack(instance *dynatracev1beta1.DynaKube, logger logr.Logger, clusterId string, majorKubernetesVersion string, minorKubernetesVersion string) Builder {
	return &ClassicFullStack{
		builderInfo{
			instance:               instance,
			hostInjectSpec:         &instance.Spec.OneAgent.ClassicFullStack.HostInjectSpec,
			logger:                 logger,
			clusterId:              clusterId,
			relatedImage:           os.Getenv(relatedImageEnvVar),
			deploymentType:         deploymentmetadata.DeploymentTypeFS,
			majorKubernetesVersion: majorKubernetesVersion,
			minorKubernetesVersion: minorKubernetesVersion,
		},
	}
}

func (dsInfo *HostMonitoring) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	result, err := dsInfo.builderInfo.BuildDaemonSet()
	if err != nil {
		return nil, err
	}

	result.Name = dsInfo.instance.Name + fmt.Sprintf("-%s", dsInfo.feature)
	result.Labels[labelFeature] = dsInfo.feature
	result.Spec.Selector.MatchLabels[labelFeature] = dsInfo.feature
	result.Spec.Template.Labels[labelFeature] = dsInfo.feature

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

func (dsInfo *HostMonitoring) setSecurityContextOptions(daemonset *appsv1.DaemonSet) {
	securityContext := daemonset.Spec.Template.Spec.Containers[0].SecurityContext

	if dsInfo.instance.ReadOnly() {
		securityContext.RunAsUser = pointer.Int64Ptr(defaultUserId)
		securityContext.RunAsGroup = pointer.Int64Ptr(defaultGroupId)
	}
}

func appendHostIdArgument(result *appsv1.DaemonSet, source string) {
	result.Spec.Template.Spec.Containers[0].Args = append(result.Spec.Template.Spec.Containers[0].Args, source)
}

func (dsInfo *builderInfo) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	instance := dsInfo.instance
	podSpec := dsInfo.podSpec()
	labels := kubeobjects.MergeLabels(dsInfo.buildLabels(), dsInfo.hostInjectSpec.Labels)
	maxUnavailable := intstr.FromInt(instance.FeatureOneAgentMaxUnavailable())
	annotations := map[string]string{
		annotationVersion:      instance.Status.OneAgent.Version,
		annotationUnprivileged: annotationUnprivilegedValue,
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
	resources := dsInfo.resources()
	dnsPolicy := dsInfo.dnsPolicy()
	arguments := dsInfo.arguments()
	environmentVariables := dsInfo.environmentVariables()
	volumeMounts := dsInfo.volumeMounts()
	volumes := dsInfo.volumes()
	image := dsInfo.image()
	imagePullSecrets := dsInfo.imagePullSecrets()
	affinity := dsInfo.affinity()

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
			SecurityContext: unprivilegedSecurityContext(),
			VolumeMounts:    volumeMounts,
		}},
		ImagePullSecrets:   imagePullSecrets,
		HostNetwork:        true,
		HostPID:            true,
		HostIPC:            false,
		NodeSelector:       dsInfo.hostInjectSpec.NodeSelector,
		PriorityClassName:  dsInfo.hostInjectSpec.PriorityClassName,
		ServiceAccountName: defaultUnprivilegedServiceAccountName,
		Tolerations:        dsInfo.hostInjectSpec.Tolerations,
		DNSPolicy:          dnsPolicy,
		Volumes:            volumes,
		Affinity:           affinity,
	}
}

func (dsInfo *builderInfo) buildLabels() map[string]string {
	return map[string]string{
		"dynatrace.com/component":         "operator",
		"operator.dynatrace.com/instance": dsInfo.instance.Name,
	}
}

func (dsInfo *builderInfo) resources() corev1.ResourceRequirements {
	resources := dsInfo.hostInjectSpec.OneAgentResources
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
	if dsInfo.hostInjectSpec.DNSPolicy != "" {
		return dsInfo.hostInjectSpec.DNSPolicy
	}
	return corev1.DNSClusterFirstWithHostNet
}

func (dsInfo *builderInfo) volumeMounts() []corev1.VolumeMount {
	return prepareVolumeMounts(dsInfo.instance)
}

func (dsInfo *builderInfo) volumes() []corev1.Volume {
	return prepareVolumes(dsInfo.instance)
}

func (dsInfo *builderInfo) image() string {
	if dsInfo.relatedImage != "" {
		return dsInfo.relatedImage
	}
	return dsInfo.instance.ImmutableOneAgentImage()
}

func (dsInfo *builderInfo) imagePullSecrets() []corev1.LocalObjectReference {
	pullSecretName := dsInfo.instance.PullSecret()
	pullSecrets := make([]corev1.LocalObjectReference, 0)

	pullSecrets = append(pullSecrets, corev1.LocalObjectReference{
		Name: pullSecretName,
	})
	return pullSecrets
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
