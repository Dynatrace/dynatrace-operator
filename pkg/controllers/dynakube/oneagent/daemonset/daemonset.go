// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package daemonset

import (
	"context"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8ssecuritycontext"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"golang.org/x/mod/semver"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	appArmorAnnotation                = corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + containerName
	appArmorUnconfined                = corev1.DeprecatedAppArmorBetaProfileNameUnconfined
	annotationTenantTokenHash         = api.InternalFlagPrefix + "tenant-token-hash"
	annotationEnableDaemonSetEviction = "cluster-autoscaler.kubernetes.io/enable-ds-eviction"
	AnnotationPGCHash                 = api.InternalFlagPrefix + "pgc-hash"
	AnnotationActiveGateTLSCertHash   = api.InternalFlagPrefix + "activegate-tls-cert-hash"
	AnnotationTrustedCAHash           = api.InternalFlagPrefix + "trusted-ca-hash"

	serviceAccountName = "dynatrace-dynakube-oneagent"

	// normal oneagent shutdown scenario with some extra time
	defaultTerminationGracePeriod = int64(80)

	hostRootVolumeName      = "host-root"
	hostRootVolumeMountPath = "/mnt/root"

	clusterCaCertVolumeName      = "dynatrace-cluster-ca"
	clusterCaCertVolumeMountPath = "/mnt/dynatrace/certs"

	activeGateCaCertVolumeName      = "active-gate-ca"
	activeGateCaCertVolumeMountPath = "/mnt/dynatrace/certs/activegate"

	csiStorageVolumeName  = "osagent-storage"
	csiStorageVolumeMount = "/mnt/volume_storage_mount"

	storageVolumeName = "volume-storage"

	nodeMetadataVolumeName = "dynatrace-operator"
	nodeMetadataFilename   = "dt_node_metadata.properties"
	nodeMetadataFolderPath = "/var/lib/dynatrace/enrichment"

	containerName = "dynatrace-oneagent"

	inframonHostIDSource = "k8s-node-name"
	classicHostIDSource  = "auto"

	probeMaxInitialDelay         = int32(90)
	probeDefaultSuccessThreshold = int32(1)

	readOnlyRootFsConstraint = "v1.291"

	userGroupID int64 = 1000

	pgcSecretVolumeName = "bootstrapper-pgc"
)

var nodeMetadataFilePath = filepath.Join(nodeMetadataFolderPath, nodeMetadataFilename)

type hostMonitoring struct {
	builder
}

type classicFullStack struct {
	builder
}

type builder struct {
	dk                     *dynakube.DynaKube
	hostInjectSpec         *oneagent.HostInjectSpec
	clusterID              string
	processGroupConfigHash string
}

type Builder interface {
	BuildDaemonSet(ctx context.Context) (*appsv1.DaemonSet, error)
}

func NewHostMonitoring(dk *dynakube.DynaKube, clusterID, processGroupConfigHash string) Builder {
	return &hostMonitoring{
		builder{
			dk:                     dk,
			hostInjectSpec:         dk.Spec.OneAgent.HostMonitoring,
			clusterID:              clusterID,
			processGroupConfigHash: processGroupConfigHash,
		},
	}
}

func NewCloudNativeFullStack(dk *dynakube.DynaKube, clusterID, processGroupConfigHash string) Builder {
	return &hostMonitoring{
		builder{
			dk:                     dk,
			hostInjectSpec:         &dk.Spec.OneAgent.CloudNativeFullStack.HostInjectSpec,
			clusterID:              clusterID,
			processGroupConfigHash: processGroupConfigHash,
		},
	}
}

func NewClassicFullStack(dk *dynakube.DynaKube, clusterID string) Builder {
	return &classicFullStack{
		builder{
			dk:             dk,
			hostInjectSpec: dk.Spec.OneAgent.ClassicFullStack,
			clusterID:      clusterID,
		},
	}
}

func (hm *hostMonitoring) BuildDaemonSet(ctx context.Context) (*appsv1.DaemonSet, error) {
	daemonSet, err := hm.builder.BuildDaemonSet(ctx)
	if err != nil {
		return nil, err
	}

	daemonSet.Name = hm.dk.OneAgent().GetDaemonsetName()

	if len(daemonSet.Spec.Template.Spec.Containers) > 0 {
		hm.appendInfraMonEnvVars(daemonSet)
	}

	return daemonSet, nil
}

func (classic *classicFullStack) BuildDaemonSet(ctx context.Context) (*appsv1.DaemonSet, error) {
	result, err := classic.builder.BuildDaemonSet(ctx)
	if err != nil {
		return nil, err
	}

	result.Name = classic.dk.OneAgent().GetDaemonsetName()

	return result, nil
}

func (b *builder) BuildDaemonSet(ctx context.Context) (*appsv1.DaemonSet, error) {
	ctx, _ = logd.NewFromContext(ctx, "daemonset")
	dk := b.dk

	podSpec, err := b.podSpec(ctx)
	if err != nil {
		return nil, err
	}

	labels := k8slabel.New(k8slabel.OneAgent, dk.Name, dk.OneAgent().GetVersion())

	// operator labels are merged last, so custom labels can't override them
	dsLabels := maputils.MergeMap(b.hostInjectSpec.Labels, labels.AsMap())
	podLabels := maputils.MergeMap(b.hostInjectSpec.Labels, labels.AsSelector())

	templateAnnotations := map[string]string{
		appArmorAnnotation:                appArmorUnconfined,
		webhook.AnnotationDynatraceInject: "false",
		annotationTenantTokenHash:         dk.Status.OneAgent.ConnectionInfo.TenantTokenHash,
		annotationEnableDaemonSetEviction: "false",
	}

	if bootstrapperconfig.NeedsPGC(dk) {
		if b.processGroupConfigHash != "" {
			templateAnnotations[AnnotationPGCHash] = b.processGroupConfigHash
		}
	}

	templateAnnotations = k8ssecuritycontext.RemoveAppArmorAnnotation(maputils.MergeMap(templateAnnotations, b.hostInjectSpec.Annotations), containerName, initContainerName)

	result := &appsv1.DaemonSet{
		Name:        dk.Name,
		Namespace:   dk.Namespace,
		Labels:      dsLabels,
		Annotations: map[string]string{},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.AsSelector(),
			},
			Template: corev1.PodTemplateSpec{
				Labels:      podLabels,
				Annotations: templateAnnotations,
				Spec:        podSpec,
			},
			UpdateStrategy: b.updateStrategy(),
		},
	}

	return result, nil
}

func (b *builder) podSpec(ctx context.Context) (corev1.PodSpec, error) {
	ids := b.runAsIDs(ctx)
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
		InitContainers: []corev1.Container{b.initContainerSpec(ids)},
		Containers: []corev1.Container{{
			Args:            arguments,
			Env:             environmentVariables,
			Image:           b.immutableOneAgentImage(),
			ImagePullPolicy: b.dk.OneAgent().GetImagePullPolicy(),
			Name:            containerName,
			Resources:       resources,
			SecurityContext: b.securityContext(ctx, ids),
			VolumeMounts:    volumeMounts,
		}},
		ImagePullSecrets:              imagePullSecrets,
		HostNetwork:                   true,
		HostPID:                       true,
		HostIPC:                       false,
		SecurityContext:               b.podSecurityContext(ids),
		NodeSelector:                  b.nodeSelector(),
		PriorityClassName:             b.priorityClassName(),
		ServiceAccountName:            serviceAccountName,
		Tolerations:                   b.tolerations(),
		DNSPolicy:                     dnsPolicy,
		Volumes:                       volumes,
		Affinity:                      affinity,
		TerminationGracePeriodSeconds: new(defaultTerminationGracePeriod),
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

func (b *builder) updateStrategy() appsv1.DaemonSetUpdateStrategy {
	maxUnavailable := intstr.FromInt(b.dk.FF().GetOneAgentMaxUnavailable()) //nolint:staticcheck

	us := appsv1.DaemonSetUpdateStrategy{
		RollingUpdate: &appsv1.RollingUpdateDaemonSet{
			MaxUnavailable: &maxUnavailable,
		},
	}

	if b.hostInjectSpec.RollingUpdate != nil {
		us.RollingUpdate = b.hostInjectSpec.RollingUpdate
	}

	return us
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
	return prepareVolumeMounts(b.dk, b.processGroupConfigHash)
}

func (b *builder) volumes() []corev1.Volume {
	return prepareVolumes(b.dk, b.processGroupConfigHash)
}

func (b *builder) imagePullSecrets() []corev1.LocalObjectReference {
	if b.dk == nil {
		return []corev1.LocalObjectReference{}
	}

	return b.dk.ImagePullSecretReferences()
}

func (b *builder) securityContext(ctx context.Context, ids runAs) *corev1.SecurityContext {
	sc := &corev1.SecurityContext{}
	if b.dk == nil || b.hostInjectSpec == nil {
		return sc
	}

	if ids.nonRoot {
		sc.RunAsNonRoot = new(true)
		sc.RunAsUser = ids.user
		sc.RunAsGroup = ids.group
	}

	if b.dk.OneAgent().IsReadOnlyFSSupported() {
		sc.ReadOnlyRootFilesystem = new(b.isRootFsReadonly(ctx))
	} else {
		sc.ReadOnlyRootFilesystem = new(false)
	}

	if b.dk.OneAgent().IsPrivilegedNeeded() {
		sc.Privileged = new(true)
	} else {
		sc.Capabilities = defaultSecurityContextCapabilities()

		if secompName := b.dk.OneAgent().GetSecCompProfile(); secompName != "" {
			sc.SeccompProfile = &corev1.SeccompProfile{
				Type:             corev1.SeccompProfileTypeLocalhost,
				LocalhostProfile: &secompName,
			}
		}
	}

	annotations := maputils.MergeMap(map[string]string{appArmorAnnotation: appArmorUnconfined}, b.hostInjectSpec.Annotations)
	sc.AppArmorProfile = k8ssecuritycontext.GetAppArmorProfile(annotations, containerName)

	return sc
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
		Exec: &corev1.ExecAction{
			Command: b.dk.Status.OneAgent.Healthcheck.Test,
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
func (b *builder) isRootFsReadonly(ctx context.Context) bool {
	log := logd.FromContext(ctx)

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

func (b *builder) podSecurityContext(ids runAs) *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		FSGroup: ids.group,
	}
}

// runAs holds the user and group the OneAgent pod runs as, and whether the OneAgent container runs as non-root.
type runAs struct {
	user    *int64
	group   *int64
	nonRoot bool
}

// runAsIDs returns the IDs the OneAgent pod runs as.
// The IDs are the default, unless the classic non-root feature flag configures custom IDs for classic full stack.
func (b *builder) runAsIDs(ctx context.Context) runAs {
	if b.dk == nil {
		return runAs{}
	}

	if b.dk.OneAgent().IsReadOnlyFSSupported() {
		return runAs{user: new(userGroupID), group: new(userGroupID), nonRoot: true}
	}

	if u, g, enabled := b.dk.FF().GetClassicOneAgentNonRoot(ctx); enabled && b.dk.OneAgent().IsClassicFullStackMode() {
		return runAs{user: new(u), group: new(g), nonRoot: true}
	}

	return runAs{user: new(userGroupID), group: new(userGroupID)}
}
