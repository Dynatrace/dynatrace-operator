package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	"golang.org/x/mod/semver"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	annotationUnprivileged            = "container.apparmor.security.beta.kubernetes.io/dynatrace-oneagent"
	annotationUnprivilegedValue       = "unconfined"
	annotationTenantTokenHash         = api.InternalFlagPrefix + "tenant-token-hash"
	annotationEnableDaemonSetEviction = "cluster-autoscaler.kubernetes.io/enable-ds-eviction"

	serviceAccountName = "dynatrace-dynakube-oneagent"

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

	storageVolumeName = "volume-storage"

	podName = "dynatrace-oneagent"

	inframonHostIDSource = "k8s-node-name"
	classicHostIDSource  = "auto"

	probeMaxInitialDelay         = int32(90)
	probeDefaultSuccessThreshold = int32(1)

	readOnlyRootFsConstraint = "v1.291"
)

type hostMonitoring struct {
	builder
}

type classicFullStack struct {
	builder
}

type builder struct {
	dk             *dynakube.DynaKube
	hostInjectSpec *oneagent.HostInjectSpec
	clusterID      string
	deploymentType string
}

type Builder interface {
	BuildDaemonSet() (*appsv1.DaemonSet, error)
}

func NewHostMonitoring(dk *dynakube.DynaKube, clusterID string) Builder {
	return &hostMonitoring{
		builder{
			dk:             dk,
			hostInjectSpec: dk.Spec.OneAgent.HostMonitoring,
			clusterID:      clusterID,
			deploymentType: deploymentmetadata.HostMonitoringDeploymentType,
		},
	}
}

func NewCloudNativeFullStack(dk *dynakube.DynaKube, clusterID string) Builder {
	return &hostMonitoring{
		builder{
			dk:             dk,
			hostInjectSpec: &dk.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec,
			clusterID:      clusterID,
			deploymentType: deploymentmetadata.CloudNativeDeploymentType,
		},
	}
}

func NewClassicFullStack(dk *dynakube.DynaKube, clusterID string) Builder {
	return &classicFullStack{
		builder{
			dk:             dk,
			hostInjectSpec: dk.Spec.OneAgent.ClassicFullStack,
			clusterID:      clusterID,
			deploymentType: deploymentmetadata.ClassicFullStackDeploymentType,
		},
	}
}

func (hm *hostMonitoring) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	daemonSet, err := hm.builder.BuildDaemonSet()
	if err != nil {
		return nil, err
	}

	daemonSet.Name = hm.dk.OneAgent().GetDaemonsetName()

	if len(daemonSet.Spec.Template.Spec.Containers) > 0 {
		hm.appendInfraMonEnvVars(daemonSet)
	}

	return daemonSet, nil
}

func (classic *classicFullStack) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	result, err := classic.builder.BuildDaemonSet()
	if err != nil {
		return nil, err
	}

	result.Name = classic.dk.OneAgent().GetDaemonsetName()

	return result, nil
}

func (b *builder) BuildDaemonSet() (*appsv1.DaemonSet, error) {
	dk := b.dk

	podSpec, err := b.podSpec()
	if err != nil {
		return nil, err
	}

	versionLabelValue := dk.OneAgent().GetVersion()

	appLabels := labels.NewAppLabels(labels.OneAgentComponentLabel, dk.Name,
		b.deploymentType, versionLabelValue)
	labels := maputils.MergeMap(
		appLabels.BuildLabels(),
		b.hostInjectSpec.Labels,
	)
	maxUnavailable := intstr.FromInt(dk.FF().GetOneAgentMaxUnavailable())

	templateAnnotations := map[string]string{
		annotationUnprivileged:            annotationUnprivilegedValue,
		webhook.AnnotationDynatraceInject: "false",
		annotationTenantTokenHash:         dk.Status.OneAgent.ConnectionInfoStatus.TenantTokenHash,
		annotationEnableDaemonSetEviction: "false",
	}

	templateAnnotations = maputils.MergeMap(templateAnnotations, b.hostInjectSpec.Annotations)

	result := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        dk.Name,
			Namespace:   dk.Namespace,
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
					Annotations: templateAnnotations,
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

func (b *builder) podSpec() (corev1.PodSpec, error) {
	resources := b.resources()
	dnsPolicy := b.dnsPolicy()

	arguments, err := b.arguments()
	if err != nil {
		return corev1.PodSpec{}, err
	}

	environmentVariables, err := b.environmentVariables()
	if err != nil {
		return corev1.PodSpec{}, err
	}

	volumeMounts := b.volumeMounts()
	volumes := b.volumes()
	imagePullSecrets := b.imagePullSecrets()
	affinity := b.affinity()

	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{{
			Args:            arguments,
			Env:             environmentVariables,
			Image:           b.immutableOneAgentImage(),
			ImagePullPolicy: corev1.PullAlways,
			Name:            podName,
			Resources:       resources,
			SecurityContext: b.securityContext(),
			VolumeMounts:    volumeMounts,
		}},
		ImagePullSecrets:              imagePullSecrets,
		HostNetwork:                   true,
		HostPID:                       true,
		HostIPC:                       false,
		NodeSelector:                  b.nodeSelector(),
		PriorityClassName:             b.priorityClassName(),
		ServiceAccountName:            serviceAccountName,
		Tolerations:                   b.tolerations(),
		DNSPolicy:                     dnsPolicy,
		Volumes:                       volumes,
		Affinity:                      affinity,
		TerminationGracePeriodSeconds: ptr.To(defaultTerminationGracePeriod),
	}

	if b.dk.OneAgent().IsReadinessProbeNeeded() {
		podSpec.Containers[0].ReadinessProbe = b.getReadinessProbe()
	}

	if b.dk.OneAgent().IsLivenessProbeNeeded() {
		podSpec.Containers[0].LivenessProbe = b.getDefaultProbeFromStatus()
	}

	return podSpec, nil
}

func (b *builder) immutableOneAgentImage() string {
	if b.dk == nil {
		return ""
	}

	return b.dk.OneAgent().GetImage()
}

func (b *builder) tolerations() []corev1.Toleration {
	if b.hostInjectSpec != nil {
		return b.hostInjectSpec.Tolerations
	}

	return nil
}

func (b *builder) priorityClassName() string {
	if b.hostInjectSpec != nil {
		return b.hostInjectSpec.PriorityClassName
	}

	return ""
}

func (b *builder) nodeSelector() map[string]string {
	if b.hostInjectSpec == nil {
		return make(map[string]string, 0)
	}

	return b.hostInjectSpec.NodeSelector
}

func (b *builder) resources() corev1.ResourceRequirements {
	resources := b.oneAgentResource()
	if resources.Requests == nil {
		resources.Requests = corev1.ResourceList{}
	}

	if _, hasCPUResource := resources.Requests[corev1.ResourceCPU]; !hasCPUResource {
		// Set CPU resource to 1 * 10**(-1) Cores, e.g. 100mC
		resources.Requests[corev1.ResourceCPU] = *resource.NewScaledQuantity(1, -1)
	}

	return resources
}

func (b *builder) oneAgentResource() corev1.ResourceRequirements {
	if b.hostInjectSpec == nil {
		return corev1.ResourceRequirements{}
	}

	return b.hostInjectSpec.OneAgentResources
}

func (b *builder) dnsPolicy() corev1.DNSPolicy {
	if b.hostInjectSpec != nil && b.hostInjectSpec.DNSPolicy != "" {
		return b.hostInjectSpec.DNSPolicy
	}

	return corev1.DNSClusterFirstWithHostNet
}

func (b *builder) volumeMounts() []corev1.VolumeMount {
	return prepareVolumeMounts(b.dk)
}

func (b *builder) volumes() []corev1.Volume {
	return prepareVolumes(b.dk)
}

func (b *builder) imagePullSecrets() []corev1.LocalObjectReference {
	if b.dk == nil {
		return []corev1.LocalObjectReference{}
	}

	return b.dk.ImagePullSecretReferences()
}

func (b *builder) securityContext() *corev1.SecurityContext {
	var securityContext corev1.SecurityContext
	if b.dk != nil && b.dk.OneAgent().IsReadOnlyFSSupported() {
		securityContext.RunAsNonRoot = ptr.To(true)
		securityContext.RunAsUser = ptr.To(int64(1000))
		securityContext.RunAsGroup = ptr.To(int64(1000))
		securityContext.ReadOnlyRootFilesystem = ptr.To(b.isRootFsReadonly())
	} else {
		securityContext.ReadOnlyRootFilesystem = ptr.To(false)
	}

	if b.dk != nil && b.dk.OneAgent().IsPrivilegedNeeded() {
		securityContext.Privileged = ptr.To(true)
	} else {
		securityContext.Capabilities = defaultSecurityContextCapabilities()

		if b.dk != nil {
			switch {
			case b.dk.OneAgent().IsHostMonitoringMode() && b.dk.Spec.OneAgent.HostMonitoring.SecCompProfile != "":
				secCompName := b.dk.Spec.OneAgent.HostMonitoring.SecCompProfile
				securityContext.SeccompProfile = &corev1.SeccompProfile{
					Type:             corev1.SeccompProfileTypeLocalhost,
					LocalhostProfile: &secCompName,
				}
			case b.dk.OneAgent().IsClassicFullStackMode() && b.dk.Spec.OneAgent.ClassicFullStack.SecCompProfile != "":
				secCompName := b.dk.Spec.OneAgent.ClassicFullStack.SecCompProfile
				securityContext.SeccompProfile = &corev1.SeccompProfile{
					Type:             corev1.SeccompProfileTypeLocalhost,
					LocalhostProfile: &secCompName,
				}
			case b.dk.OneAgent().IsCloudNativeFullstackMode() && b.dk.Spec.OneAgent.CloudNativeFullStack.SecCompProfile != "":
				secCompName := b.dk.Spec.OneAgent.CloudNativeFullStack.SecCompProfile
				securityContext.SeccompProfile = &corev1.SeccompProfile{
					Type:             corev1.SeccompProfileTypeLocalhost,
					LocalhostProfile: &secCompName,
				}
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

// getDefaultProbeFromStatus uses the docker HEALTHCHECK from status
func (b *builder) getDefaultProbeFromStatus() *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: b.dk.Status.OneAgent.Healthcheck.Test,
			},
		},
		InitialDelaySeconds: int32(b.dk.Status.OneAgent.Healthcheck.StartPeriod.Seconds()),
		PeriodSeconds:       int32(b.dk.Status.OneAgent.Healthcheck.Interval.Seconds()),
		TimeoutSeconds:      int32(b.dk.Status.OneAgent.Healthcheck.Timeout.Seconds()),
		FailureThreshold:    int32(b.dk.Status.OneAgent.Healthcheck.Retries), //nolint:gosec
		SuccessThreshold:    probeDefaultSuccessThreshold,
	}
}

// getReadinessProbe overrides the default HEALTHCHECK to ensure early readiness
func (b *builder) getReadinessProbe() *corev1.Probe {
	defaultProbe := b.getDefaultProbeFromStatus()
	if defaultProbe.InitialDelaySeconds > probeMaxInitialDelay {
		defaultProbe.InitialDelaySeconds = probeMaxInitialDelay
	}

	return defaultProbe
}

// isRootFsReadonly checks if the given version of the OneAgent supports the `ReadOnlyRootFilesystem` securityContext setting.
// if the version is not set, ie.: unknown, we  consider the OneAgent to support `ReadOnlyRootFilesystem`.
func (b *builder) isRootFsReadonly() bool {
	if b.dk != nil &&
		b.dk.OneAgent().IsReadOnlyFSSupported() &&
		b.dk.OneAgent().GetVersion() != "" &&
		b.dk.OneAgent().GetVersion() != string(status.CustomImageVersionSource) {
		agentSemver, err := dtversion.ToSemver(b.dk.OneAgent().GetVersion())
		if err != nil {
			log.Debug("Unable to determine OneAgent version to enable readonly pod filesystem, skipping", "version", b.dk.OneAgent().GetVersion(), "error", err.Error())

			return true
		}

		return semver.Compare(readOnlyRootFsConstraint, agentSemver) != 1 // if threshold <= agent-version
	}

	return true
}
