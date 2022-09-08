package statefulset

import (
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder"
	agmodifiers "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder/modifiers"
	"github.com/Dynatrace/dynatrace-operator/src/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type StatefulSetBuilder struct {
	kubeUID    types.UID
	configHash string
	dynakube   dynatracev1beta1.DynaKube
	capability capability.Capability
}

func NewStatefulSetBuilder(kubeUID types.UID, configHash string, dynakube dynatracev1beta1.DynaKube, capability capability.Capability) StatefulSetBuilder {
	return StatefulSetBuilder{
		kubeUID:    kubeUID,
		configHash: configHash,
		dynakube:   dynakube,
		capability: capability,
	}
}

func (builder StatefulSetBuilder) CreateStatefulSet(modifiers []agbuilder.Modifier) (*appsv1.StatefulSet, error) {
	activeGateBuilder := agbuilder.NewBuilder(builder.getBase())
	if len(modifiers) == 0 {
		modifiers = agmodifiers.GetAllModifiers(builder.dynakube, builder.capability)
	}
	sts := activeGateBuilder.AddModifier(modifiers...).Build()

	if err := setHash(&sts); err != nil {
		return nil, err
	}

	return &sts, nil
}

func (builder StatefulSetBuilder) getBase() appsv1.StatefulSet {
	var sts appsv1.StatefulSet
	sts.ObjectMeta = builder.getBaseObjectMeta()
	sts.Spec = builder.getBaseSpec()
	builder.addLabels(&sts)
	builder.addTemplateSpec(&sts)

	if builder.dynakube.FeatureActiveGateAppArmor() {
		sts.Spec.Template.ObjectMeta.Annotations[consts.AnnotationActiveGateContainerAppArmor] = "runtime/default"
	}
	return sts
}

func (builder StatefulSetBuilder) getBaseObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        builder.dynakube.Name + "-" + builder.capability.ShortName(),
		Namespace:   builder.dynakube.Namespace,
		Annotations: map[string]string{},
	}
}

func (builder StatefulSetBuilder) getBaseSpec() appsv1.StatefulSetSpec {
	return appsv1.StatefulSetSpec{
		Replicas:            builder.capability.Properties().Replicas,
		PodManagementPolicy: appsv1.ParallelPodManagement,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					consts.AnnotationActiveGateConfigurationHash: builder.configHash,
				},
			},
		},
	}
}

func (builder StatefulSetBuilder) addLabels(sts *appsv1.StatefulSet) {
	versionLabelValue := builder.dynakube.Status.ActiveGate.Version
	if builder.dynakube.CustomActiveGateImage() != "" {
		versionLabelValue = kubeobjects.CustomImageLabelValue
	}
	appLabels := kubeobjects.NewAppLabels(kubeobjects.ActiveGateComponentLabel, builder.dynakube.Name, builder.capability.ShortName(), versionLabelValue)

	sts.ObjectMeta.Labels = appLabels.BuildLabels()
	sts.Spec.Selector = &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}
	sts.Spec.Template.ObjectMeta.Labels = kubeobjects.MergeMap(builder.capability.Properties().Labels, appLabels.BuildLabels())
}

func (builder StatefulSetBuilder) addTemplateSpec(sts *appsv1.StatefulSet) {
	podSpec := corev1.PodSpec{
		Containers:         builder.buildBaseContainer(),
		InitContainers:     builder.buildInitContainers(),
		NodeSelector:       builder.capability.Properties().NodeSelector,
		ServiceAccountName: builder.buildServiceAccountName(),
		Affinity:           nodeAffinity(),
		Tolerations:        buildTolerations(builder.capability),
		Volumes:            builder.capability.Volumes(),
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: builder.dynakube.PullSecret()},
		},
		PriorityClassName:         builder.dynakube.Spec.ActiveGate.PriorityClassName,
		DNSPolicy:                 builder.dynakube.Spec.ActiveGate.DNSPolicy,
		TopologySpreadConstraints: builder.capability.Properties().TopologySpreadConstraints,
	}
	sts.Spec.Template.Spec = podSpec
}

func buildTolerations(capability capability.Capability) []corev1.Toleration {
	tolerations := append(capability.Properties().Tolerations, kubeobjects.TolerationForAmd()...)
	return tolerations
}

func (builder StatefulSetBuilder) buildBaseContainer() []corev1.Container {
	container := corev1.Container{
		Name:            consts.ActiveGateContainerName,
		Image:           builder.dynakube.ActiveGateImage(),
		Resources:       builder.capability.Properties().Resources,
		Env:             builder.buildCommonEnvs(),
		VolumeMounts:    builder.capability.ContainerVolumeMounts(),
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
	if builder.capability.Config().SetReadinessPort {
		container.ReadinessProbe.HTTPGet.Port = intstr.FromString(consts.HttpsServicePortName)
	}
	if builder.capability.Config().SetCommunicationPort {
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
	if builder.capability.Config().SetDnsEntryPoint {
		container.Env = append(container.Env,
			corev1.EnvVar{
				Name:  consts.EnvDtDnsEntryPoint,
				Value: builder.buildDNSEntryPoint(),
			})
	}

	return []corev1.Container{container}
}

func (builder StatefulSetBuilder) buildDNSEntryPoint() string {
	return fmt.Sprintf("https://%s/communication", buildServiceHostName(builder.dynakube.Name, builder.capability.ShortName()))
}

func (builder StatefulSetBuilder) buildInitContainers() []corev1.Container {
	initContainers := builder.capability.InitContainersTemplates()

	for i := range initContainers {
		initContainers[i].Image = builder.dynakube.ActiveGateImage()
		initContainers[i].Resources = builder.capability.Properties().Resources
	}

	return initContainers
}

func (builder StatefulSetBuilder) buildCommonEnvs() []corev1.EnvVar {
	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(string(builder.kubeUID), consts.DeploymentTypeActiveGate)

	envs := []corev1.EnvVar{
		{Name: consts.EnvDtCapabilities, Value: builder.capability.ArgName()},
		{Name: consts.EnvDtIdSeedNamespace, Value: builder.dynakube.Namespace},
		{Name: consts.EnvDtIdSeedClusterId, Value: string(builder.kubeUID)},
		{Name: consts.EnvDtDeploymentMetadata, Value: deploymentMetadata.AsString()},
	}
	envs = append(envs, builder.capability.Properties().Env...)

	if builder.capability.Properties().Group != "" {
		envs = append(envs, corev1.EnvVar{Name: consts.EnvDtGroup, Value: builder.capability.Properties().Group})
	}
	if builder.dynakube.Spec.NetworkZone != "" {
		envs = append(envs, corev1.EnvVar{Name: consts.EnvDtNetworkZone, Value: builder.dynakube.Spec.NetworkZone})
	}
	return envs
}

func (builder StatefulSetBuilder) buildServiceAccountName() string {
	return consts.ServiceAccountPrefix + builder.capability.Config().ServiceAccountOwner
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
