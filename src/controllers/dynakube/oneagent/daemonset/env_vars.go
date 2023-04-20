package daemonset

import (
	"sort"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	dtNodeName      = "DT_K8S_NODE_NAME"
	dtClusterId     = "DT_K8S_CLUSTER_ID"
	dtCommunication = "DT_COMMUNICATION"

	oneagentDisableContainerInjection = "ONEAGENT_DISABLE_CONTAINER_INJECTION"
	oneagentReadOnlyMode              = "ONEAGENT_READ_ONLY_MODE"

	proxy = "https_proxy"
)

func (dsInfo *builderInfo) environmentVariables() []corev1.EnvVar {
	environmentVariables := make([]corev1.EnvVar, 0)

	if dsInfo.hostInjectSpec != nil {
		environmentVariables = dsInfo.hostInjectSpec.Env
	}

	envVarMap := envVarsToMap(environmentVariables)
	envVarMap = addNodeNameEnv(envVarMap)

	envVarMap = dsInfo.addClusterIDEnv(envVarMap)
	envVarMap = dsInfo.addDeploymentMetadataEnv(envVarMap)
	envVarMap = dsInfo.addConnectionInfoEnvs(envVarMap)
	envVarMap = dsInfo.addProxyEnv(envVarMap)
	envVarMap = dsInfo.addReadOnlyEnv(envVarMap)

	return mapToArray(envVarMap)
}

func addNodeNameEnv(envVarMap map[string]corev1.EnvVar) map[string]corev1.EnvVar {
	return addDefaultValueSource(envVarMap, dtNodeName, &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}})
}

func (dsInfo *builderInfo) addClusterIDEnv(envVarMap map[string]corev1.EnvVar) map[string]corev1.EnvVar {
	return addDefaultValue(envVarMap, dtClusterId, dsInfo.clusterID)
}

func (dsInfo *builderInfo) addDeploymentMetadataEnv(envVarMap map[string]corev1.EnvVar) map[string]corev1.EnvVar {
	return addDefaultValueSource(envVarMap, deploymentmetadata.EnvDtDeploymentMetadata, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: deploymentmetadata.GetDeploymentMetadataConfigMapName(dsInfo.dynakube.Name),
		},
		Key:      deploymentmetadata.OneAgentMetadataKey,
		Optional: address.Of(false),
	}})
}

func (dsInfo *builderInfo) addConnectionInfoEnvs(envVarMap map[string]corev1.EnvVar) map[string]corev1.EnvVar {
	envVarMap = addDefaultValueSource(envVarMap, connectioninfo.EnvDtTenant, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: dsInfo.dynakube.OneAgentConnectionInfoConfigMapName(),
		},
		Key:      connectioninfo.TenantUUIDName,
		Optional: address.Of(false),
	}})
	envVarMap = addDefaultValueSource(envVarMap, connectioninfo.EnvDtServer, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: dsInfo.dynakube.OneAgentConnectionInfoConfigMapName(),
		},
		Key:      connectioninfo.CommunicationEndpointsName,
		Optional: address.Of(false),
	}})
	return envVarMap
}

func (dsInfo *builderInfo) addProxyEnv(envVarMap map[string]corev1.EnvVar) map[string]corev1.EnvVar {
	if !dsInfo.hasProxy() {
		return envVarMap
	}
	if dsInfo.dynakube.Spec.Proxy.ValueFrom != "" {
		addDefaultValueSource(envVarMap, proxy, &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: dsInfo.dynakube.Spec.Proxy.ValueFrom},
				Key:                  dynatracev1beta1.ProxyKey,
			},
		})
	} else {
		addDefaultValue(envVarMap, proxy, dsInfo.dynakube.Spec.Proxy.Value)
	}
	return envVarMap
}

func (dsInfo *builderInfo) addReadOnlyEnv(envVarMap map[string]corev1.EnvVar) map[string]corev1.EnvVar {
	if dsInfo.dynakube != nil && dsInfo.dynakube.NeedsReadOnlyOneAgents() {
		envVarMap = addDefaultValue(envVarMap, oneagentReadOnlyMode, "true")
	}
	return envVarMap
}

func (dsInfo *HostMonitoring) appendInfraMonEnvVars(daemonset *appsv1.DaemonSet) {
	envVars := daemonset.Spec.Template.Spec.Containers[0].Env
	envVarMap := envVarsToMap(envVars)
	envVarMap = addDefaultValue(envVarMap, oneagentDisableContainerInjection, "true")
	daemonset.Spec.Template.Spec.Containers[0].Env = mapToArray(envVarMap)
}

func mapToArray(envVarMap map[string]corev1.EnvVar) []corev1.EnvVar {
	result := make([]corev1.EnvVar, 0)
	keys := make([]string, 0)

	for key := range envVarMap {
		keys = append(keys, key)
	}

	// Keys have to be sorted, because when the environment variables are not always in the same order the hash differs
	// In which case the daemonset appears as if it had changed, although it did not
	sort.Strings(keys)

	for _, key := range keys {
		result = append(result, envVarMap[key])
	}

	return result
}

func addDefaultValue(envVarMap map[string]corev1.EnvVar, name string, value string) map[string]corev1.EnvVar {
	if _, hasVar := envVarMap[name]; !hasVar {
		envVarMap[name] = corev1.EnvVar{
			Name:  name,
			Value: value,
		}
	}
	return envVarMap
}

func addDefaultValueSource(envVarMap map[string]corev1.EnvVar, name string, value *corev1.EnvVarSource) map[string]corev1.EnvVar {
	if _, hasVar := envVarMap[name]; !hasVar {
		envVarMap[name] = corev1.EnvVar{
			Name:      name,
			ValueFrom: value,
		}
	}
	return envVarMap
}

func envVarsToMap(variables []corev1.EnvVar) map[string]corev1.EnvVar {
	result := make(map[string]corev1.EnvVar)
	for _, env := range variables {
		result[env.Name] = env
	}
	return result
}
