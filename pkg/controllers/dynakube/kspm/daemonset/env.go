package daemonset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	corev1 "k8s.io/api/core/v1"
)

const (
	nodeNameEnv           = "KUBERNETES_NODE_NAME"
	nodeRootEnv           = "KUBERNETES_NODE_ROOT"
	activeGateEndpointEnv = "DT_ACTIVEGATE_ENDPOINT"
	tokenEnv              = "DT_K8S_NODE_CONFIGURATION_COLLECTOR_TOKEN"
)

func getEnvs(dk dynakube.DynaKube, tenantUUID string) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name: nodeNameEnv,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name:  nodeRootEnv,
			Value: nodeRootMountPath,
		},
		{
			Name:  activeGateEndpointEnv,
			Value: getActiveGateEndpointTemplate(dk, tenantUUID),
		},
		{
			Name: tokenEnv, // TODO: Remove once reading from file is supported
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: dk.KSPM().GetTokenSecretName(),
					},
					Key: kspm.TokenSecretKey,
				},
			},
		},
	}

	if needsCerts(dk) {
		envs = append(envs, getCertEnv())
	}

	envs = append(envs, dk.KSPM().Env...)

	return envs
}

func getActiveGateEndpointTemplate(dk dynakube.DynaKube, tenantUUID string) string {
	activeGateEndpointTemplate := "https://%s.%s/e/%s/api/v2/kubernetes/node-config"
	serviceName := capability.BuildServiceName(dk.Name, agconsts.MultiActiveGateName)

	return fmt.Sprintf(activeGateEndpointTemplate, serviceName, dk.Namespace, tenantUUID)
}
