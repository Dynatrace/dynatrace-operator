package statefulset

import (
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder/modifiers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const defaultEnvPriority = prioritymap.DefaultPriority
const customEnvPriority = prioritymap.HighPriority

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

	if err := setHash(&sts); err != nil {
		return nil, err
	}

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

	if statefulSetBuilder.dynakube.FeatureActiveGateAppArmor() {
		sts.Spec.Template.ObjectMeta.Annotations[consts.AnnotationActiveGateContainerAppArmor] = "runtime/default"
	}

	return sts
}

func (statefulSetBuilder Builder) getBaseObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        statefulSetBuilder.dynakube.Name + "-" + statefulSetBuilder.capability.ShortName(),
		Namespace:   statefulSetBuilder.dynakube.Namespace,
		Annotations: map[string]string{},
	}
}

func (statefulSetBuilder Builder) getBaseSpec() appsv1.StatefulSetSpec {
	return appsv1.StatefulSetSpec{
		Replicas:            &statefulSetBuilder.capability.Properties().Replicas,
		PodManagementPolicy: appsv1.ParallelPodManagement,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					consts.AnnotationActiveGateConfigurationHash: statefulSetBuilder.configHash,
				},
			},
		},
	}
}

func (statefulSetBuilder Builder) addLabels(sts *appsv1.StatefulSet) {
	appLabels := statefulSetBuilder.buildAppLabels()
	sts.ObjectMeta.Labels = appLabels.BuildLabels()
	sts.Spec.Selector = &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}
	sts.Spec.Template.ObjectMeta.Labels = maputils.MergeMap(statefulSetBuilder.capability.Properties().Labels, appLabels.BuildLabels())
}

func (statefulSetBuilder Builder) buildAppLabels() *labels.AppLabels {
	version := statefulSetBuilder.dynakube.Status.ActiveGate.Version

	return labels.NewAppLabels(labels.ActiveGateComponentLabel, statefulSetBuilder.dynakube.Name, statefulSetBuilder.capability.ShortName(), version)
}

func (statefulSetBuilder Builder) addUserAnnotations(sts *appsv1.StatefulSet) {
	sts.ObjectMeta.Annotations = maputils.MergeMap(sts.ObjectMeta.Annotations, statefulSetBuilder.dynakube.Spec.ActiveGate.Annotations)
	sts.Spec.Template.ObjectMeta.Annotations = maputils.MergeMap(sts.Spec.Template.ObjectMeta.Annotations, statefulSetBuilder.dynakube.Spec.ActiveGate.Annotations)
}

func (statefulSetBuilder Builder) addTemplateSpec(sts *appsv1.StatefulSet) {
	podSpec := corev1.PodSpec{
		Containers:         statefulSetBuilder.buildBaseContainer(),
		NodeSelector:       statefulSetBuilder.capability.Properties().NodeSelector,
		ServiceAccountName: statefulSetBuilder.dynakube.ActiveGate().GetServiceAccountName(),
		Affinity:           nodeAffinity(),
		Tolerations:        statefulSetBuilder.capability.Properties().Tolerations,
		SecurityContext: &corev1.PodSecurityContext{
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
		ImagePullSecrets:  statefulSetBuilder.dynakube.ImagePullSecretReferences(),
		PriorityClassName: statefulSetBuilder.dynakube.Spec.ActiveGate.PriorityClassName,
		DNSPolicy:         statefulSetBuilder.dynakube.Spec.ActiveGate.DNSPolicy,

		TopologySpreadConstraints: statefulSetBuilder.buildTopologySpreadConstraints(statefulSetBuilder.capability),
	}
	sts.Spec.Template.Spec = podSpec
}

func (statefulSetBuilder Builder) buildTopologySpreadConstraints(capability capability.Capability) []corev1.TopologySpreadConstraint {
	if len(capability.Properties().TopologySpreadConstraints) > 0 {
		return capability.Properties().TopologySpreadConstraints
	}

	return statefulSetBuilder.defaultTopologyConstraints()
}

func (statefulSetBuilder Builder) defaultTopologyConstraints() []corev1.TopologySpreadConstraint {
	appLabels := statefulSetBuilder.buildAppLabels()

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
					Port:   intstr.IntOrString{IntVal: consts.HttpsContainerPort},
					Scheme: "HTTPS",
				},
			},
			InitialDelaySeconds: 90,
			PeriodSeconds:       15,
			FailureThreshold:    3,
			TimeoutSeconds:      2,
		},
		SecurityContext: modifiers.GetSecurityContext(false),
	}

	return []corev1.Container{container}
}

func (statefulSetBuilder Builder) buildResources() corev1.ResourceRequirements {
	return statefulSetBuilder.capability.Properties().Resources
}

func (statefulSetBuilder Builder) buildCommonEnvs() []corev1.EnvVar {
	prioritymap.Append(statefulSetBuilder.envMap, []corev1.EnvVar{
		{Name: consts.EnvDtCapabilities, Value: statefulSetBuilder.capability.ArgName()},
		{Name: consts.EnvDtIdSeedNamespace, Value: statefulSetBuilder.dynakube.Namespace},
		{Name: consts.EnvDtIdSeedClusterId, Value: string(statefulSetBuilder.kubeUID)},
		{Name: deploymentmetadata.EnvDtDeploymentMetadata, ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: deploymentmetadata.GetDeploymentMetadataConfigMapName(statefulSetBuilder.dynakube.Name),
				},
				Key:      deploymentmetadata.ActiveGateMetadataKey,
				Optional: address.Of(false),
			},
		}},
	})

	if statefulSetBuilder.capability.Properties().Group != "" {
		prioritymap.Append(statefulSetBuilder.envMap, corev1.EnvVar{Name: consts.EnvDtGroup, Value: statefulSetBuilder.capability.Properties().Group})
	}

	if statefulSetBuilder.dynakube.Spec.NetworkZone != "" {
		prioritymap.Append(statefulSetBuilder.envMap, corev1.EnvVar{Name: consts.EnvDtNetworkZone, Value: statefulSetBuilder.dynakube.Spec.NetworkZone})
	}

	if statefulSetBuilder.dynakube.ActiveGate().IsMetricsIngestEnabled() {
		prioritymap.Append(statefulSetBuilder.envMap, corev1.EnvVar{Name: consts.EnvDtHttpPort, Value: strconv.Itoa(consts.HttpContainerPort)})
	}

	prioritymap.Append(statefulSetBuilder.envMap, statefulSetBuilder.capability.Properties().Env, prioritymap.WithPriority(customEnvPriority))

	return statefulSetBuilder.envMap.AsEnvVars()
}

func nodeAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: node.AffinityNodeRequirementForSupportedArches(),
					},
				},
			},
		},
	}
}

func setHash(sts *appsv1.StatefulSet) error {
	hash, err := hasher.GenerateHash(sts)
	if err != nil {
		return errors.WithStack(err)
	}

	sts.ObjectMeta.Annotations[hasher.AnnotationHash] = hash

	return nil
}
