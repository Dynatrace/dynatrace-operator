package daemonset

import (
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	dtNodeName  = "DT_K8S_NODE_NAME"
	dtClusterId = "DT_K8S_CLUSTER_ID"

	oneagentDisableContainerInjection = "ONEAGENT_DISABLE_CONTAINER_INJECTION"
	oneagentReadOnlyMode              = "ONEAGENT_READ_ONLY_MODE"

	proxy = "https_proxy"
)

func (dsInfo *builderInfo) environmentVariables() []corev1.EnvVar {
	environmentVariables := dsInfo.hostInjectSpec.Env
	envVarMap := envVarsToMap(environmentVariables)
	envVarMap = setDefaultValueSource(envVarMap, dtNodeName, &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}})
	envVarMap = setDefaultValue(envVarMap, dtClusterId, dsInfo.clusterId)

	if dsInfo.hasProxy() {
		envVarMap = dsInfo.setDefaultProxy(envVarMap)
	}

	return mapToArray(envVarMap)
}

func (dsInfo *HostMonitoring) appendInfraMonEnvVars(daemonset *appsv1.DaemonSet) {
	envVars := daemonset.Spec.Template.Spec.Containers[0].Env
	envVarMap := envVarsToMap(envVars)
	envVarMap = setDefaultValue(envVarMap, oneagentDisableContainerInjection, "true")
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

func (dsInfo *builderInfo) setDefaultProxy(envVarMap map[string]corev1.EnvVar) map[string]corev1.EnvVar {
	if dsInfo.instance.Spec.Proxy.ValueFrom != "" {
		setDefaultValueSource(envVarMap, proxy, &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: dsInfo.instance.Spec.Proxy.ValueFrom},
				Key:                  "proxy",
			},
		})
	} else {
		setDefaultValue(envVarMap, proxy, dsInfo.instance.Spec.Proxy.Value)
	}
	return envVarMap
}

func setDefaultValue(envVarMap map[string]corev1.EnvVar, name string, value string) map[string]corev1.EnvVar {
	if _, hasVar := envVarMap[name]; !hasVar {
		envVarMap[name] = corev1.EnvVar{
			Name:  name,
			Value: value,
		}
	}
	return envVarMap
}

func setDefaultValueSource(envVarMap map[string]corev1.EnvVar, name string, value *corev1.EnvVarSource) map[string]corev1.EnvVar {
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
