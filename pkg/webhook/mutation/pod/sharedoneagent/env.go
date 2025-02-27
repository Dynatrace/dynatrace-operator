package sharedoneagent

import (
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	corev1 "k8s.io/api/core/v1"
)

const PreloadEnv = "LD_PRELOAD"

func AddPreloadEnv(container *corev1.Container, installPath string) { // TODO gakr could be combined
	preloadPath := filepath.Join(installPath, consts.LibAgentProcPath)

	ldPreloadEnv := env.FindEnvVar(container.Env, PreloadEnv)
	if ldPreloadEnv != nil {
		if strings.Contains(ldPreloadEnv.Value, installPath) {
			return
		}

		ldPreloadEnv.Value = concatPreloadPaths(ldPreloadEnv.Value, preloadPath)
	} else {
		container.Env = append(container.Env,
			corev1.EnvVar{
				Name:  PreloadEnv,
				Value: preloadPath,
			})
	}
}

func AddDeploymentMetadataEnv(container *corev1.Container, dk dynakube.DynaKube, clusterID string) {
	if env.IsIn(container.Env, deploymentmetadata.EnvDtDeploymentMetadata) {
		return
	}

	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(clusterID, deploymentmetadata.GetOneAgentDeploymentType(dk))
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name:  deploymentmetadata.EnvDtDeploymentMetadata,
			Value: deploymentMetadata.AsString(),
		})
}

func concatPreloadPaths(originalPaths, additionalPath string) string {
	if strings.Contains(originalPaths, " ") {
		return originalPaths + " " + additionalPath
	} else {
		return originalPaths + ":" + additionalPath
	}
}
