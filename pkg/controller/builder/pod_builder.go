package builder

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func BuildActiveGatePodSpecs(instance *v1alpha1.DynaKube, kubeSystemUID types.UID) (corev1.PodSpec, error) {
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume
	sa := MonitoringServiceAccount
	activeGateSpec := &instance.Spec.KubernetesMonitoringSpec
	additionalArgs := activeGateSpec.Args
	image := activeGateSpec.Image

	if activeGateSpec.ServiceAccountName != "" {
		sa = activeGateSpec.ServiceAccountName
	}
	if activeGateSpec.Image != "" {
		image = activeGateSpec.Image
	}

	envVars := buildEnvVars(instance, kubeSystemUID)

	checkMinimumResources(activeGateSpec)

	if instance.Spec.NetworkZone != "" {
		additionalArgs = append(additionalArgs,
			fmt.Sprintf(`--networkzone="%s"`, instance.Spec.NetworkZone))
	}

	additionalArgs, envVars = appendProxySettings(additionalArgs, envVars, instance.Spec.Proxy)
	additionalArgs = appendActivationGroup(additionalArgs, activeGateSpec.Group)
	volumeMounts, volumes = prepareCertificateVolumes(volumeMounts, volumes, instance.Spec.TrustedCAs)
	volumeMounts, volumes = prepareCustomPropertiesVolumes(volumeMounts, volumes, activeGateSpec.CustomProperties, instance.Name)

	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:            ActivegateName,
			Image:           image,
			Resources:       activeGateSpec.Resources,
			ImagePullPolicy: corev1.PullAlways,
			Env:             envVars,
			Args:            buildArgs(additionalArgs),
			VolumeMounts:    volumeMounts,
			ReadinessProbe:  buildReadinessProbe(),
			LivenessProbe:   buildLivenessProbe(),
		}},
		DNSPolicy:          activeGateSpec.DNSPolicy,
		NodeSelector:       activeGateSpec.NodeSelector,
		ServiceAccountName: sa,
		Affinity:           buildAffinity(),
		Tolerations:        activeGateSpec.Tolerations,
		PriorityClassName:  activeGateSpec.PriorityClassName,
		Volumes:            volumes,
	}

	err := preparePodSpecImmutableImage(&podSpec, instance)
	return podSpec, err
}

func prepareCustomPropertiesVolumes(volumeMounts []corev1.VolumeMount, volumes []corev1.Volume, customProperties *v1alpha1.DynaKubeValueSource, instanceName string) ([]corev1.VolumeMount, []corev1.Volume) {
	if customProperties == nil ||
		(customProperties.Value == "" && customProperties.ValueFrom == "") {
		return volumeMounts, volumes
	}

	valueFrom := customProperties.ValueFrom
	if valueFrom == "" {
		valueFrom = fmt.Sprintf("%s-%s", instanceName, _const.KubernetesMonitoringCustomPropertiesConfigMapNameSuffix)
	}

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		ReadOnly:  true,
		Name:      "custom-properties",
		MountPath: "/mnt/dynatrace/gateway/config"})

	volumes = append(volumes, corev1.Volume{
		Name: "custom-properties",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: valueFrom,
				Items: []corev1.KeyToPath{
					{
						Key:  _const.CustomPropertiesKey,
						Path: "custom.properties",
					},
				},
			},
		},
	})

	return volumeMounts, volumes
}

func appendActivationGroup(args []string, group string) []string {
	if group == "" {
		return args
	}
	return append(args,
		fmt.Sprintf(`--group="%s"`, group))
}

// prepareCertificateVolumes is currently a no-op, since the feature is not yet ready to be implemented
// Reevaluate this state by 2021-01-25 at the latest
// Until reevaluation, this function is being kept for reference
func prepareCertificateVolumes(volumeMounts []corev1.VolumeMount, volumes []corev1.Volume, _ string) ([]corev1.VolumeMount, []corev1.Volume) {
	var tmpVolumeMounts []corev1.VolumeMount
	var tmpVolumes []corev1.Volume

	//if trustedCAsConfig != "" {
	//	tmpVolumeMounts = append(tmpVolumeMounts, corev1.VolumeMount{
	//		Name:      "certs",
	//		MountPath: "/mnt/dynatrace/certs"})
	//
	//	tmpVolumes = append(tmpVolumes, corev1.Volume{
	//		Name: "certs",
	//		VolumeSource: corev1.VolumeSource{
	//			ConfigMap: &corev1.ConfigMapVolumeSource{
	//				LocalObjectReference: corev1.LocalObjectReference{
	//					Name: trustedCAsConfig,
	//				},
	//				Items: []corev1.KeyToPath{
	//					{
	//						Key:  "certs",
	//						Path: "certs.pem",
	//					},
	//				},
	//			},
	//		},
	//	})
	//}

	return append(tmpVolumeMounts, volumeMounts...), append(tmpVolumes, volumes...)
}

func appendProxySettings(args []string, envVars []corev1.EnvVar, proxy *v1alpha1.DynaKubeProxy) ([]string, []corev1.EnvVar) {
	if proxy == nil || (proxy.Value == "" && proxy.ValueFrom == "") {
		return args, envVars
	}

	proxyEnvVar := corev1.EnvVar{
		Name: "ACTIVE_GATE_PROXY",
	}

	if proxy.ValueFrom != "" {
		proxyEnvVar.ValueFrom = &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: proxy.ValueFrom},
				Key:                  "proxy",
			},
		}
	} else {
		proxyEnvVar.Value = proxy.Value
	}

	args = append(args, `PROXY="${ACTIVE_GATE_PROXY}"`)
	envVars = append(envVars, proxyEnvVar)
	return args, envVars
}

func checkMinimumResources(activeGateSpec *v1alpha1.KubernetesMonitoringSpec) {
	cpuMin := resource.MustParse(ResourceCPUMinimum)
	cpuMax := resource.MustParse(ResourceCPUMaximum)
	memoryMin := resource.MustParse(ResourceMemoryMinimum)
	memoryMax := resource.MustParse(ResourceMemoryMaximum)

	if activeGateSpec.Resources.Requests == nil {
		activeGateSpec.Resources.Requests = corev1.ResourceList{}
	}

	if activeGateSpec.Resources.Limits == nil {
		activeGateSpec.Resources.Limits = corev1.ResourceList{}
	}

	cpuRequest, hasCPURequest := activeGateSpec.Resources.Requests[corev1.ResourceCPU]
	memoryRequest, hasMemoryRequest := activeGateSpec.Resources.Requests[corev1.ResourceCPU]
	cpuLimit, hasCPULimit := activeGateSpec.Resources.Limits[corev1.ResourceCPU]
	memoryLimit, hasMemoryLimit := activeGateSpec.Resources.Limits[corev1.ResourceCPU]

	// Memory limit does not exist or is higher than necessary => set to maximum
	if !hasMemoryLimit || memoryLimit.Cmp(memoryMax) > 0 {
		memoryLimit = memoryMax
	}

	// CPU limit does not exist or is higher than necessary => set to maximum
	if !hasCPULimit || cpuLimit.Cmp(cpuMax) > 0 {
		cpuLimit = cpuMax
	}

	// Memory request does not exist or is lower than required => set to minimum
	if !hasMemoryRequest || memoryRequest.Cmp(memoryMin) < 0 {
		memoryRequest = memoryMin
	}

	// CPU request does not exist or is lower than required => set to minimum
	if !hasCPURequest || cpuRequest.Cmp(cpuMin) < 0 {
		cpuRequest = cpuMin
	}

	// CPU request is higher than limit => set to limit
	if cpuRequest.Cmp(cpuLimit) > 0 {
		cpuRequest = cpuLimit
	}

	// Memory request is higher than limit => set to limit
	if memoryRequest.Cmp(memoryLimit) > 0 {
		memoryRequest = memoryLimit
	}

	activeGateSpec.Resources.Requests[corev1.ResourceCPU] = cpuRequest
	activeGateSpec.Resources.Requests[corev1.ResourceMemory] = memoryRequest
	activeGateSpec.Resources.Limits[corev1.ResourceCPU] = cpuLimit
	activeGateSpec.Resources.Limits[corev1.ResourceMemory] = memoryLimit
}

func buildLivenessProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/rest/state",
				Port:   intstr.IntOrString{IntVal: 9999},
				Scheme: "HTTPS",
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       30,
		FailureThreshold:    2,
	}
}

func buildReadinessProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/rest/health",
				Port:   intstr.IntOrString{IntVal: 9999},
				Scheme: "HTTPS",
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       15,
		FailureThreshold:    3,
	}
}

func buildArgs(additionalArgs []string) []string {
	return append([]string{
		DtCapabilitiesArg,
	}, additionalArgs...)
}

func buildEnvVars(instance *v1alpha1.DynaKube, kubeSystemUID types.UID) []corev1.EnvVar {
	var capabilities []string

	if instance.Spec.KubernetesMonitoringSpec.Enabled {
		capabilities = append(capabilities, "kubernetes_monitoring")
	}

	return append([]corev1.EnvVar{
		{
			Name:  DtCapabilities,
			Value: strings.Join(capabilities, Comma),
		},
		{
			Name:  DtIdSeedNamespace,
			Value: instance.Namespace,
		},
		{
			Name:  DtIdSeedClusterId,
			Value: string(kubeSystemUID),
		},
	}, instance.Spec.KubernetesMonitoringSpec.Env...)
}

func buildAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      KubernetesBetaArch,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{AMD64, ARM64},
							},
							{
								Key:      KubernetesBetaOs,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{LINUX},
							},
						},
					},
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      KubernetesArch,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{AMD64, ARM64},
							},
							{
								Key:      KubernetesOs,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{LINUX},
							},
						},
					},
				},
			},
		},
	}
}

func preparePodSpecImmutableImage(podSpec *corev1.PodSpec, instance *v1alpha1.DynaKube) error {
	pullSecretName := instance.GetName() + "-pull-secret"
	if instance.Spec.CustomPullSecret != "" {
		pullSecretName = instance.Spec.CustomPullSecret
	}

	podSpec.ImagePullSecrets = append(podSpec.ImagePullSecrets, corev1.LocalObjectReference{
		Name: pullSecretName,
	})

	if instance.Spec.KubernetesMonitoringSpec.Image == "" {
		i, err := BuildActiveGateImage(instance.Spec.APIURL, instance.Spec.KubernetesMonitoringSpec.ActiveGateVersion)
		if err != nil {
			return err
		}
		podSpec.Containers[0].Image = i
	}

	return nil
}

func BuildLabels(name string, labels map[string]string) map[string]string {
	result := BuildLabelsForQuery(name)
	for key, value := range labels {
		result[key] = value
	}
	return result
}

func MergeLabels(labels ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range labels {
		for k, v := range m {
			res[k] = v
		}
	}

	return res
}

// buildLabels returns generic labels based on the name given for a Dynatrace OneAgent
func BuildLabelsForQuery(name string) map[string]string {
	return map[string]string{
		"dynatrace":  "activegate",
		"activegate": name,
	}
}

const (
	ActivegateName = "dynatrace-operator"

	MonitoringServiceAccount = "dynatrace-kubernetes-monitoring"

	KubernetesArch     = "kubernetes.io/arch"
	KubernetesOs       = "kubernetes.io/os"
	KubernetesBetaArch = "beta.kubernetes.io/arch"
	KubernetesBetaOs   = "beta.kubernetes.io/os"

	AMD64 = "amd64"
	ARM64 = "arm64"
	LINUX = "linux"

	DtCapabilities    = "DT_CAPABILITIES"
	DtIdSeedNamespace = "DT_ID_SEED_NAMESPACE"
	DtIdSeedClusterId = "DT_ID_SEED_K8S_CLUSTER_ID"

	DtCapabilitiesArg = "--enable=kubernetes_monitoring"

	Comma = ","

	// Usage of SI-Prefix Mega instead of IEC-Prefix Mebi to make use of
	// scaling provided by resource.*. E.g., resource.Milli
	ResourceMemoryMinimum = "250M"
	ResourceCPUMinimum    = "150m"
	ResourceMemoryMaximum = "1G"
	ResourceCPUMaximum    = "300m"
)
