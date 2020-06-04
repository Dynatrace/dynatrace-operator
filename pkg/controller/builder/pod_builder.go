package builder

import (
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

var logger = log.Log.WithName("builder.pod_builder")

func BuildActiveGatePodSpecs(
	acitveGatePodSpec *v1alpha1.ActiveGateSpec,
	tenantInfo *dtclient.TenantInfo) corev1.PodSpec {
	serviceaccount := ACTIVEGATE_NAME
	image := ACTIVEGATE_IMAGE

	if len(acitveGatePodSpec.ServiceAccountName) > 0 {
		serviceaccount = acitveGatePodSpec.ServiceAccountName
	}
	if len(acitveGatePodSpec.Image) > 0 {
		image = acitveGatePodSpec.Image
	}
	if tenantInfo == nil {
		tenantInfo = &dtclient.TenantInfo{
			ID:        "",
			Token:     "",
			Endpoints: []string{},
		}
	}

	return corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:            ACTIVEGATE_NAME,
			Image:           image,
			Resources:       acitveGatePodSpec.Resources,
			ImagePullPolicy: corev1.PullAlways,
			Env:             buildEnvVars(acitveGatePodSpec, tenantInfo),
			Args:            buildArgs(),
		}},
		DNSPolicy:          acitveGatePodSpec.DNSPolicy,
		NodeSelector:       acitveGatePodSpec.NodeSelector,
		ServiceAccountName: serviceaccount,
		HostNetwork:        true,
		HostPID:            true,
		HostIPC:            true,
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: IMAGE_PULL_SECRET},
		},
		Affinity:          buildAffinity(),
		Tolerations:       acitveGatePodSpec.Tolerations,
		PriorityClassName: acitveGatePodSpec.PriorityClassName,
	}
}

func buildArgs() []string {
	return []string{
		DT_TENANT_ARG,
		DT_TOKEN_ARG,
		DT_SERVER_ARG,
		DT_CAPABILITIES_ARG,
	}
}

func buildEnvVars(acitveGatePodSpec *v1alpha1.ActiveGateSpec, tenantInfo *dtclient.TenantInfo) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  DT_TENANT,
			Value: tenantInfo.ID,
		},
		{
			Name:  DT_TOKEN,
			Value: tenantInfo.Token,
		},
		{
			Name:  DT_SERVER,
			Value: tenantInfo.CommunicationEndpoint,
		},
		{
			Name:  DT_CAPABILITIES,
			Value: strings.Join(acitveGatePodSpec.Capabilities, COMMA),
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
								Key:      KUBERNETES_BETA_ARCH,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{AMD64, ARM64},
							},
							{
								Key:      KUBERNETES_BETA_OS,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{LINUX},
							},
						},
					},
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      KUBERNETES_ARCH,
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{AMD64, ARM64},
							},
							{
								Key:      KUBERNETES_OS,
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

const (
	ACTIVEGATE_IMAGE  = "612044533526.dkr.ecr.us-east-1.amazonaws.com/activegate:latest"
	ACTIVEGATE_NAME   = "dynatrace-activegate-operator"
	IMAGE_PULL_SECRET = "aws-registry"

	KUBERNETES_ARCH      = "kubernetes.io/arch"
	KUBERNETES_OS        = "kubernetes.io/os"
	KUBERNETES_BETA_ARCH = "beta.kubernetes.io/arch"
	KUBERNETES_BETA_OS   = "beta.kubernetes.io/os"

	AMD64 = "amd64"
	ARM64 = "arm64"
	LINUX = "linux"

	DT_TENANT       = "DT_TENANT"
	DT_SERVER       = "DT_SERVER"
	DT_TOKEN        = "DT_TOKEN"
	DT_CAPABILITIES = "DT_CAPABILITIES"

	DT_TENANT_ARG       = "--tenant=$(DT_TENANT)"
	DT_TOKEN_ARG        = "--token=$(DT_TOKEN)"
	DT_SERVER_ARG       = "--server=$(DT_SERVER)"
	DT_CAPABILITIES_ARG = "--enable=$(DT_CAPABILITIES)"

	COMMA = ","
)
