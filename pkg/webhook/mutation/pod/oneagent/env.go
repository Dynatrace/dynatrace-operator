package oneagent

import (
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	corev1 "k8s.io/api/core/v1"
)

func addPreloadEnv(container *corev1.Container, installPath string) {
	preloadPath := filepath.Join(installPath, consts.LibAgentProcPath)

	ldPreloadEnv := env.FindEnvVar(container.Env, preloadEnv)
	if ldPreloadEnv != nil {
		if strings.Contains(ldPreloadEnv.Value, installPath) {
			return
		}

		ldPreloadEnv.Value = concatPreloadPaths(ldPreloadEnv.Value, preloadPath)
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
		if env.IsIn(container.Env, envName) {
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

func addInstallerInitEnvs(initContainer *corev1.Container, installer installerInfo) {
	initContainer.Env = append(initContainer.Env,
		corev1.EnvVar{Name: consts.AgentInstallerFlavorEnv, Value: installer.flavor}, // TODO: is this needed
		corev1.EnvVar{Name: consts.AgentInstallerTechEnv, Value: installer.technologies},
		corev1.EnvVar{Name: consts.AgentInstallPathEnv, Value: installer.installPath},
		corev1.EnvVar{Name: consts.AgentInstallerUrlEnv, Value: installer.installerURL},
		corev1.EnvVar{Name: consts.AgentInstallerVersionEnv, Value: installer.version},
		corev1.EnvVar{Name: consts.AgentInjectedEnv, Value: "true"},
	)
}

func addDeploymentMetadataEnv(container *corev1.Container, dk dynakube.DynaKube, clusterID string) {
	if env.IsIn(container.Env, dynatraceMetadataEnv) {
		return
	}

	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(clusterID, deploymentmetadata.GetOneAgentDeploymentType(dk))
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name:  dynatraceMetadataEnv,
			Value: deploymentMetadata.AsString(),
		})
}
