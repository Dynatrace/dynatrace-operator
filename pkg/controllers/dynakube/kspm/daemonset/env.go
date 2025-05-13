package daemonset

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
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
	}

	if dk.ActiveGate().HasCaCert() {
		envs = append(envs, getCertEnv())
	}

	envs = append(envs, dk.KSPM().Env...)

	return envs
}

func getActiveGateEndpointTemplate(dk dynakube.DynaKube, tenantUUID string) string {
	activeGateEndpointTemplate := "https://%s.%s/e/%s/api/v2/kubernetes/node-config"
	serviceName := capability.BuildServiceName(dk.Name)

	return fmt.Sprintf(activeGateEndpointTemplate, serviceName, dk.Namespace, tenantUUID)
}
