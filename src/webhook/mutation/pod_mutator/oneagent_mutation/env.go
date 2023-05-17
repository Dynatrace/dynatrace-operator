package oneagent_mutation

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	corev1 "k8s.io/api/core/v1"
)

func addPreloadEnv(container *corev1.Container, installPath string) {
	preloadPath := filepath.Join(installPath, config.LibAgentProcPath)

	env := kubeobjects.FindEnvVar(container.Env, preloadEnv)
	if env != nil {
		if strings.Contains(env.Value, installPath) {
			return
		}
		env.Value = concatPreloadPaths(env.Value, preloadPath)
	} else {
		container.Env = append(container.Env,
			corev1.EnvVar{
				Name:  preloadEnv,
				Value: preloadPath,
			})
	}
}

func concatPreloadPaths(originalPaths, additionalPath string) string {
	if strings.Contains(originalPaths, " ") {
		return originalPaths + " " + additionalPath
	} else {
		return originalPaths + ":" + additionalPath
	}
}

func addNetworkZoneEnv(container *corev1.Container, networkZone string) {
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name:  networkZoneEnv,
			Value: networkZone,
		},
	)
}

func addVersionDetectionEnvs(container *corev1.Container, labelMapping VersionLabelMapping) {
	for envName, fieldPath := range labelMapping {
		if kubeobjects.EnvVarIsIn(container.Env, envName) {
			continue
		}
		container.Env = append(container.Env,
			corev1.EnvVar{
				Name: envName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: fieldPath,
					},
				},
			},
		)
	}
}

func addProxyEnv(container *corev1.Container) {
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name: proxyEnv,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: config.AgentInitSecretName,
					},
					Key: dynatracev1.ProxyKey,
				},
			},
		})
}

func addInstallerInitEnvs(initContainer *corev1.Container, installer installerInfo, dynakube dynatracev1.DynaKube) {
	initContainer.Env = append(initContainer.Env,
		corev1.EnvVar{Name: config.AgentInstallerFlavorEnv, Value: installer.flavor},
		corev1.EnvVar{Name: config.AgentInstallerTechEnv, Value: installer.technologies},
		corev1.EnvVar{Name: config.AgentInstallPathEnv, Value: installer.installPath},
		corev1.EnvVar{Name: config.AgentInstallerUrlEnv, Value: installer.installerURL},
		corev1.EnvVar{Name: config.AgentInstallerVersionEnv, Value: installer.version},
		corev1.EnvVar{Name: config.AgentInstallModeEnv, Value: getVolumeMode(dynakube)},
		corev1.EnvVar{Name: config.AgentReadonlyCSI, Value: strconv.FormatBool(dynakube.FeatureReadOnlyCsiVolume())},
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

func addDeploymentMetadataEnv(container *corev1.Container, dynakube dynatracev1.DynaKube, clusterID string) {
	if kubeobjects.EnvVarIsIn(container.Env, dynatraceMetadataEnv) {
		return
	}

	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(clusterID, deploymentmetadata.GetOneAgentDeploymentType(dynakube))
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name:  dynatraceMetadataEnv,
			Value: deploymentMetadata.AsString(),
		})
}
