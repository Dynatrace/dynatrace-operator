package statefulset

import (
	"encoding/json"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/controllers/tokens"
	"hash/fnv"
	"strconv"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/internal/events"
	"github.com/Dynatrace/dynatrace-operator/controllers/customproperties"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
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

	kubernetesArch     = "kubernetes.io/arch"
	kubernetesOS       = "kubernetes.io/os"
	kubernetesBetaArch = "beta.kubernetes.io/arch"
	kubernetesBetaOS   = "beta.kubernetes.io/os"

	amd64 = "amd64"
	linux = "linux"

	AnnotationTemplateHash    = "internal.operator.dynatrace.com/template-hash"
	AnnotationVersion         = "internal.operator.dynatrace.com/version"
	AnnotationCustomPropsHash = "internal.operator.dynatrace.com/custom-properties-hash"

	DTCapabilities       = "DT_CAPABILITIES"
	DTIdSeedNamespace    = "DT_ID_SEED_NAMESPACE"
	DTIdSeedClusterId    = "DT_ID_SEED_K8S_CLUSTER_ID"
	DTNetworkZone        = "DT_NETWORK_ZONE"
	DTGroup              = "DT_GROUP"
	DTInternalProxy      = "DT_INTERNAL_PROXY"
	DTDeploymentMetadata = "DT_DEPLOYMENT_METADATA"

	ProxyKey = "ProxyKey"

	TokensSecretVolumeName = "dynatrace-tokens-volume"
)

type statefulSetProperties struct {
	*dynatracev1alpha1.DynaKube
	*dynatracev1alpha1.CapabilityProperties
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

func NewStatefulSetProperties(instance *dynatracev1alpha1.DynaKube, capabilityProperties *dynatracev1alpha1.CapabilityProperties,
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

	hash, err := generateStatefulSetHash(sts)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sts.ObjectMeta.Annotations[AnnotationTemplateHash] = hash
	return sts, nil
}

func buildTemplateSpec(stsProperties *statefulSetProperties) corev1.PodSpec {
	return corev1.PodSpec{
		Containers:         []corev1.Container{buildContainer(stsProperties)},
		InitContainers:     buildInitContainers(stsProperties),
		NodeSelector:       stsProperties.NodeSelector,
		ServiceAccountName: determineServiceAccountName(stsProperties),
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{MatchExpressions: buildKubernetesExpression(kubernetesBetaArch, kubernetesBetaOS)},
						{MatchExpressions: buildKubernetesExpression(kubernetesArch, kubernetesOS)},
					}}}},
		Tolerations: stsProperties.Tolerations,
		Volumes:     buildVolumes(stsProperties),
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: stsProperties.Name + dtpullsecret.PullSecretSuffix},
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
		Name:            dynatracev1alpha1.OperatorName,
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

	volumes = append(volumes, corev1.Volume{
		Name: TokensSecretVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: tokens.TokensSecretsName,
			},
		},
	})

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

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      TokensSecretVolumeName,
		ReadOnly:  true,
		MountPath: "/var/lib/dynatrace/secrets",
	})

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
	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(string(stsProperties.kubeSystemUID))

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

func buildProxyEnv(proxy *dynatracev1alpha1.DynaKubeProxy) corev1.EnvVar {
	if proxy.ValueFrom != "" {
		return corev1.EnvVar{
			Name: DTInternalProxy,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: proxy.ValueFrom},
					Key:                  ProxyKey,
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
	if stsProperties.ServiceAccountName == "" {
		return serviceAccountPrefix + stsProperties.serviceAccountOwner
	}
	return stsProperties.ServiceAccountName
}

func buildKubernetesExpression(archKey string, osKey string) []corev1.NodeSelectorRequirement {
	return []corev1.NodeSelectorRequirement{
		{
			Key:      archKey,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{amd64},
		},
		{
			Key:      osKey,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{linux},
		},
	}
}

func isCustomPropertiesNilOrEmpty(customProperties *dynatracev1alpha1.DynaKubeValueSource) bool {
	return customProperties == nil ||
		(customProperties.Value == "" &&
			customProperties.ValueFrom == "")
}

func isProxyNilOrEmpty(proxy *dynatracev1alpha1.DynaKubeProxy) bool {
	return proxy == nil || (proxy.Value == "" && proxy.ValueFrom == "")
}

func generateStatefulSetHash(sts *appsv1.StatefulSet) (string, error) {
	data, err := json.Marshal(sts)
	if err != nil {
		return "", errors.WithStack(err)
	}

	hasher := fnv.New32()
	_, err = hasher.Write(data)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return strconv.FormatUint(uint64(hasher.Sum32()), 10), nil
}

func HasStatefulSetChanged(a *appsv1.StatefulSet, b *appsv1.StatefulSet) bool {
	return GetTemplateHash(a) != GetTemplateHash(b)
}

func GetTemplateHash(a metav1.Object) string {
	if annotations := a.GetAnnotations(); annotations != nil {
		return annotations[AnnotationTemplateHash]
	}
	return ""
}
