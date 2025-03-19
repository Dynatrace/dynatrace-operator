package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	corev1 "k8s.io/api/core/v1"
)

func addInstallerInitEnvs(initContainer *corev1.Container, installer installerInfo) {
	initContainer.Env = append(initContainer.Env,
		corev1.EnvVar{Name: consts.AgentInstallerFlavorEnv, Value: installer.flavor},
		corev1.EnvVar{Name: consts.AgentInstallerTechEnv, Value: installer.technologies},
		corev1.EnvVar{Name: consts.AgentInstallPathEnv, Value: installer.installPath},
		corev1.EnvVar{Name: consts.AgentInstallerUrlEnv, Value: installer.installerURL},
		corev1.EnvVar{Name: consts.AgentInstallerVersionEnv, Value: installer.version},
		corev1.EnvVar{Name: consts.AgentInjectedEnv, Value: "true"},
	)
}
