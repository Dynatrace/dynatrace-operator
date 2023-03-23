package daemonset

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	annotationUnprivileged      = "container.apparmor.security.beta.kubernetes.io/dynatrace-oneagent"
	annotationUnprivilegedValue = "unconfined"

	unprivilegedServiceAccountName = "dynatrace-dynakube-oneagent-unprivileged"
	privilegedServiceAccountName   = "dynatrace-dynakube-oneagent-privileged"

	// normal oneagent shutdown scenario with some extra time
	defaultTerminationGracePeriod = int64(80)

	hostRootVolumeName      = "host-root"
	hostRootVolumeMountPath = "/mnt/root"

	clusterCaCertVolumeName      = "dynatrace-cluster-ca"
	clusterCaCertVolumeMountPath = "/mnt/dynatrace/certs"

	activeGateCaCertVolumeName      = "active-gate-ca"
	activeGateCaCertVolumeMountPath = "/mnt/dynatrace/certs/activegate/"

	csiStorageVolumeName  = "osagent-storage"
	csiStorageVolumeMount = "/mnt/volume_storage_mount"

	podName = "dynatrace-oneagent"

	inframonHostIdSource = "--set-host-id-source=k8s-node-name"
	classicHostIdSource  = "--set-host-id-source=auto"
)

type HostMonitoring struct {
	builderInfo
}

type ClassicFullStack struct {
	builderInfo
}

type builderInfo struct {
	dynakube       *dynatracev1beta1.DynaKube
	hostInjectSpec *dynatracev1beta1.HostInjectSpec
	clusterID      string
	deploymentType string
}

type Builder interface {
	BuildDaemonSet() (*appsv1.DaemonSet, error)
}

func NewHostMonitoring(instance *dynatracev1beta1.DynaKube, clusterId string) Builder {
	return &HostMonitoring{
		builderInfo{
			dynakube:       instance,
			hostInjectSpec: instance.Spec.OneAgent.HostMonitoring,
			clusterID:      clusterId,
			deploymentType: deploymentmetadata.HostMonitoringDeploymentType,
		},
	}
}

func NewCloudNativeFullStack(instance *dynatracev1beta1.DynaKube, clusterId string) Builder {
	return &HostMonitoring{
		builderInfo{
			dynakube:       instance,
			hostInjectSpec: &instance.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec,
			clusterID:      clusterId,
			deploymentType: deploymentmetadata.CloudNativeDeploymentType,
		},
	}
}

func NewClassicFullStack(instance *dynatracev1beta1.DynaKube, clusterId string) Builder {
	return &ClassicFullStack{
		builderInfo{
			dynakube:       instance,
			hostInjectSpec: instance.Spec.OneAgent.ClassicFullStack,
			clusterID:      clusterId,
			deploymentType: deploymentmetadata.ClassicFullStackDeploymentType,
		},
	}
}

func (dsInfo *HostMonitoring) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	result, err := dsInfo.builderInfo.BuildDaemonSet()
	if err != nil {
		return nil, err
	}

	result.Name = dsInfo.dynakube.OneAgentDaemonsetName()

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

	result.Name = dsInfo.dynakube.OneAgentDaemonsetName()

	if len(result.Spec.Template.Spec.Containers) > 0 {
		appendHostIdArgument(result, classicHostIdSource)
	}

	return result, nil
}

func appendHostIdArgument(result *appsv1.DaemonSet, source string) {
	result.Spec.Template.Spec.Containers[0].Args = append(result.Spec.Template.Spec.Containers[0].Args, source)
}

func (dsInfo *builderInfo) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	dynakube := dsInfo.dynakube
	podSpec := dsInfo.podSpec()

	versionLabelValue := dynakube.Status.OneAgent.Version

	appLabels := kubeobjects.NewAppLabels(kubeobjects.OneAgentComponentLabel, dynakube.Name,
		dsInfo.deploymentType, versionLabelValue)
	labels := kubeobjects.MergeMap(
		appLabels.BuildLabels(),
		dsInfo.hostInjectSpec.Labels,
	)
	maxUnavailable := intstr.FromInt(dynakube.FeatureOneAgentMaxUnavailable())
	annotations := map[string]string{
		annotationUnprivileged:            annotationUnprivilegedValue,
		webhook.AnnotationDynatraceInject: "false",
	}

	annotations = kubeobjects.MergeMap(annotations, dsInfo.hostInjectSpec.Annotations)

	result := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        dynakube.Name,
			Namespace:   dynakube.Namespace,
			Labels:      labels,
			Annotations: map[string]string{},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: appLabels.BuildMatchLabels(),
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
			Image:           dsInfo.immutableOneAgentImage(),
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
			SecurityContext: dsInfo.securityContext(),
			VolumeMounts:    volumeMounts,
		}},
		ImagePullSecrets:              imagePullSecrets,
		HostNetwork:                   true,
		HostPID:                       true,
		HostIPC:                       false,
		NodeSelector:                  dsInfo.nodeSelector(),
		PriorityClassName:             dsInfo.priorityClassName(),
		ServiceAccountName:            dsInfo.serviceAccountName(),
		Tolerations:                   dsInfo.tolerations(),
		DNSPolicy:                     dnsPolicy,
		Volumes:                       volumes,
		Affinity:                      affinity,
		TerminationGracePeriodSeconds: address.Of(defaultTerminationGracePeriod),
	}
}

func (dsInfo *builderInfo) serviceAccountName() string {
	if dsInfo.dynakube != nil && dsInfo.dynakube.NeedsOneAgentPrivileged() {
		return privilegedServiceAccountName
	}

	return unprivilegedServiceAccountName
}

func (dsInfo *builderInfo) immutableOneAgentImage() string {
	if dsInfo.dynakube == nil {
		return ""
	}
	return dsInfo.dynakube.OneAgentImage()
}

func (dsInfo *builderInfo) tolerations() []corev1.Toleration {
	if dsInfo.hostInjectSpec != nil {
		return dsInfo.hostInjectSpec.Tolerations
	}

	return nil
}

func (dsInfo *builderInfo) priorityClassName() string {
	if dsInfo.hostInjectSpec != nil {
		return dsInfo.hostInjectSpec.PriorityClassName
	}

	return ""
}

func (dsInfo *builderInfo) nodeSelector() map[string]string {
	if dsInfo.hostInjectSpec == nil {
		return make(map[string]string, 0)
	}

	return dsInfo.hostInjectSpec.NodeSelector
}

func (dsInfo *builderInfo) resources() corev1.ResourceRequirements {
	resources := dsInfo.oneAgentResource()
	if resources.Requests == nil {
		resources.Requests = corev1.ResourceList{}
	}
	if _, hasCPUResource := resources.Requests[corev1.ResourceCPU]; !hasCPUResource {
		// Set CPU resource to 1 * 10**(-1) Cores, e.g. 100mC
		resources.Requests[corev1.ResourceCPU] = *resource.NewScaledQuantity(1, -1)
	}
	return resources
}

func (dsInfo *builderInfo) oneAgentResource() corev1.ResourceRequirements {
	if dsInfo.hostInjectSpec == nil {
		return corev1.ResourceRequirements{}
	}

	return dsInfo.hostInjectSpec.OneAgentResources
}

func (dsInfo *builderInfo) dnsPolicy() corev1.DNSPolicy {
	if dsInfo.hostInjectSpec != nil && dsInfo.hostInjectSpec.DNSPolicy != "" {
		return dsInfo.hostInjectSpec.DNSPolicy
	}
	return corev1.DNSClusterFirstWithHostNet
}

func (dsInfo *builderInfo) volumeMounts() []corev1.VolumeMount {
	return prepareVolumeMounts(dsInfo.dynakube)
}

func (dsInfo *builderInfo) volumes() []corev1.Volume {
	return prepareVolumes(dsInfo.dynakube)
}

func (dsInfo *builderInfo) imagePullSecrets() []corev1.LocalObjectReference {
	if dsInfo.dynakube == nil {
		return []corev1.LocalObjectReference{}
	}

	return []corev1.LocalObjectReference{{Name: dsInfo.dynakube.PullSecret()}}
}

func (dsInfo *builderInfo) securityContext() *corev1.SecurityContext {
	var securityContext corev1.SecurityContext
	if dsInfo.dynakube != nil && dsInfo.dynakube.NeedsReadOnlyOneAgents() {
		securityContext.RunAsNonRoot = address.Of(true)
		securityContext.RunAsUser = address.Of(int64(1000))
		securityContext.RunAsGroup = address.Of(int64(1000))
	}

	if dsInfo.dynakube != nil && dsInfo.dynakube.NeedsOneAgentPrivileged() {
		securityContext.Privileged = address.Of(true)
	} else {
		securityContext.Capabilities = defaultSecurityContextCapabilities()

		if dsInfo.dynakube != nil && dsInfo.dynakube.FeatureOneAgentSecCompProfile() != "" {
			secCompName := dsInfo.dynakube.FeatureOneAgentSecCompProfile()
			securityContext.SeccompProfile = &corev1.SeccompProfile{
				Type:             corev1.SeccompProfileTypeLocalhost,
				LocalhostProfile: &secCompName,
			}
		}
	}
	return &securityContext
}

func defaultSecurityContextCapabilities() *corev1.Capabilities {
	return &corev1.Capabilities{
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
	}
}
