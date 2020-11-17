package kubemon

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
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
	KubernetesOs             = "kubernetes.io/os"
	KubernetesBetaArch       = "beta.kubernetes.io/arch"
	KubernetesBetaOs         = "beta.kubernetes.io/os"

	AMD64 = "amd64"
	ARM64 = "arm64"
	LINUX = "linux"

	DtTenant          = "DT_TENANT"
	DtServer          = "DT_SERVER"
	DtToken           = "DT_TOKEN"
	DtCapabilities    = "DT_CAPABILITIES"
	DtIdSeedNamespace = "DT_ID_SEED_NAMESPACE"
	DtIdSeedClusterId = "DT_ID_SEED_K8S_CLUSTER_ID"

	DtTenantArg       = "--tenant=$(DT_TENANT)"
	DtTokenArg        = "--token=$(DT_TOKEN)"
	DtServerArg       = "--server=$(DT_SERVER)"
	DtCapabilitiesArg = "--enable=$(DT_CAPABILITIES)"

	ProxyArg = `PROXY="${ACTIVE_GATE_PROXY}"`
	ProxyEnv = "ACTIVE_GATE_PROXY"
	ProxyKey = "ProxyKey"

	CapabilityEnv = "kubernetes_monitoring"
)

func newStatefulSet(instance v1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) *v1.StatefulSet {
	return &v1.StatefulSet{
		ObjectMeta: buildObjectMeta(&instance),
		Spec:       buildSpec(&instance, tenantInfo, kubeSystemUID),
	}
}

func buildObjectMeta(instance *v1alpha1.DynaKube) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        v1alpha1.Name,
		Namespace:   instance.Namespace,
		Labels:      buildLabels(instance),
		Annotations: map[string]string{},
	}
}

func buildSpec(instance *v1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) v1.StatefulSetSpec {
	return v1.StatefulSetSpec{
		Replicas: instance.Spec.KubernetesMonitoringSpec.Replicas,
		Selector: buildLabelSelector(instance),
		Template: buildTemplate(instance, tenantInfo, kubeSystemUID),
	}
}

func buildLabels(instance *v1alpha1.DynaKube) map[string]string {
	return MergeLabels(instance.Labels,
		BuildLabelsFromInstance(instance),
		instance.Spec.KubernetesMonitoringSpec.Labels)
}

func buildLabelSelector(instance *v1alpha1.DynaKube) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: MergeLabels(
			BuildLabelsFromInstance(instance),
			instance.Spec.KubernetesMonitoringSpec.Labels),
	}
}

func buildTemplate(instance *v1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: buildLabels(instance)},
		Spec:       buildTemplateSpec(instance, tenantInfo, kubeSystemUID),
	}
}

func buildTemplateSpec(instance *v1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) corev1.PodSpec {
	return corev1.PodSpec{
		Containers:         []corev1.Container{buildContainer(instance, tenantInfo, kubeSystemUID)},
		DNSPolicy:          instance.Spec.KubernetesMonitoringSpec.DNSPolicy,
		NodeSelector:       instance.Spec.KubernetesMonitoringSpec.NodeSelector,
		ServiceAccountName: buildServiceAccountName(instance),
		Affinity:           buildAffinity(),
		Tolerations:        instance.Spec.KubernetesMonitoringSpec.Tolerations,
		PriorityClassName:  instance.Spec.KubernetesMonitoringSpec.PriorityClassName,
		Volumes:            buildVolumes(instance),
		ImagePullSecrets:   buildImagePullSecrets(instance),
	}
}

func buildImagePullSecrets(instance *v1alpha1.DynaKube) []corev1.LocalObjectReference {
	return []corev1.LocalObjectReference{
		buildPullSecret(instance),
	}
}

func buildContainer(instance *v1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) corev1.Container {
	return corev1.Container{
		Name:            v1alpha1.OperatorName,
		Image:           buildImage(instance),
		Resources:       buildResources(instance),
		ImagePullPolicy: corev1.PullAlways,
		Env:             buildEnvs(instance, tenantInfo, kubeSystemUID),
		Args:            buildArgs(instance),
		VolumeMounts:    buildVolumeMounts(instance),
		ReadinessProbe:  buildReadinessProbe(),
		LivenessProbe:   buildLivenessProbe(),
	}
}

func buildServiceAccountName(instance *v1alpha1.DynaKube) string {
	if instance.Spec.KubernetesMonitoringSpec.ServiceAccountName != "" {
		return instance.Spec.KubernetesMonitoringSpec.ServiceAccountName
	}
	return MonitoringServiceAccount
}

func buildAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: buildNodeAffinity(),
	}
}

func buildNodeAffinity() *corev1.NodeAffinity {
	return &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: buildNodeSelectorForAffinity(),
	}
}

func buildNodeSelectorForAffinity() *corev1.NodeSelector {
	return &corev1.NodeSelector{
		NodeSelectorTerms: []corev1.NodeSelectorTerm{
			{MatchExpressions: buildKubernetesBetaArchExpression()},
			{MatchExpressions: buildKubernetesArchExpression()},
		},
	}
}

func buildKubernetesBetaArchExpression() []corev1.NodeSelectorRequirement {
	return buildKubernetesExpression(KubernetesBetaArch, KubernetesBetaOs)
}

func buildKubernetesArchExpression() []corev1.NodeSelectorRequirement {
	return buildKubernetesExpression(KubernetesArch, KubernetesOs)
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

func buildVolumes(instance *v1alpha1.DynaKube) []corev1.Volume {
	return append([]corev1.Volume{}, buildCustomPropertiesVolume(instance)...)
}

func buildCustomPropertiesVolume(instance *v1alpha1.DynaKube) []corev1.Volume {
	customProperties := instance.Spec.KubernetesMonitoringSpec.CustomProperties
	if isCustomPropertiesNilOrEmpty(customProperties) {
		return nil
	}
	return []corev1.Volume{{
		Name: customproperties.VolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: buildSecretVolumeSource(
				buildSecretName(instance),
				customproperties.DataKey,
				customproperties.DataPath,
			),
		},
	},
	}
}

func isCustomPropertiesNilOrEmpty(customProperties *v1alpha1.DynaKubeValueSource) bool {
	return customProperties == nil ||
		(customProperties.Value == "" &&
			customProperties.ValueFrom == "")
}

func buildSecretVolumeSource(name string, key string, path string) *corev1.SecretVolumeSource {
	return &corev1.SecretVolumeSource{
		SecretName: name,
		Items: []corev1.KeyToPath{
			{Key: key, Path: path},
		},
	}
}

func buildSecretName(instance *v1alpha1.DynaKube) string {
	valueFrom := instance.Spec.KubernetesMonitoringSpec.CustomProperties.ValueFrom
	if valueFrom == "" {
		valueFrom = fmt.Sprintf("%s-kubernetes-monitoring%s", instance.Name, customproperties.Suffix)
	}
	return valueFrom
}

func buildEnvs(instance *v1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) []corev1.EnvVar {
	return appendProxySettingsEnvVar(instance,
		append(instance.Spec.KubernetesMonitoringSpec.Env,
			buildDefaultEnvVars(instance, tenantInfo, kubeSystemUID)...),
	)
}

func appendProxySettingsEnvVar(instance *v1alpha1.DynaKube, envVars []corev1.EnvVar) []corev1.EnvVar {
	proxy := instance.Spec.Proxy
	if isProxyNilOrEmpty(proxy) {
		return envVars
	}
	return append(envVars, buildProxyEnvVar(proxy))
}

func buildProxyEnvVar(proxy *v1alpha1.DynaKubeProxy) corev1.EnvVar {
	if proxy.ValueFrom != "" {
		return buildProxyEnvVarFromValueSource(proxy)
	} else {
		return buildProxyEnvVarFromValue(proxy)
	}
}

func buildProxyEnvVarFromValueSource(proxy *v1alpha1.DynaKubeProxy) corev1.EnvVar {
	return corev1.EnvVar{
		Name: ProxyEnv,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: proxy.ValueFrom},
				Key:                  ProxyKey,
			},
		},
	}
}

func buildProxyEnvVarFromValue(proxy *v1alpha1.DynaKubeProxy) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  ProxyEnv,
		Value: proxy.Value,
	}
}

func buildDefaultEnvVars(instance *v1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: DtTenant, Value: tenantInfo.ID},
		{Name: DtToken, Value: tenantInfo.Token},
		{Name: DtServer, Value: tenantInfo.CommunicationEndpoint},
		{Name: DtCapabilities, Value: CapabilityEnv},
		{Name: DtIdSeedNamespace, Value: instance.Namespace},
		{Name: DtIdSeedClusterId, Value: string(kubeSystemUID)},
	}
}

func buildArgs(instance *v1alpha1.DynaKube) []string {
	return appendActivationGroupArg(instance,
		appendProxySettingsArg(instance,
			appendNetworkZoneArg(instance,
				append(instance.Spec.KubernetesMonitoringSpec.Args,
					buildDefaultArgs()...),
			)))
}

func buildDefaultArgs() []string {
	return []string{
		DtTenantArg,
		DtTokenArg,
		DtServerArg,
		DtCapabilitiesArg,
	}
}

func appendActivationGroupArg(instance *v1alpha1.DynaKube, args []string) []string {
	group := instance.Spec.KubernetesMonitoringSpec.Group
	if group == "" {
		return args
	}
	return append(args, buildActivationGroupArg(group))
}

func buildActivationGroupArg(group string) string {
	return fmt.Sprintf(`--group="%s"`, group)
}

func appendProxySettingsArg(instance *v1alpha1.DynaKube, args []string) []string {
	proxy := instance.Spec.Proxy
	if isProxyNilOrEmpty(proxy) {
		return args
	}
	return append(args, ProxyArg)
}

func isProxyNilOrEmpty(proxy *v1alpha1.DynaKubeProxy) bool {
	return proxy == nil || (proxy.Value == "" && proxy.ValueFrom == "")
}

func appendNetworkZoneArg(instance *v1alpha1.DynaKube, args []string) []string {
	if instance.Spec.NetworkZone != "" {
		return append(args, buildNetworkZoneArg(instance))
	}
	return args
}

func buildNetworkZoneArg(instance *v1alpha1.DynaKube) string {
	return fmt.Sprintf(`--networkzone="%s"`, instance.Spec.NetworkZone)
}

func buildVolumeMounts(instance *v1alpha1.DynaKube) []corev1.VolumeMount {
	customProperties := instance.Spec.KubernetesMonitoringSpec.CustomProperties
	if isCustomPropertiesNilOrEmpty(customProperties) {
		return nil
	}
	return []corev1.VolumeMount{{
		ReadOnly:  true,
		Name:      customproperties.VolumeName,
		MountPath: customproperties.MountPath,
	}}
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

func BuildLabelsFromInstance(instance *v1alpha1.DynaKube) map[string]string {
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
