package daemonset

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	labelFeature        = "operator.dynatrace.com/feature"
	labelAgentType      = "operator.dynatrace.com/agenttype"
	labelAgentTypeValue = "os"

	annotationUnprivileged      = "container.apparmor.security.beta.kubernetes.io/dynatrace-oneagent"
	annotationUnprivilegedValue = "unconfined"
	annotationVersion           = dynatracev1beta1.InternalFlagPrefix + "version"

	defaultUnprivilegedServiceAccountName = "dynatrace-dynakube-oneagent-unprivileged"

	hostRootMount = "host-root"

	podName = "dynatrace-oneagent"

	inframonHostIdSource = "--set-host-id-source=k8s-node-name"
	classicHostIdSource  = "--set-host-id-source=auto"

	PodNameOSAgent = "oneagent"

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
	instance       *dynatracev1beta1.DynaKube
	hostInjectSpec *dynatracev1beta1.HostInjectSpec
	clusterId      string
	deploymentType string
}

type Builder interface {
	BuildDaemonSet() (*appsv1.DaemonSet, error)
}

func NewHostMonitoring(instance *dynatracev1beta1.DynaKube, clusterId string) Builder {
	return &HostMonitoring{
		builderInfo{
			instance:       instance,
			hostInjectSpec: &instance.Spec.OneAgent.HostMonitoring.HostInjectSpec,
			clusterId:      clusterId,
			deploymentType: deploymentmetadata.DeploymentTypeHostMonitoring,
		},
		HostMonitoringFeature,
	}
}

func NewCloudNativeFullStack(instance *dynatracev1beta1.DynaKube, clusterId string) Builder {
	return &HostMonitoring{
		builderInfo{
			instance:       instance,
			hostInjectSpec: &instance.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec,
			clusterId:      clusterId,
			deploymentType: deploymentmetadata.DeploymentTypeCloudNative,
		},
		CloudNativeFeature,
	}
}

func NewClassicFullStack(instance *dynatracev1beta1.DynaKube, clusterId string) Builder {
	return &ClassicFullStack{
		builderInfo{
			instance:       instance,
			hostInjectSpec: &instance.Spec.OneAgent.ClassicFullStack.HostInjectSpec,
			clusterId:      clusterId,
			deploymentType: deploymentmetadata.DeploymentTypeFullStack,
		},
	}
}

func (dsInfo *HostMonitoring) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	result, err := dsInfo.builderInfo.BuildDaemonSet()
	if err != nil {
		return nil, err
	}

	result.Name = fmt.Sprintf("%s-%s", dsInfo.instance.Name, PodNameOSAgent)
	result.Labels[labelFeature] = dsInfo.feature
	result.Spec.Selector.MatchLabels[labelAgentType] = labelAgentTypeValue
	result.Spec.Template.Labels[labelFeature] = dsInfo.feature
	result.Spec.Template.Labels[labelAgentType] = labelAgentTypeValue

	if len(result.Spec.Template.Spec.Containers) > 0 {
		appendHostIdArgument(result, inframonHostIdSource)
		dsInfo.appendInfraMonEnvVars(result)
	}

	return result, nil
}

func (dsInfo *ClassicFullStack) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	result, err := dsInfo.builderInfo.BuildDaemonSet()
	if err != nil {
		return nil, err
	}

	result.Name = fmt.Sprintf("%s-%s", dsInfo.instance.Name, PodNameOSAgent)
	result.Labels[labelFeature] = ClassicFeature
	result.Spec.Selector.MatchLabels[labelAgentType] = labelAgentTypeValue
	result.Spec.Template.Labels[labelFeature] = ClassicFeature
	result.Spec.Template.Labels[labelAgentType] = labelAgentTypeValue

	if len(result.Spec.Template.Spec.Containers) > 0 {
		appendHostIdArgument(result, classicHostIdSource)
	}

	return result, nil
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
	imagePullSecrets := dsInfo.imagePullSecrets()
	affinity := dsInfo.affinity()

	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Args:            arguments,
			Env:             environmentVariables,
			Image:           dsInfo.instance.ImmutableOneAgentImage(),
			ImagePullPolicy: corev1.PullAlways,
			Name:            podName,
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
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
