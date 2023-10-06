package daemonset

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/parametermap"
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

const customEnvPriority = 2
const defaultEnvPriority = 1

func (dsInfo *builderInfo) environmentVariables() []corev1.EnvVar {
	envMap := parametermap.NewMap(parametermap.WithPriority(defaultEnvPriority))

	if dsInfo.hostInjectSpec != nil {
		parametermap.Append(envMap, dsInfo.hostInjectSpec.Env, parametermap.WithPriority(customEnvPriority))
	}

	addNodeNameEnv(envMap)
	dsInfo.addClusterIDEnv(envMap)
	dsInfo.addDeploymentMetadataEnv(envMap)
	dsInfo.addOperatorVersionInfoEnv(envMap)
	dsInfo.addConnectionInfoEnvs(envMap)
	dsInfo.addProxyEnv(envMap)
	dsInfo.addReadOnlyEnv(envMap)

	return envMap.AsEnvVars()
}

func addNodeNameEnv(envVarMap *parametermap.Map) {
	addDefaultValueSource(envVarMap, dtNodeName, &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}})
}

func (dsInfo *builderInfo) addClusterIDEnv(envVarMap *parametermap.Map) {
	addDefaultValue(envVarMap, dtClusterId, dsInfo.clusterID)
}

func (dsInfo *builderInfo) addDeploymentMetadataEnv(envVarMap *parametermap.Map) {
	addDefaultValueSource(envVarMap, deploymentmetadata.EnvDtDeploymentMetadata, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: deploymentmetadata.GetDeploymentMetadataConfigMapName(dsInfo.dynakube.Name),
		},
		Key:      deploymentmetadata.OneAgentMetadataKey,
		Optional: address.Of(false),
	}})
}

func (dsInfo *builderInfo) addOperatorVersionInfoEnv(envVarMap *parametermap.Map) {
	addDefaultValueSource(envVarMap, deploymentmetadata.EnvDtOperatorVersion, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: deploymentmetadata.GetDeploymentMetadataConfigMapName(dsInfo.dynakube.Name),
		},
		Key:      deploymentmetadata.OperatorVersionKey,
		Optional: address.Of(false),
	}})
}

func (dsInfo *builderInfo) addConnectionInfoEnvs(envVarMap *parametermap.Map) {
	addDefaultValueSource(envVarMap, connectioninfo.EnvDtTenant, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: dsInfo.dynakube.OneAgentConnectionInfoConfigMapName(),
		},
		Key:      connectioninfo.TenantUUIDName,
		Optional: address.Of(false),
	}})
	addDefaultValueSource(envVarMap, connectioninfo.EnvDtServer, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: dsInfo.dynakube.OneAgentConnectionInfoConfigMapName(),
		},
		Key:      connectioninfo.CommunicationEndpointsName,
		Optional: address.Of(false),
	}})
}

func (dsInfo *builderInfo) addProxyEnv(envVarMap *parametermap.Map) {
	if !dsInfo.hasProxy() {
		return
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
}

func (dsInfo *builderInfo) addReadOnlyEnv(envVarMap *parametermap.Map) {
	if dsInfo.dynakube != nil && dsInfo.dynakube.NeedsReadOnlyOneAgents() {
		addDefaultValue(envVarMap, oneagentReadOnlyMode, "true")
	}
}

func (dsInfo *HostMonitoring) appendInfraMonEnvVars(daemonset *appsv1.DaemonSet) {
	envVars := parametermap.NewMap()
	parametermap.Append(envVars, daemonset.Spec.Template.Spec.Containers[0].Env)
	addDefaultValue(envVars, oneagentDisableContainerInjection, "true")
	daemonset.Spec.Template.Spec.Containers[0].Env = envVars.AsEnvVars()
}

func addDefaultValue(envVarMap *parametermap.Map, name string, value string) {
	parametermap.Append(envVarMap, corev1.EnvVar{
		Name:  name,
		Value: value,
	})
}

func addDefaultValueSource(envVarMap *parametermap.Map, name string, value *corev1.EnvVarSource) {
	parametermap.Append(envVarMap, corev1.EnvVar{
		Name:      name,
		ValueFrom: value,
	})
}
