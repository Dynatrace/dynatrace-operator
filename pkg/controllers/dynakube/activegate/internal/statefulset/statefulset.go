package statefulset

import (
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder/modifiers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	defaultEnvPriority = prioritymap.DefaultPriority
	customEnvPriority  = prioritymap.HighPriority
)

type Builder struct {
	capability capability.Capability
	envMap     *prioritymap.Map
	kubeUID    types.UID
	configHash string
	dynakube   dynakube.DynaKube
}

func NewStatefulSetBuilder(kubeUID types.UID, configHash string, dk dynakube.DynaKube, capability capability.Capability) Builder {
	return Builder{
		kubeUID:    kubeUID,
		configHash: configHash,
		dynakube:   dk,
		capability: capability,
		envMap:     prioritymap.New(prioritymap.WithPriority(defaultEnvPriority)),
	}
}

func (statefulSetBuilder Builder) CreateStatefulSet(mods []builder.Modifier) (*appsv1.StatefulSet, error) {
	activeGateBuilder := builder.NewBuilder(statefulSetBuilder.getBase())

	if len(mods) == 0 {
		mods = modifiers.GenerateAllModifiers(statefulSetBuilder.dynakube, statefulSetBuilder.capability, statefulSetBuilder.envMap)
	}

	sts, _ := activeGateBuilder.AddModifier(mods...).Build()

	return &sts, nil
}

func (statefulSetBuilder Builder) getBase() appsv1.StatefulSet {
	var sts appsv1.StatefulSet

	sts.Kind = "StatefulSet"
	sts.APIVersion = "apps/v1"
	sts.ObjectMeta = statefulSetBuilder.getBaseObjectMeta()
	sts.Spec = statefulSetBuilder.getBaseSpec()
	statefulSetBuilder.addUserAnnotations(&sts)
	statefulSetBuilder.addLabels(&sts)
	statefulSetBuilder.addTemplateSpec(&sts)
	statefulSetBuilder.addPersistentVolumeClaim(&sts)

	if statefulSetBuilder.dynakube.FF().IsActiveGateAppArmor() {
		sts.Spec.Template.Annotations[consts.AnnotationActiveGateContainerAppArmor] = "runtime/default"
	}

	return sts
}

func (statefulSetBuilder Builder) getBaseObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        statefulSetBuilder.dynakube.Name + "-" + consts.MultiActiveGateName,
		Namespace:   statefulSetBuilder.dynakube.Namespace,
		Annotations: map[string]string{},
	}
}

func (statefulSetBuilder Builder) getBaseSpec() appsv1.StatefulSetSpec {
	return appsv1.StatefulSetSpec{
		Replicas:            statefulSetBuilder.capability.Properties().Replicas,
		PodManagementPolicy: appsv1.ParallelPodManagement,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					consts.AnnotationActiveGateConfigurationHash: statefulSetBuilder.configHash,
					consts.AnnotationActiveGateTenantTokenHash:   statefulSetBuilder.dynakube.Status.ActiveGate.ConnectionInfo.TenantTokenHash,
					exp.InjectionSplitMounts:                     "true",
				},
			},
		},
	}
}

func (statefulSetBuilder Builder) addLabels(sts *appsv1.StatefulSet) {
	appLabels := statefulSetBuilder.buildAppLabels()
	sts.Labels = appLabels.BuildLabels()
	sts.Spec.Selector = &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}
	sts.Spec.Template.Labels = maputils.MergeMap(statefulSetBuilder.capability.Properties().Labels, appLabels.BuildLabels())
}

func (statefulSetBuilder Builder) buildAppLabels() *labels.AppLabels {
	version := statefulSetBuilder.dynakube.Status.ActiveGate.Version

	return labels.NewAppLabels(labels.ActiveGateComponentLabel, statefulSetBuilder.dynakube.Name, consts.MultiActiveGateName, version)
}

func (statefulSetBuilder Builder) addUserAnnotations(sts *appsv1.StatefulSet) {
	sts.Annotations = maputils.MergeMap(sts.Annotations, statefulSetBuilder.dynakube.Spec.ActiveGate.Annotations)
	sts.Spec.Template.Annotations = maputils.MergeMap(sts.Spec.Template.Annotations, statefulSetBuilder.dynakube.Spec.ActiveGate.Annotations)
}

func (statefulSetBuilder Builder) addTemplateSpec(sts *appsv1.StatefulSet) {
	podSpec := corev1.PodSpec{
		Containers:                    statefulSetBuilder.buildBaseContainer(),
		NodeSelector:                  statefulSetBuilder.capability.Properties().NodeSelector,
		ServiceAccountName:            statefulSetBuilder.dynakube.ActiveGate().GetServiceAccountName(),
		Affinity:                      statefulSetBuilder.nodeAffinity(),
		Tolerations:                   statefulSetBuilder.capability.Properties().Tolerations,
		SecurityContext:               statefulSetBuilder.buildPodSecurityContext(),
		ImagePullSecrets:              statefulSetBuilder.dynakube.ImagePullSecretReferences(),
		PriorityClassName:             statefulSetBuilder.dynakube.Spec.ActiveGate.PriorityClassName,
		DNSPolicy:                     statefulSetBuilder.dynakube.Spec.ActiveGate.DNSPolicy,
		TerminationGracePeriodSeconds: statefulSetBuilder.dynakube.ActiveGate().GetTerminationGracePeriodSeconds(),
		TopologySpreadConstraints:     statefulSetBuilder.buildTopologySpreadConstraints(statefulSetBuilder.capability),
		Volumes:                       statefulSetBuilder.buildVolumes(),
		AutomountServiceAccountToken:  ptr.To(false),
	}
	sts.Spec.Template.Spec = podSpec
}

func (statefulSetBuilder Builder) buildTopologySpreadConstraints(capability capability.Capability) []corev1.TopologySpreadConstraint {
	if len(capability.Properties().TopologySpreadConstraints) > 0 {
		return capability.Properties().TopologySpreadConstraints
	}

	return statefulSetBuilder.defaultTopologyConstraints()
}

func (statefulSetBuilder Builder) buildVolumes() []corev1.Volume {
	volumes := []corev1.Volume{}

	if statefulSetBuilder.dynakube.Spec.ActiveGate.VolumeClaimTemplate == nil {
		if !isDefaultPVCNeeded(statefulSetBuilder.dynakube) {
			volumes = append(volumes, corev1.Volume{
				Name: consts.GatewayTmpVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})
		}
	}

	return volumes
}

func (statefulSetBuilder Builder) buildVolumeMounts() []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      consts.GatewayTmpVolumeName,
		MountPath: consts.GatewayTmpMountPoint,
	})

	return volumeMounts
}

func (statefulSetBuilder Builder) buildPodSecurityContext() *corev1.PodSecurityContext {
	sc := corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	if !statefulSetBuilder.dynakube.Spec.ActiveGate.UseEphemeralVolume {
		sc.FSGroup = ptr.To(consts.DockerImageGroup)
	}

	return &sc
}

func (statefulSetBuilder Builder) defaultTopologyConstraints() []corev1.TopologySpreadConstraint {
	appLabels := statefulSetBuilder.buildAppLabels()
	nodeInclusionPolicyHonor := corev1.NodeInclusionPolicyHonor

	return []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       "topology.kubernetes.io/zone",
			WhenUnsatisfiable: "ScheduleAnyway",
			LabelSelector:     &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()},
		},
		{
			MaxSkew:           1,
			TopologyKey:       "kubernetes.io/hostname",
			WhenUnsatisfiable: "DoNotSchedule",
			NodeTaintsPolicy:  &nodeInclusionPolicyHonor,
			LabelSelector:     &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()},
		},
	}
}

func (statefulSetBuilder Builder) buildBaseContainer() []corev1.Container {
	container := corev1.Container{
		Name:            consts.ActiveGateContainerName,
		Image:           statefulSetBuilder.dynakube.ActiveGate().GetImage(),
		Resources:       statefulSetBuilder.buildResources(),
		Env:             statefulSetBuilder.buildCommonEnvs(),
		ImagePullPolicy: corev1.PullAlways,
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/rest/health",
					Port:   intstr.IntOrString{IntVal: consts.HTTPSContainerPort},
					Scheme: "HTTPS",
				},
			},
			InitialDelaySeconds: 90,
			PeriodSeconds:       15,
			FailureThreshold:    3,
			TimeoutSeconds:      2,
		},
		SecurityContext: modifiers.GetSecurityContext(false),
		VolumeMounts:    statefulSetBuilder.buildVolumeMounts(),
	}

	return []corev1.Container{container}
}

func (statefulSetBuilder Builder) buildResources() corev1.ResourceRequirements {
	return statefulSetBuilder.capability.Properties().Resources
}

func (statefulSetBuilder Builder) buildCommonEnvs() []corev1.EnvVar {
	prioritymap.Append(statefulSetBuilder.envMap, []corev1.EnvVar{
		{Name: consts.EnvDtCapabilities, Value: statefulSetBuilder.capability.ArgName()},
		{Name: consts.EnvDtIDSeedNamespace, Value: statefulSetBuilder.dynakube.Namespace},
		{Name: consts.EnvDtIDSeedClusterID, Value: string(statefulSetBuilder.kubeUID)},
		{Name: deploymentmetadata.EnvDtDeploymentMetadata, ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: deploymentmetadata.GetDeploymentMetadataConfigMapName(statefulSetBuilder.dynakube.Name),
				},
				Key:      deploymentmetadata.ActiveGateMetadataKey,
				Optional: ptr.To(false),
			},
		}},
		{Name: consts.EnvDtHTTPPort, Value: strconv.Itoa(consts.HTTPContainerPort)},
	})

	if statefulSetBuilder.capability.Properties().Group != "" {
		prioritymap.Append(statefulSetBuilder.envMap, corev1.EnvVar{Name: consts.EnvDtGroup, Value: statefulSetBuilder.capability.Properties().Group})
	}

	if statefulSetBuilder.dynakube.Spec.NetworkZone != "" {
		prioritymap.Append(statefulSetBuilder.envMap, corev1.EnvVar{Name: consts.EnvDtNetworkZone, Value: statefulSetBuilder.dynakube.Spec.NetworkZone})
	}

	prioritymap.Append(statefulSetBuilder.envMap, statefulSetBuilder.capability.Properties().Env, prioritymap.WithPriority(customEnvPriority))

	return statefulSetBuilder.envMap.AsEnvVars()
}

func (statefulSetBuilder Builder) nodeAffinity() *corev1.Affinity {
	var affinity corev1.Affinity
	if statefulSetBuilder.dynakube.Status.ActiveGate.Source == status.TenantRegistryVersionSource || statefulSetBuilder.dynakube.Status.ActiveGate.Source == status.CustomVersionVersionSource {
		affinity = node.AMDOnlyAffinity()
	} else {
		affinity = node.Affinity()
	}

	return &affinity
}

func isDefaultPVCNeeded(dk dynakube.DynaKube) bool {
	return dk.TelemetryIngest().IsEnabled() && !dk.Spec.ActiveGate.UseEphemeralVolume
}

func (statefulSetBuilder Builder) addPersistentVolumeClaim(sts *appsv1.StatefulSet) {
	if statefulSetBuilder.dynakube.Spec.ActiveGate.VolumeClaimTemplate != nil {
		// validation webhook ensures that statefulSetBuilder.dynakube.Spec.ActiveGate.UseEphemeralVolume is false at this point
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: consts.GatewayTmpVolumeName,
				},
				Spec: *statefulSetBuilder.dynakube.Spec.ActiveGate.VolumeClaimTemplate,
			},
		}
		sts.Spec.PersistentVolumeClaimRetentionPolicy = defaultPVCRetentionPolicy()
	} else if isDefaultPVCNeeded(statefulSetBuilder.dynakube) {
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: consts.GatewayTmpVolumeName,
				},
				Spec: defaultPVCSpec(),
			},
		}
		sts.Spec.PersistentVolumeClaimRetentionPolicy = defaultPVCRetentionPolicy()
	}

	statefulset.SetPVCAnnotation()(sts)
}

func defaultPVCSpec() corev1.PersistentVolumeClaimSpec {
	return corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{
			corev1.ReadWriteOnce,
		},
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}
}

func defaultPVCRetentionPolicy() *appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy {
	return &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
		WhenDeleted: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
		WhenScaled:  appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
	}
}
