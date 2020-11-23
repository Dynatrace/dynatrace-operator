package kubemon

import (
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	MonitoringServiceAccount = "dynatrace-kubernetes-monitoring"
	KubernetesArch           = "kubernetes.io/arch"
	KubernetesOS             = "kubernetes.io/os"
	KubernetesBetaArch       = "beta.kubernetes.io/arch"
	KubernetesBetaOS         = "beta.kubernetes.io/os"

	AMD64 = "amd64"
	ARM64 = "arm64"
	LINUX = "linux"

	DTTenant          = "DT_TENANT"
	DTServer          = "DT_SERVER"
	DTToken           = "DT_TOKEN"
	DTCapabilities    = "DT_CAPABILITIES"
	DTIdSeedNamespace = "DT_ID_SEED_NAMESPACE"
	DTIdSeedClusterId = "DT_ID_SEED_K8S_CLUSTER_ID"

	DTTenantArg       = "--tenant=$(DT_TENANT)"
	DTTokenArg        = "--token=$(DT_TOKEN)"
	DTServerArg       = "--server=$(DT_SERVER)"
	DTCapabilitiesArg = "--enable=$(DT_CAPABILITIES)"

	ProxyArg = `PROXY="${ACTIVE_GATE_PROXY}"`
	ProxyEnv = "ACTIVE_GATE_PROXY"
	ProxyKey = "ProxyKey"

	CapabilityEnv = "kubernetes_monitoring"
)

func newStatefulSet(instance dynatracev1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) *v1.StatefulSet {
	return &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        dynatracev1alpha1.Name,
			Namespace:   instance.Namespace,
			Labels:      buildLabels(&instance),
			Annotations: map[string]string{},
		},
		Spec: v1.StatefulSetSpec{
			Replicas: instance.Spec.KubernetesMonitoringSpec.Replicas,
			Selector: buildLabelSelector(&instance),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: buildLabels(&instance)},
				Spec:       buildTemplateSpec(&instance, tenantInfo, kubeSystemUID),
			},
		},
	}
}

func buildTemplateSpec(instance *dynatracev1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) corev1.PodSpec {
	serviceAccountName := instance.Spec.KubernetesMonitoringSpec.ServiceAccountName
	if serviceAccountName == "" {
		serviceAccountName = MonitoringServiceAccount
	}

	return corev1.PodSpec{
		Containers:         []corev1.Container{buildContainer(instance, tenantInfo, kubeSystemUID)},
		DNSPolicy:          instance.Spec.KubernetesMonitoringSpec.DNSPolicy,
		NodeSelector:       instance.Spec.KubernetesMonitoringSpec.NodeSelector,
		ServiceAccountName: serviceAccountName,
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{MatchExpressions: buildKubernetesExpression(KubernetesBetaArch, KubernetesBetaOS)},
						{MatchExpressions: buildKubernetesExpression(KubernetesArch, KubernetesOS)},
					},
				},
			},
		},
		Tolerations:       instance.Spec.KubernetesMonitoringSpec.Tolerations,
		PriorityClassName: instance.Spec.KubernetesMonitoringSpec.PriorityClassName,
		Volumes:           buildVolumes(instance),
		ImagePullSecrets: []corev1.LocalObjectReference{
			buildPullSecret(instance),
		},
	}
}

func buildContainer(instance *dynatracev1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) corev1.Container {
	var volumeMounts []corev1.VolumeMount
	customProperties := instance.Spec.KubernetesMonitoringSpec.CustomProperties
	if !isCustomPropertiesNilOrEmpty(customProperties) {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      customproperties.VolumeName,
			MountPath: customproperties.MountPath,
		})
	}

	return corev1.Container{
		Name:            dynatracev1alpha1.OperatorName,
		Image:           buildImage(instance),
		Resources:       buildResources(instance),
		ImagePullPolicy: corev1.PullAlways,
		Env:             buildEnvs(instance, tenantInfo, kubeSystemUID),
		Args:            buildArgs(instance),
		VolumeMounts:    volumeMounts,
		ReadinessProbe: &corev1.Probe{
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
		},
		LivenessProbe: &corev1.Probe{
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
		},
	}
}
func buildLabels(instance *dynatracev1alpha1.DynaKube) map[string]string {
	return MergeLabels(instance.Labels,
		BuildLabelsFromInstance(instance),
		instance.Spec.KubernetesMonitoringSpec.Labels)
}

func buildLabelSelector(instance *dynatracev1alpha1.DynaKube) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: MergeLabels(
			BuildLabelsFromInstance(instance),
			instance.Spec.KubernetesMonitoringSpec.Labels),
	}
}

func buildKubernetesExpression(archKey string, osKey string) []corev1.NodeSelectorRequirement {
	return []corev1.NodeSelectorRequirement{
		{
			Key:      archKey,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{AMD64, ARM64},
		},
		{
			Key:      osKey,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{LINUX},
		},
	}
}

func buildVolumes(instance *dynatracev1alpha1.DynaKube) []corev1.Volume {
	var volumes []corev1.Volume

	customProperties := instance.Spec.KubernetesMonitoringSpec.CustomProperties
	if !isCustomPropertiesNilOrEmpty(customProperties) {
		valueFrom := instance.Spec.KubernetesMonitoringSpec.CustomProperties.ValueFrom
		if valueFrom == "" {
			valueFrom = fmt.Sprintf("%s-kubernetes-monitoring-%s", instance.Name, customproperties.Suffix)
		}

		volumes = append(volumes, corev1.Volume{
			Name: customproperties.VolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: valueFrom,
					Items: []corev1.KeyToPath{
						{Key: customproperties.DataKey, Path: customproperties.DataPath},
					},
				},
			},
		})
	}

	return volumes
}

func isCustomPropertiesNilOrEmpty(customProperties *dynatracev1alpha1.DynaKubeValueSource) bool {
	return customProperties == nil ||
		(customProperties.Value == "" &&
			customProperties.ValueFrom == "")
}

func buildEnvs(instance *dynatracev1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{Name: DTTenant, Value: tenantInfo.ID},
		{Name: DTToken, Value: tenantInfo.Token},
		{Name: DTServer, Value: tenantInfo.CommunicationEndpoint},
		{Name: DTCapabilities, Value: CapabilityEnv},
		{Name: DTIdSeedNamespace, Value: instance.Namespace},
		{Name: DTIdSeedClusterId, Value: string(kubeSystemUID)},
	}

	envVars = append(envVars, instance.Spec.KubernetesMonitoringSpec.Env...)

	proxy := instance.Spec.Proxy
	if !isProxyNilOrEmpty(proxy) {
		var proxyEnvVar corev1.EnvVar

		if proxy.ValueFrom != "" {
			proxyEnvVar = corev1.EnvVar{
				Name: ProxyEnv,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: proxy.ValueFrom},
						Key:                  ProxyKey,
					},
				},
			}
		} else {
			proxyEnvVar = corev1.EnvVar{
				Name:  ProxyEnv,
				Value: proxy.Value,
			}
		}

		envVars = append(envVars, proxyEnvVar)
	}

	return envVars
}

func buildArgs(instance *dynatracev1alpha1.DynaKube) []string {
	args := []string{
		DTTenantArg,
		DTTokenArg,
		DTServerArg,
		DTCapabilitiesArg,
	}

	args = append(args, instance.Spec.KubernetesMonitoringSpec.Args...)

	if instance.Spec.NetworkZone != "" {
		args = append(args, fmt.Sprintf(`--networkzone="%s"`, instance.Spec.NetworkZone))
	}

	proxy := instance.Spec.Proxy
	if !isProxyNilOrEmpty(proxy) {
		args = append(args, ProxyArg)
	}

	group := instance.Spec.KubernetesMonitoringSpec.Group
	if group != "" {
		args = append(args, fmt.Sprintf(`--group="%s"`, group))
	}

	return args

}

func isProxyNilOrEmpty(proxy *dynatracev1alpha1.DynaKubeProxy) bool {
	return proxy == nil || (proxy.Value == "" && proxy.ValueFrom == "")
}

func BuildLabelsFromInstance(instance *dynatracev1alpha1.DynaKube) map[string]string {
	return map[string]string{
		"dynatrace":  "activegate",
		"activegate": instance.Name,
	}
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
