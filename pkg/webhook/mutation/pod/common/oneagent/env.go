package oneagent

import (
	"path/filepath"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	corev1 "k8s.io/api/core/v1"
)

func AddDeploymentMetadataEnv(container *corev1.Container, dk dynakube.DynaKube) {
	if env.IsIn(container.Env, DynatraceMetadataEnv) {
		return
	}

	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(dk.Status.KubeSystemUUID, deploymentmetadata.GetOneAgentDeploymentType(dk))
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name:  DynatraceMetadataEnv,
			Value: deploymentMetadata.AsString(),
		})
}

func AddNetworkZoneEnv(container *corev1.Container, networkZone string) {
	container.Env = append(container.Env,
		corev1.EnvVar{
			Name:  NetworkZoneEnv,
			Value: networkZone,
		},
	)
}

func AddVersionDetectionEnvs(container *corev1.Container, namespace corev1.Namespace) {
	labelMapping := NewVersionLabelMapping(namespace)
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

func AddPreloadEnv(container *corev1.Container, installPath string) {
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

func concatPreloadPaths(originalPaths, additionalPath string) string {
	if strings.Contains(originalPaths, " ") {
		return originalPaths + " " + additionalPath
	} else {
		return originalPaths + ":" + additionalPath
	}
}
