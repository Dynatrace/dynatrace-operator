package statefulset

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder/modifiers"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Builder struct {
	kubeUID    types.UID
	configHash string
	dynakube   dynatracev1beta1.DynaKube
	capability capability.Capability
}

func NewStatefulSetBuilder(kubeUID types.UID, configHash string, dynakube dynatracev1beta1.DynaKube, capability capability.Capability) Builder {
	return Builder{
		kubeUID:    kubeUID,
		configHash: configHash,
		dynakube:   dynakube,
		capability: capability,
	}
}

func (statefulSetBuilder Builder) CreateStatefulSet(mods []builder.Modifier) (*appsv1.StatefulSet, error) {
	activeGateBuilder := builder.NewBuilder(statefulSetBuilder.getBase())
	if len(mods) == 0 {
		mods = modifiers.GenerateAllModifiers(statefulSetBuilder.dynakube, statefulSetBuilder.capability)
	}
	sts, _ := activeGateBuilder.AddModifier(mods...).Build()

	if err := setHash(&sts); err != nil {
		return nil, err
	}

	return &sts, nil
}

func (statefulSetBuilder Builder) getBase() appsv1.StatefulSet {
	var sts appsv1.StatefulSet
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
		Replicas:            statefulSetBuilder.capability.Properties().Replicas,
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
	versionLabelValue := statefulSetBuilder.dynakube.Status.ActiveGate.Version
	if statefulSetBuilder.dynakube.CustomActiveGateImage() != "" {
		versionLabelValue = kubeobjects.CustomImageLabelValue
	}
	appLabels := kubeobjects.NewAppLabels(kubeobjects.ActiveGateComponentLabel, statefulSetBuilder.dynakube.Name, statefulSetBuilder.capability.ShortName(), versionLabelValue)

	sts.ObjectMeta.Labels = appLabels.BuildLabels()
	sts.Spec.Selector = &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}
	sts.Spec.Template.ObjectMeta.Labels = kubeobjects.MergeMap(statefulSetBuilder.capability.Properties().Labels, appLabels.BuildLabels())
}

func (statefulSetBuilder Builder) addUserAnnotations(sts *appsv1.StatefulSet) {
	sts.ObjectMeta.Annotations = kubeobjects.MergeMap(sts.ObjectMeta.Annotations, statefulSetBuilder.dynakube.Spec.ActiveGate.Annotations)
	sts.Spec.Template.ObjectMeta.Annotations = kubeobjects.MergeMap(sts.Spec.Template.ObjectMeta.Annotations, statefulSetBuilder.dynakube.Spec.ActiveGate.Annotations)
}

func (statefulSetBuilder Builder) addTemplateSpec(sts *appsv1.StatefulSet) {
	podSpec := corev1.PodSpec{
		Containers:         statefulSetBuilder.buildBaseContainer(),
		NodeSelector:       statefulSetBuilder.capability.Properties().NodeSelector,
		ServiceAccountName: statefulSetBuilder.dynakube.ActiveGateServiceAccountName(),
		Affinity:           nodeAffinity(),
		Tolerations:        buildTolerations(statefulSetBuilder.capability),
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: statefulSetBuilder.dynakube.PullSecret()},
		},
		PriorityClassName:         statefulSetBuilder.dynakube.Spec.ActiveGate.PriorityClassName,
		DNSPolicy:                 statefulSetBuilder.dynakube.Spec.ActiveGate.DNSPolicy,
		TopologySpreadConstraints: statefulSetBuilder.capability.Properties().TopologySpreadConstraints,
	}
	sts.Spec.Template.Spec = podSpec
}

func buildTolerations(capability capability.Capability) []corev1.Toleration {
	tolerations := make([]corev1.Toleration, len(capability.Properties().Tolerations))
	copy(tolerations, capability.Properties().Tolerations)
	tolerations = append(tolerations, kubeobjects.TolerationForAmd()...)
	return tolerations
}

func (statefulSetBuilder Builder) buildBaseContainer() []corev1.Container {
	container := corev1.Container{
		Name:            consts.ActiveGateContainerName,
		Image:           statefulSetBuilder.dynakube.ActiveGateImage(),
		Resources:       statefulSetBuilder.buildResources(),
		Env:             statefulSetBuilder.buildCommonEnvs(),
		ImagePullPolicy: corev1.PullAlways,
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/rest/health",
					Port:   intstr.IntOrString{IntVal: 9999},
					Scheme: "HTTPS",
				},
			},
			InitialDelaySeconds: 90,
			PeriodSeconds:       15,
			FailureThreshold:    3,
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged:               address.Of(false),
			AllowPrivilegeEscalation: address.Of(false),
			RunAsNonRoot:             address.Of(true),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{
					"ALL",
				},
			},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
	}

	return []corev1.Container{container}
}

func (statefulSetBuilder Builder) buildResources() corev1.ResourceRequirements {
	if statefulSetBuilder.dynakube.IsSyntheticActiveGateEnabled() {
		return modifiers.ActiveGateResourceRequirements
	} else {
		return statefulSetBuilder.capability.Properties().Resources
	}
}

func (statefulSetBuilder Builder) buildCommonEnvs() []corev1.EnvVar {
	envs := []corev1.EnvVar{
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
	}
	envs = append(envs, statefulSetBuilder.capability.Properties().Env...)

	if statefulSetBuilder.capability.Properties().Group != "" {
		envs = append(envs, corev1.EnvVar{Name: consts.EnvDtGroup, Value: statefulSetBuilder.capability.Properties().Group})
	}
	if statefulSetBuilder.dynakube.Spec.NetworkZone != "" {
		envs = append(envs, corev1.EnvVar{Name: consts.EnvDtNetworkZone, Value: statefulSetBuilder.dynakube.Spec.NetworkZone})
	}

	return envs
}

func nodeAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: kubeobjects.AffinityNodeRequirement(),
					},
				},
			},
		},
	}
}

func setHash(sts *appsv1.StatefulSet) error {
	hash, err := kubeobjects.GenerateHash(sts)
	if err != nil {
		return errors.WithStack(err)
	}
	sts.ObjectMeta.Annotations[kubeobjects.AnnotationHash] = hash
	return nil
}
