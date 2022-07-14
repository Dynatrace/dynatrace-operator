package oneagent_mutation

import (
	"fmt"
	"path/filepath"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/src/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	corev1 "k8s.io/api/core/v1"
)

func addPreloadEnv(container *corev1.Container, installPath string) {
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name:  preloadEnvVarName,
			Value: filepath.Join(installPath, config.LibAgentProcPath),
		})
}

func addNetworkZoneEnv(container *corev1.Container, networkZone string) {
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name:  networkZoneEnvVarName,
			Value: networkZone,
		},
	)
}

func addProxyEnv(container *corev1.Container) {
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name: proxyEnvVarName,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: config.AgentInitSecretName,
					},
					Key: dynatracev1beta1.ProxyKey,
				},
			},
		})
}

func addInstallerInitEnvs(initContainer *corev1.Container, installer installerInfo, volumeMode string) {
	initContainer.Env = append(initContainer.Env,
		corev1.EnvVar{Name: config.AgentInstallerFlavorEnv, Value: installer.flavor},
		corev1.EnvVar{Name: config.AgentInstallerTechEnv, Value: installer.technologies},
		corev1.EnvVar{Name: config.AgentInstallPathEnv, Value: installer.installPath},
		corev1.EnvVar{Name: config.AgentInstallerUrlEnv, Value: installer.installerURL},
		corev1.EnvVar{Name: config.AgentInstallModeEnv, Value: volumeMode},
		corev1.EnvVar{Name: config.AgentInjectedEnv, Value: "true"},
	)
}

func addContainerInfoInitEnv(initContainer *corev1.Container, containerIndex int, name string, image string) {
	log.Info("updating init container with new container", "name", name, "image", image)
	initContainer.Env = append(initContainer.Env,
		corev1.EnvVar{Name: getContainerNameEnv(containerIndex), Value: name},
		corev1.EnvVar{Name: getContainerImageEnv(containerIndex), Value: image})
}

func getContainerNameEnv(containerIndex int) string {
	return fmt.Sprintf(config.AgentContainerNameEnvTemplate, containerIndex)
}

func getContainerImageEnv(containerIndex int) string {
	return fmt.Sprintf(config.AgentContainerImageEnvTemplate, containerIndex)
}

func addDeploymentMetadataEnv(container *corev1.Container, dynakube dynatracev1beta1.DynaKube, clusterID string) {
	if kubeobjects.EnvVarIsIn(container.Env, dynatraceMetadataEnvVarName) {
		return
	}
	var deploymentMetadata *deploymentmetadata.DeploymentMetadata
	if dynakube.CloudNativeFullstackMode() {
		deploymentMetadata = deploymentmetadata.NewDeploymentMetadata(clusterID, daemonset.DeploymentTypeCloudNative)
	} else {
		deploymentMetadata = deploymentmetadata.NewDeploymentMetadata(clusterID, daemonset.DeploymentTypeApplicationMonitoring)
	}
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name:  dynatraceMetadataEnvVarName,
			Value: deploymentMetadata.AsString(),
		})
}
