package daemonset

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	dtNodeName      = "DT_K8S_NODE_NAME"
	dtClusterId     = "DT_K8S_CLUSTER_ID"
	dtCommunication = "DT_COMMUNICATION"

	oneagentDisableContainerInjection = "ONEAGENT_DISABLE_CONTAINER_INJECTION"
	oneagentReadOnlyMode              = "ONEAGENT_READ_ONLY_MODE"

	proxyEnv = "https_proxy"
)

const customEnvPriority = prioritymap.HighPriority
const defaultEnvPriority = prioritymap.DefaultPriority

func (dsInfo *builderInfo) environmentVariables() ([]corev1.EnvVar, error) {
	envMap := prioritymap.New(prioritymap.WithPriority(defaultEnvPriority))

	if dsInfo.hostInjectSpec != nil {
		prioritymap.Append(envMap, dsInfo.hostInjectSpec.Env, prioritymap.WithPriority(customEnvPriority))
	}

	addNodeNameEnv(envMap)
	dsInfo.addClusterIDEnv(envMap)
	dsInfo.addDeploymentMetadataEnv(envMap)
	dsInfo.addOperatorVersionInfoEnv(envMap)
	dsInfo.addConnectionInfoEnvs(envMap)
	dsInfo.addReadOnlyEnv(envMap)

	isProxyAsEnvVarDeprecated, err := IsProxyAsEnvVarDeprecated(dsInfo.dynakube.OneAgentVersion())
	if err != nil {
		return []corev1.EnvVar{}, err
	}
	if !isProxyAsEnvVarDeprecated {
		// deprecated
		dsInfo.addProxyEnv(envMap)
	}

	return envMap.AsEnvVars(), nil
}

func addNodeNameEnv(envVarMap *prioritymap.Map) {
	addDefaultValueSource(envVarMap, dtNodeName, &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}})
}

func (dsInfo *builderInfo) addClusterIDEnv(envVarMap *prioritymap.Map) {
	addDefaultValue(envVarMap, dtClusterId, dsInfo.clusterID)
}

func (dsInfo *builderInfo) addDeploymentMetadataEnv(envVarMap *prioritymap.Map) {
	addDefaultValueSource(envVarMap, deploymentmetadata.EnvDtDeploymentMetadata, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: deploymentmetadata.GetDeploymentMetadataConfigMapName(dsInfo.dynakube.Name),
		},
		Key:      deploymentmetadata.OneAgentMetadataKey,
		Optional: address.Of(false),
	}})
}

func (dsInfo *builderInfo) addOperatorVersionInfoEnv(envVarMap *prioritymap.Map) {
	addDefaultValueSource(envVarMap, deploymentmetadata.EnvDtOperatorVersion, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: deploymentmetadata.GetDeploymentMetadataConfigMapName(dsInfo.dynakube.Name),
		},
		Key:      deploymentmetadata.OperatorVersionKey,
		Optional: address.Of(false),
	}})
}

func (dsInfo *builderInfo) addConnectionInfoEnvs(envVarMap *prioritymap.Map) {
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

// deprecated
func (dsInfo *builderInfo) addProxyEnv(envVarMap *prioritymap.Map) {
	if !dsInfo.hasProxy() {
		return
	}
	if dsInfo.dynakube.Spec.Proxy.ValueFrom != "" {
		addDefaultValueSource(envVarMap, proxyEnv, &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: dsInfo.dynakube.Spec.Proxy.ValueFrom},
				Key:                  dynatracev1beta1.ProxyKey,
			},
		})
	} else {
		addDefaultValue(envVarMap, proxyEnv, dsInfo.dynakube.Spec.Proxy.Value)
	}
}

func (dsInfo *builderInfo) addReadOnlyEnv(envVarMap *prioritymap.Map) {
	if dsInfo.dynakube != nil && dsInfo.dynakube.NeedsReadOnlyOneAgents() {
		addDefaultValue(envVarMap, oneagentReadOnlyMode, "true")
	}
}

func (dsInfo *HostMonitoring) appendInfraMonEnvVars(daemonset *appsv1.DaemonSet) {
	envVars := prioritymap.New()
	prioritymap.Append(envVars, daemonset.Spec.Template.Spec.Containers[0].Env)
	addDefaultValue(envVars, oneagentDisableContainerInjection, "true")
	daemonset.Spec.Template.Spec.Containers[0].Env = envVars.AsEnvVars()
}

func addDefaultValue(envVarMap *prioritymap.Map, name string, value string) {
	prioritymap.Append(envVarMap, corev1.EnvVar{
		Name:  name,
		Value: value,
	})
}

func addDefaultValueSource(envVarMap *prioritymap.Map, name string, value *corev1.EnvVarSource) {
	prioritymap.Append(envVarMap, corev1.EnvVar{
		Name:      name,
		ValueFrom: value,
	})
}

const (
	// starting with this version, OneAgent allows mounting proxy as file, therefore
	// enabling us to deprecate the env var/arg approach (which is non security compliant)
	ProxyAsEnvVarDeprecatedVersion = "1.273.0.0-0"
)

func IsProxyAsEnvVarDeprecated(oneAgentVersion string) (bool, error) {
	if oneAgentVersion == "" {
		return false, nil
	}
	runningVersion, err := version.ExtractSemanticVersion(oneAgentVersion)
	if err != nil {
		return false, err
	}
	versionConstraint, err := version.ExtractSemanticVersion(ProxyAsEnvVarDeprecatedVersion)
	if err != nil {
		return false, err
	}

	result := version.CompareSemanticVersions(runningVersion, versionConstraint)

	// if current OneAgent version is older than fix version
	if result < 0 {
		return false, nil
	}
	return true, nil
}
