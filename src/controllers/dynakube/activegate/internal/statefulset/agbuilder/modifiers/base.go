package modifiers

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	agbuilderTypes "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder/internal/types"
	"github.com/Dynatrace/dynatrace-operator/src/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func NewBaseModifier(kubeUID types.UID, dynakube dynatracev1beta1.DynaKube, capability capability.Capability) agbuilderTypes.Modifier {
	return BaseModifier{
		kubeUID:    kubeUID,
		dynakube:   dynakube,
		capability: capability,
	}
}

// Sets the properties that are common for all Capabilities
// <probably should be moved/merged into the statefulSetBuilder>
type BaseModifier struct {
	kubeUID    types.UID
	dynakube   dynatracev1beta1.DynaKube
	capability capability.Capability
}

func (mod BaseModifier) Modify(sts *appsv1.StatefulSet) {
	mod.addLabels(sts)
	mod.addTemplateSpec(sts)

	if mod.dynakube.FeatureActiveGateAppArmor() { // maybe security modifier ?
		sts.Spec.Template.ObjectMeta.Annotations[consts.AnnotationActiveGateContainerAppArmor] = "runtime/default"
	}
}

func (mod BaseModifier) addLabels(sts *appsv1.StatefulSet) {
	versionLabelValue := mod.dynakube.Status.ActiveGate.Version
	if mod.dynakube.CustomActiveGateImage() != "" {
		versionLabelValue = kubeobjects.CustomImageLabelValue
	}
	appLabels := kubeobjects.NewAppLabels(kubeobjects.ActiveGateComponentLabel, mod.dynakube.Name, mod.capability.ShortName(), versionLabelValue)

	sts.ObjectMeta.Labels = appLabels.BuildLabels()
	sts.Spec.Selector = &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}
	sts.Spec.Template.ObjectMeta.Labels = kubeobjects.MergeLabels(mod.capability.Properties().Labels, appLabels.BuildLabels())

}

func (mod BaseModifier) addTemplateSpec(sts *appsv1.StatefulSet) {
	podSpec := corev1.PodSpec{
		Containers:         mod.buildBaseContainer(),
		InitContainers:     mod.buildInitContainers(),
		NodeSelector:       mod.capability.Properties().NodeSelector,
		ServiceAccountName: mod.buildServiceAccountName(),
		Affinity:           nodeAffinity(),
		Tolerations:        mod.capability.Properties().Tolerations,
		Volumes:            mod.capability.Volumes(),
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: mod.dynakube.PullSecret()},
		},
		PriorityClassName:         mod.dynakube.Spec.ActiveGate.PriorityClassName,
		DNSPolicy:                 mod.dynakube.Spec.ActiveGate.DNSPolicy,
		TopologySpreadConstraints: mod.capability.Properties().TopologySpreadConstraints,
	}
	sts.Spec.Template.Spec = podSpec
}

func (mod BaseModifier) buildBaseContainer() []corev1.Container {
	container := corev1.Container{
		Name:            consts.ActiveGateContainerName,
		Image:           mod.dynakube.ActiveGateImage(),
		Resources:       mod.capability.Properties().Resources,
		Env:             mod.buildCommonEnvs(),
		VolumeMounts:    mod.capability.ContainerVolumeMounts(),
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
	if mod.capability.Config().SetReadinessPort {
		container.ReadinessProbe.HTTPGet.Port = intstr.FromString(consts.HttpsServicePortName)
	}
	if mod.capability.Config().SetCommunicationPort {
		container.Ports = []corev1.ContainerPort{
			{
				Name:          consts.HttpsServicePortName,
				ContainerPort: consts.HttpsContainerPort,
			},
			{
				Name:          consts.HttpServicePortName,
				ContainerPort: consts.HttpContainerPort,
			},
		}
	}
	if mod.capability.Config().SetDnsEntryPoint {
		container.Env = append(container.Env,
			corev1.EnvVar{
				Name:  consts.EnvDtDnsEntryPoint,
				Value: mod.buildDNSEntryPoint(),
			})
	}

	return []corev1.Container{container}
}

func (mod BaseModifier) buildDNSEntryPoint() string {
	return fmt.Sprintf("https://%s/communication", buildServiceHostName(mod.dynakube.Name, mod.capability.ShortName()))
}

// BuildServiceHostName converts the name returned by BuildServiceName
// into the variable name which Kubernetes uses to reference the associated service.
// For more information see: https://kubernetes.io/docs/concepts/services-networking/service/
func buildServiceHostName(dynakubeName string, module string) string {
	serviceName :=
		strings.ReplaceAll(
			strings.ToUpper(
				capability.BuildServiceName(dynakubeName, module)),
			"-", "_")

	return fmt.Sprintf("$(%s_SERVICE_HOST):$(%s_SERVICE_PORT)", serviceName, serviceName)
}

func (mod BaseModifier) buildInitContainers() []corev1.Container {
	initContainers := mod.capability.InitContainersTemplates()

	for i := range initContainers {
		initContainers[i].Image = mod.dynakube.ActiveGateImage()
		initContainers[i].Resources = mod.capability.Properties().Resources
	}

	return initContainers
}

func (mod BaseModifier) buildCommonEnvs() []corev1.EnvVar {
	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(string(mod.kubeUID), consts.DeploymentTypeActiveGate)

	envs := []corev1.EnvVar{
		{Name: consts.EnvDtCapabilities, Value: mod.capability.ArgName()},
		{Name: consts.EnvDtIdSeedNamespace, Value: mod.dynakube.Namespace},
		{Name: consts.EnvDtIdSeedClusterId, Value: string(mod.kubeUID)},
		{Name: consts.EnvDtDeploymentMetadata, Value: deploymentMetadata.AsString()},
	}
	envs = append(envs, mod.capability.Properties().Env...)

	if mod.capability.Properties().Group != "" {
		envs = append(envs, corev1.EnvVar{Name: consts.EnvDtGroup, Value: mod.capability.Properties().Group})
	}
	if mod.dynakube.Spec.NetworkZone != "" {
		envs = append(envs, corev1.EnvVar{Name: consts.EnvDtNetworkZone, Value: mod.dynakube.Spec.NetworkZone})
	}
	return envs
}

func (mod BaseModifier) buildServiceAccountName() string {
	return consts.ServiceAccountPrefix + mod.capability.Config().ServiceAccountOwner
}

func nodeAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: affinityNodeSelectorTerms(),
			},
		},
	}
}

func affinityNodeSelectorTerms() []corev1.NodeSelectorTerm {
	nodeSelectorTerms := []corev1.NodeSelectorTerm{
		kubernetesArchOsSelectorTerm(),
	}
	return nodeSelectorTerms
}

func kubernetesArchOsSelectorTerm() corev1.NodeSelectorTerm {
	return corev1.NodeSelectorTerm{
		MatchExpressions: kubeobjects.AffinityNodeRequirement(),
	}
}
