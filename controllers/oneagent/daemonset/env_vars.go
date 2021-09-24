package daemonset

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/dtclient"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	dtNodeName  = "DT_K8S_NODE_NAME"
	dtClusterId = "DT_K8S_CLUSTER_ID"

	oneagentDownloadToken             = "ONEAGENT_INSTALLER_DOWNLOAD_TOKEN"
	oneagentInstallScript             = "ONEAGENT_INSTALLER_SCRIPT_URL"
	oneagentSkipCertCheck             = "ONEAGENT_INSTALLER_SKIP_CERT_CHECK"
	oneagentDisableContainerInjection = "ONEAGENT_DISABLE_CONTAINER_INJECTION"
	oneagentReadOnlyMode              = "ONEAGENT_READ_ONLY_MODE"

	proxy = "https_proxy"
)

func (dsInfo *builderInfo) environmentVariables() []corev1.EnvVar {
	environmentVariables := dsInfo.hostInjectSpec.Env
	envVarMap := envVarsToMap(environmentVariables)
	envVarMap = setDefaultValueSource(envVarMap, dtNodeName, &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}})
	envVarMap = setDefaultValue(envVarMap, dtClusterId, dsInfo.clusterId)

	if !dsInfo.useImmutableImage() {
		envVarMap = setDefaultValueSource(envVarMap, oneagentDownloadToken, &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: dsInfo.instance.Tokens()},
				Key:                  dtclient.DynatracePaasToken,
			},
		})
		envVarMap = setDefaultValue(envVarMap, oneagentInstallScript, dsInfo.installerUrl())
		envVarMap = setDefaultValue(envVarMap, oneagentSkipCertCheck, strconv.FormatBool(dsInfo.instance.Spec.SkipCertCheck))
	}

	if dsInfo.hasProxy() {
		envVarMap = dsInfo.setDefaultProxy(envVarMap)
	}

	return mapToArray(envVarMap)
}

func (dsInfo *HostMonitoring) appendInfraMonEnvVars(daemonset *appsv1.DaemonSet) {
	envVars := daemonset.Spec.Template.Spec.Containers[0].Env
	envVarMap := envVarsToMap(envVars)
	envVarMap = setDefaultValue(envVarMap, oneagentDisableContainerInjection, "true")
	envVarMap = setDefaultValue(envVarMap, oneagentReadOnlyMode, "true")

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

func (dsInfo *builderInfo) installerUrl() string {
	return fmt.Sprintf("%s/v1/deployment/installer/agent/unix/default/latest?arch=x86&flavor=default", dsInfo.instance.Spec.APIURL)
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
