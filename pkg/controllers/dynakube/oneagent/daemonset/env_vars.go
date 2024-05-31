package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	dtNodeName  = "DT_K8S_NODE_NAME"
	dtClusterId = "DT_K8S_CLUSTER_ID"

	oneagentDisableContainerInjection = "ONEAGENT_DISABLE_CONTAINER_INJECTION"
	oneagentReadOnlyMode              = "ONEAGENT_READ_ONLY_MODE"

	proxyEnv = "https_proxy"

	// ProxyAsEnvVarDeprecatedVersion holds the version after which OneAgent allows mounting proxy as file, therefore
	// enabling us to deprecate the env var/arg approach (which is non security compliant)
	ProxyAsEnvVarDeprecatedVersion = "1.273.0.0-0"
)

const customEnvPriority = prioritymap.HighPriority
const defaultEnvPriority = prioritymap.DefaultPriority

func (b *builder) environmentVariables() ([]corev1.EnvVar, error) {
	envMap := prioritymap.New(prioritymap.WithPriority(defaultEnvPriority))

	if b.hostInjectSpec != nil {
		prioritymap.Append(envMap, b.hostInjectSpec.Env, prioritymap.WithPriority(customEnvPriority))
	}

	addNodeNameEnv(envMap)
	b.addClusterIDEnv(envMap)
	b.addDeploymentMetadataEnv(envMap)
	b.addOperatorVersionInfoEnv(envMap)
	b.addConnectionInfoEnvs(envMap)
	b.addReadOnlyEnv(envMap)

	isProxyAsEnvDeprecated, err := isProxyAsEnvVarDeprecated(b.dk.OneAgentVersion())
	if err != nil {
		return []corev1.EnvVar{}, err
	}

	if !isProxyAsEnvDeprecated {
		// deprecated
		b.addProxyEnv(envMap)
	}

	return envMap.AsEnvVars(), nil
}

func addNodeNameEnv(envVarMap *prioritymap.Map) {
	addDefaultValueSource(envVarMap, dtNodeName, &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}})
}

func (b *builder) addClusterIDEnv(envVarMap *prioritymap.Map) {
	addDefaultValue(envVarMap, dtClusterId, b.clusterID)
}

func (b *builder) addDeploymentMetadataEnv(envVarMap *prioritymap.Map) {
	addDefaultValueSource(envVarMap, deploymentmetadata.EnvDtDeploymentMetadata, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: deploymentmetadata.GetDeploymentMetadataConfigMapName(b.dk.Name),
		},
		Key:      deploymentmetadata.OneAgentMetadataKey,
		Optional: address.Of(false),
	}})
}

func (b *builder) addOperatorVersionInfoEnv(envVarMap *prioritymap.Map) {
	addDefaultValueSource(envVarMap, deploymentmetadata.EnvDtOperatorVersion, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: deploymentmetadata.GetDeploymentMetadataConfigMapName(b.dk.Name),
		},
		Key:      deploymentmetadata.OperatorVersionKey,
		Optional: address.Of(false),
	}})
}

func (b *builder) addConnectionInfoEnvs(envVarMap *prioritymap.Map) {
	addDefaultValueSource(envVarMap, connectioninfo.EnvDtTenant, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: b.dk.OneAgentConnectionInfoConfigMapName(),
		},
		Key:      connectioninfo.TenantUUIDKey,
		Optional: address.Of(false),
	}})
	addDefaultValueSource(envVarMap, connectioninfo.EnvDtServer, &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: b.dk.OneAgentConnectionInfoConfigMapName(),
		},
		Key:      connectioninfo.CommunicationEndpointsKey,
		Optional: address.Of(false),
	}})
}

// deprecated
func (b *builder) addProxyEnv(envVarMap *prioritymap.Map) {
	if !b.hasProxy() {
		return
	}

	if b.dk.Spec.Proxy.ValueFrom != "" {
		addDefaultValueSource(envVarMap, proxyEnv, &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: b.dk.Spec.Proxy.ValueFrom},
				Key:                  dynatracev1beta2.ProxyKey,
			},
		})
	} else {
		addDefaultValue(envVarMap, proxyEnv, b.dk.Spec.Proxy.Value)
	}
}

func (b *builder) addReadOnlyEnv(envVarMap *prioritymap.Map) {
	if b.dk != nil && b.dk.NeedsReadOnlyOneAgents() {
		addDefaultValue(envVarMap, oneagentReadOnlyMode, "true")
	}
}

func (b *hostMonitoring) appendInfraMonEnvVars(daemonset *appsv1.DaemonSet) {
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

func isProxyAsEnvVarDeprecated(oneAgentVersion string) (bool, error) {
	if oneAgentVersion == "" || oneAgentVersion == string(status.CustomImageVersionSource) {
		// If the version is unknown or from a custom image, then we don't care about deprecation.
		return true, nil
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

	// if a current OneAgent version is older than fix version
	if result < 0 {
		return false, nil
	}

	return true, nil
}
