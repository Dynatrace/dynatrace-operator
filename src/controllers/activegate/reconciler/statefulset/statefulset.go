package statefulset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/agproxysecret"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/customproperties"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/internal/events"
	"github.com/Dynatrace/dynatrace-operator/src/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	serviceAccountPrefix = "dynatrace-"

	AnnotationVersion         = dynatracev1beta1.InternalFlagPrefix + "version"
	AnnotationCustomPropsHash = dynatracev1beta1.InternalFlagPrefix + "custom-properties-hash"

	DTCapabilities       = "DT_CAPABILITIES"
	DTIdSeedNamespace    = "DT_ID_SEED_NAMESPACE"
	DTIdSeedClusterId    = "DT_ID_SEED_K8S_CLUSTER_ID"
	DTNetworkZone        = "DT_NETWORK_ZONE"
	DTGroup              = "DT_GROUP"
	DTDeploymentMetadata = "DT_DEPLOYMENT_METADATA"
)

type statefulSetProperties struct {
	*dynatracev1beta1.DynaKube
	*dynatracev1beta1.CapabilityProperties
	customPropertiesHash    string
	kubeSystemUID           types.UID
	feature                 string
	capabilityName          string
	serviceAccountOwner     string
	OnAfterCreateListener   []events.StatefulSetEvent
	initContainersTemplates []corev1.Container
	containerVolumeMounts   []corev1.VolumeMount
	volumes                 []corev1.Volume
}

func NewStatefulSetProperties(instance *dynatracev1beta1.DynaKube, capabilityProperties *dynatracev1beta1.CapabilityProperties,
	kubeSystemUID types.UID, customPropertiesHash string, feature string, capabilityName string, serviceAccountOwner string,
	initContainers []corev1.Container, containerVolumeMounts []corev1.VolumeMount, volumes []corev1.Volume) *statefulSetProperties {
	if serviceAccountOwner == "" {
		serviceAccountOwner = feature
	}

	return &statefulSetProperties{
		DynaKube:                instance,
		CapabilityProperties:    capabilityProperties,
		customPropertiesHash:    customPropertiesHash,
		kubeSystemUID:           kubeSystemUID,
		feature:                 feature,
		capabilityName:          capabilityName,
		serviceAccountOwner:     serviceAccountOwner,
		OnAfterCreateListener:   []events.StatefulSetEvent{},
		initContainersTemplates: initContainers,
		containerVolumeMounts:   containerVolumeMounts,
		volumes:                 volumes,
	}
}

func CreateStatefulSet(stsProperties *statefulSetProperties) (*appsv1.StatefulSet, error) {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        stsProperties.Name + "-" + stsProperties.feature,
			Namespace:   stsProperties.Namespace,
			Labels:      buildLabels(stsProperties.DynaKube, stsProperties.feature, stsProperties.CapabilityProperties),
			Annotations: map[string]string{},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            stsProperties.Replicas,
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Selector:            &metav1.LabelSelector{MatchLabels: BuildLabelsFromInstance(stsProperties.DynaKube, stsProperties.feature)},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: buildLabels(stsProperties.DynaKube, stsProperties.feature, stsProperties.CapabilityProperties),
					Annotations: map[string]string{
						AnnotationVersion:         stsProperties.Status.ActiveGate.Version,
						AnnotationCustomPropsHash: stsProperties.customPropertiesHash,
					},
				},
				Spec: buildTemplateSpec(stsProperties),
			},
		}}

	for _, onAfterCreateListener := range stsProperties.OnAfterCreateListener {
		onAfterCreateListener(sts)
	}

	hash, err := kubeobjects.GenerateHash(sts)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sts.ObjectMeta.Annotations[kubeobjects.AnnotationHash] = hash
	return sts, nil
}

func getContainerBuilders(stsProperties *statefulSetProperties) []kubeobjects.ContainerBuilder {
	if stsProperties.NeedsStatsD() {
		return []kubeobjects.ContainerBuilder{
			NewExtensionController(stsProperties),
			NewStatsD(stsProperties),
		}
	}
	return nil
}

func buildTemplateSpec(stsProperties *statefulSetProperties) corev1.PodSpec {
	extraContainerBuilders := getContainerBuilders(stsProperties)
	podSpec := corev1.PodSpec{
		Containers:         buildContainers(stsProperties, extraContainerBuilders),
		InitContainers:     buildInitContainers(stsProperties),
		NodeSelector:       stsProperties.CapabilityProperties.NodeSelector,
		ServiceAccountName: determineServiceAccountName(stsProperties),
		Affinity:           affinity(),
		Tolerations:        stsProperties.Tolerations,
		Volumes:            buildVolumes(stsProperties, extraContainerBuilders),
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: stsProperties.PullSecret()},
		},
	}
	if dnsPolicy := buildDNSPolicy(stsProperties); dnsPolicy != "" {
		podSpec.DNSPolicy = dnsPolicy
	}
	return podSpec
}

func buildDNSPolicy(stsProperties *statefulSetProperties) corev1.DNSPolicy {
	if stsProperties.ActiveGateMode() {
		return stsProperties.Spec.ActiveGate.DNSPolicy
	}
	return ""
}

func buildInitContainers(stsProperties *statefulSetProperties) []corev1.Container {
	ics := stsProperties.initContainersTemplates

	for idx := range ics {
		ics[idx].Image = stsProperties.DynaKube.ActiveGateImage()
		ics[idx].Resources = stsProperties.CapabilityProperties.Resources
	}

	return ics
}

func buildContainers(stsProperties *statefulSetProperties, extraContainerBuilders []kubeobjects.ContainerBuilder) []corev1.Container {
	containers := []corev1.Container{
		buildActiveGateContainer(stsProperties),
	}

	for _, containerBuilder := range extraContainerBuilders {
		containers = append(containers,
			containerBuilder.BuildContainer(),
		)
	}
	return containers
}

func buildActiveGateContainer(stsProperties *statefulSetProperties) corev1.Container {
	return corev1.Container{
		Name:            consts.ActiveGateContainerName,
		Image:           stsProperties.DynaKube.ActiveGateImage(),
		Resources:       stsProperties.CapabilityProperties.Resources,
		ImagePullPolicy: corev1.PullAlways,
		Env:             buildEnvs(stsProperties),
		VolumeMounts:    buildVolumeMounts(stsProperties),
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
	}
}

func buildVolumes(stsProperties *statefulSetProperties, extraContainerBuilders []kubeobjects.ContainerBuilder) []corev1.Volume {
	var volumes []corev1.Volume

	if !isCustomPropertiesNilOrEmpty(stsProperties.CustomProperties) {
		valueFrom := determineCustomPropertiesSource(stsProperties)
		volumes = append(volumes, corev1.Volume{
			Name: customproperties.VolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: valueFrom,
					Items: []corev1.KeyToPath{
						{Key: customproperties.DataKey, Path: customproperties.DataPath},
					}}}},
		)
	}

	for _, containerBuilder := range extraContainerBuilders {
		volumes = append(volumes,
			containerBuilder.BuildVolumes()...,
		)
	}

	volumes = append(volumes, stsProperties.volumes...)

	if stsProperties.HasProxy() {
		volumes = append(volumes, buildProxyVolumes(stsProperties)...)
	}

	return volumes
}

func buildProxyVolumes(stsProperties *statefulSetProperties) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: InternalProxySecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: agproxysecret.BuildProxySecretName(),
				},
			},
		},
	}
}

func determineCustomPropertiesSource(stsProperties *statefulSetProperties) string {
	if stsProperties.CustomProperties.ValueFrom == "" {
		return fmt.Sprintf("%s-%s-%s", stsProperties.Name, stsProperties.serviceAccountOwner, customproperties.Suffix)
	}
	return stsProperties.CustomProperties.ValueFrom
}

func buildVolumeMounts(stsProperties *statefulSetProperties) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount

	if !isCustomPropertiesNilOrEmpty(stsProperties.CustomProperties) {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      customproperties.VolumeName,
			MountPath: customproperties.MountPath,
			SubPath:   customproperties.DataPath,
		})
	}

	if stsProperties.NeedsStatsD() {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{Name: eecAuthToken, MountPath: "/var/lib/dynatrace/gateway/config"},
		)
	}

	volumeMounts = append(volumeMounts, stsProperties.containerVolumeMounts...)

	if stsProperties.HasProxy() {
		volumeMounts = append(volumeMounts, buildProxyMounts()...)
	}

	return volumeMounts
}

func buildProxyMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretHostMountPath,
			SubPath:   InternalProxySecretHost,
		},
		{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretPortMountPath,
			SubPath:   InternalProxySecretPort,
		},
		{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretUsernameMountPath,
			SubPath:   InternalProxySecretUsername,
		},
		{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretPasswordMountPath,
			SubPath:   InternalProxySecretPassword,
		},
	}
}

func buildEnvs(stsProperties *statefulSetProperties) []corev1.EnvVar {
	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(string(stsProperties.kubeSystemUID), deploymentmetadata.DeploymentTypeActiveGate)

	envs := []corev1.EnvVar{
		{Name: DTCapabilities, Value: stsProperties.capabilityName},
		{Name: DTIdSeedNamespace, Value: stsProperties.Namespace},
		{Name: DTIdSeedClusterId, Value: string(stsProperties.kubeSystemUID)},
		{Name: DTDeploymentMetadata, Value: deploymentMetadata.AsString()},
	}
	envs = append(envs, stsProperties.Env...)

	if stsProperties.Group != "" {
		envs = append(envs, corev1.EnvVar{Name: DTGroup, Value: stsProperties.Group})
	}
	if stsProperties.Spec.NetworkZone != "" {
		envs = append(envs, corev1.EnvVar{Name: DTNetworkZone, Value: stsProperties.Spec.NetworkZone})
	}

	return envs
}

func determineServiceAccountName(stsProperties *statefulSetProperties) string {
	return serviceAccountPrefix + stsProperties.serviceAccountOwner
}

func isCustomPropertiesNilOrEmpty(customProperties *dynatracev1beta1.DynaKubeValueSource) bool {
	return customProperties == nil ||
		(customProperties.Value == "" &&
			customProperties.ValueFrom == "")
}
