package statefulset

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/internal/events"
	"github.com/Dynatrace/dynatrace-operator/controllers/customproperties"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	serviceAccountPrefix = "dynatrace-"

	AnnotationVersion         = "internal.operator.dynatrace.com/version"
	AnnotationCustomPropsHash = "internal.operator.dynatrace.com/custom-properties-hash"

	DTCapabilities       = "DT_CAPABILITIES"
	DTIdSeedNamespace    = "DT_ID_SEED_NAMESPACE"
	DTIdSeedClusterId    = "DT_ID_SEED_K8S_CLUSTER_ID"
	DTNetworkZone        = "DT_NETWORK_ZONE"
	DTGroup              = "DT_GROUP"
	DTInternalProxy      = "DT_INTERNAL_PROXY"
	DTDeploymentMetadata = "DT_DEPLOYMENT_METADATA"

	ProxySecretKey = "proxy"
)

type statefulSetProperties struct {
	*dynatracev1beta1.DynaKube
	*dynatracev1beta1.CapabilityProperties
	customPropertiesHash    string
	kubeSystemUID           types.UID
	feature                 string
	capabilityName          string
	serviceAccountOwner     string
	majorKubernetesVersion  string
	minorKubernetesVersion  string
	OnAfterCreateListener   []events.StatefulSetEvent
	initContainersTemplates []corev1.Container
	containerVolumeMounts   []corev1.VolumeMount
	volumes                 []corev1.Volume
}

func NewStatefulSetProperties(instance *dynatracev1beta1.DynaKube, capabilityProperties *dynatracev1beta1.CapabilityProperties,
	kubeSystemUID types.UID, customPropertiesHash string, feature string, capabilityName string, serviceAccountOwner string,
	majorKubernetesVersion string, minorKubernetesVersion string, initContainers []corev1.Container,
	containerVolumeMounts []corev1.VolumeMount, volumes []corev1.Volume) *statefulSetProperties {
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
		majorKubernetesVersion:  majorKubernetesVersion,
		minorKubernetesVersion:  minorKubernetesVersion,
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

func buildTemplateSpec(stsProperties *statefulSetProperties) corev1.PodSpec {
	return corev1.PodSpec{
		Containers:         []corev1.Container{buildContainer(stsProperties)},
		InitContainers:     buildInitContainers(stsProperties),
		NodeSelector:       stsProperties.CapabilityProperties.NodeSelector,
		ServiceAccountName: determineServiceAccountName(stsProperties),
		Affinity:           affinity(stsProperties),
		Tolerations:        stsProperties.Tolerations,
		Volumes:            buildVolumes(stsProperties),
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: stsProperties.PullSecret()},
		},
	}
}

func buildInitContainers(stsProperties *statefulSetProperties) []corev1.Container {
	ics := stsProperties.initContainersTemplates

	for idx := range ics {
		ics[idx].Image = stsProperties.DynaKube.ActiveGateImage()
		ics[idx].Resources = stsProperties.CapabilityProperties.Resources
	}

	return ics
}

func buildContainer(stsProperties *statefulSetProperties) corev1.Container {
	return corev1.Container{
		Name:            dynatracev1beta1.OperatorName,
		Image:           stsProperties.DynaKube.ActiveGateImage(),
		Resources:       stsProperties.CapabilityProperties.Resources,
		ImagePullPolicy: corev1.PullAlways,
		Env:             buildEnvs(stsProperties),
		VolumeMounts:    buildVolumeMounts(stsProperties),
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
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

func buildVolumes(stsProperties *statefulSetProperties) []corev1.Volume {
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

	volumes = append(volumes, stsProperties.volumes...)

	return volumes
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

	volumeMounts = append(volumeMounts, stsProperties.containerVolumeMounts...)

	return volumeMounts
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

	if !isProxyNilOrEmpty(stsProperties.Spec.Proxy) {
		envs = append(envs, buildProxyEnv(stsProperties.Spec.Proxy))
	}
	if stsProperties.Group != "" {
		envs = append(envs, corev1.EnvVar{Name: DTGroup, Value: stsProperties.Group})
	}
	if stsProperties.Spec.NetworkZone != "" {
		envs = append(envs, corev1.EnvVar{Name: DTNetworkZone, Value: stsProperties.Spec.NetworkZone})
	}

	return envs
}

func buildProxyEnv(proxy *dynatracev1beta1.DynaKubeProxy) corev1.EnvVar {
	if proxy.ValueFrom != "" {
		return corev1.EnvVar{
			Name: DTInternalProxy,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: proxy.ValueFrom},
					Key:                  ProxySecretKey,
				},
			},
		}
	} else {
		return corev1.EnvVar{
			Name:  DTInternalProxy,
			Value: proxy.Value,
		}
	}
}

func determineServiceAccountName(stsProperties *statefulSetProperties) string {
	return serviceAccountPrefix + stsProperties.serviceAccountOwner
}

func isCustomPropertiesNilOrEmpty(customProperties *dynatracev1beta1.DynaKubeValueSource) bool {
	return customProperties == nil ||
		(customProperties.Value == "" &&
			customProperties.ValueFrom == "")
}

func isProxyNilOrEmpty(proxy *dynatracev1beta1.DynaKubeProxy) bool {
	return proxy == nil || (proxy.Value == "" && proxy.ValueFrom == "")
}
