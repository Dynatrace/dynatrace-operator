package builder

import (
	"strings"

	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
)

func BuildActiveGatePodSpecs(instance *v1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) corev1.PodSpec {
	serviceAccount := MonitoringServiceAccount
	image := ActivegateImage
	activeGateSpec := &instance.Spec.KubernetesMonitoringSpec

	if activeGateSpec.ServiceAccountName != "" {
		serviceAccount = activeGateSpec.ServiceAccountName
	}
	if activeGateSpec.Image != "" {
		image = activeGateSpec.Image
	}
	if tenantInfo == nil {
		tenantInfo = &dtclient.TenantInfo{
			ID:        "",
			Token:     "",
			Endpoints: []string{},
		}
	}

	if activeGateSpec.Resources.Requests == nil {
		activeGateSpec.Resources.Requests = corev1.ResourceList{}
	}
	if _, hasCPUResource := activeGateSpec.Resources.Requests[corev1.ResourceCPU]; !hasCPUResource {
		// Set CPU resource to 1 * 10**(-1) Cores, e.g. 100mC
		activeGateSpec.Resources.Requests[corev1.ResourceCPU] = *resource.NewScaledQuantity(1, -1)
	}

	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:            ActivegateName,
			Image:           image,
			Resources:       activeGateSpec.Resources,
			ImagePullPolicy: corev1.PullAlways,
			Env:             buildEnvVars(instance, tenantInfo, kubeSystemUID),
			Args:            buildArgs(),
		}},
		DNSPolicy:          activeGateSpec.DNSPolicy,
		NodeSelector:       activeGateSpec.NodeSelector,
		ServiceAccountName: serviceAccount,
		Affinity:           buildAffinity(),
		Tolerations:        activeGateSpec.Tolerations,
		PriorityClassName:  activeGateSpec.PriorityClassName,
	}
}

func buildArgs() []string {
	return []string{
		DtTenantArg,
		DtTokenArg,
		DtServerArg,
		DtCapabilitiesArg,
	}
}

func buildEnvVars(instance *v1alpha1.DynaKube, tenantInfo *dtclient.TenantInfo, kubeSystemUID types.UID) []corev1.EnvVar {
	var capabilities []string

	if instance.Spec.KubernetesMonitoringSpec.Enabled {
		capabilities = append(capabilities, "kubernetes_monitoring")
	}

	return []corev1.EnvVar{
		{
			Name:  DtTenant,
			Value: tenantInfo.ID,
		},
		{
			Name:  DtToken,
			Value: tenantInfo.Token,
		},
		{
			Name:  DtServer,
			Value: tenantInfo.CommunicationEndpoint,
		},
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
	}
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

func BuildLabels(name string, labels map[string]string) map[string]string {
	result := BuildLabelsForQuery(name)
	for key, value := range labels {
		result[key] = value
	}
	return result
}

func BuildMergeLabels(labels ...map[string]string) map[string]string {
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
	ActivegateImage = "612044533526.dkr.ecr.us-east-1.amazonaws.com/activegate:latest"
	ActivegateName  = "dynatrace-operator"

	MonitoringServiceAccount = "dynatrace-activegate"

	KubernetesArch     = "kubernetes.io/arch"
	KubernetesOs       = "kubernetes.io/os"
	KubernetesBetaArch = "beta.kubernetes.io/arch"
	KubernetesBetaOs   = "beta.kubernetes.io/os"

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

	Comma = ","
)
