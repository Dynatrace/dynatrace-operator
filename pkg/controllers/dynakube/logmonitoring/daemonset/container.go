package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/address"
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultImageRepo = "registry.lab.dynatrace.org/oneagent/dynatrace-logmodule-amd64" // TODO: finalize
	defaultImageTag  = "latest"

	containerName       = "main"
	runAs         int64 = 65532

	initContainerName = "init-volume"
	bootstrapCommand  = "/opt/dynatrace/oneagent/agent/lib64/bootstrap"
)

var (
	neededCapabilities = []corev1.Capability{
		"DAC_READ_SEARCH",
	}

	neededInitCapabilities = []corev1.Capability{
		"CHOWN",
	}
)

func getContainer(dk dynakube.DynaKube) corev1.Container {
	securityContext := getBaseSecurityContext(dk)
	securityContext.Capabilities.Add = neededCapabilities

	container := corev1.Container{
		Name:            containerName,
		Image:           dk.LogMonitoring().ImageRef.StringWithDefaults(defaultImageRepo, defaultImageTag),
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts:    getVolumeMounts(),
		Env:             getEnvs(),
		SecurityContext: &securityContext,
	}

	return container
}

func getInitContainer(dk dynakube.DynaKube) corev1.Container {
	securityContext := getBaseSecurityContext(dk)
	securityContext.Capabilities.Add = neededInitCapabilities

	container := corev1.Container{
		Name:            initContainerName,
		Image:           dk.LogMonitoring().ImageRef.StringWithDefaults(defaultImageRepo, defaultImageTag),
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts:    getDTVolumeMounts(),
		Command:         []string{bootstrapCommand},
		Env:             getInitEnvs(dk),
		Args:            getInitArgs(dk),
		SecurityContext: &securityContext,
	}

	return container
}

func getBaseSecurityContext(dk dynakube.DynaKube) corev1.SecurityContext {
	securityContext := corev1.SecurityContext{
		Privileged:               address.Of(false),
		ReadOnlyRootFilesystem:   address.Of(true),
		AllowPrivilegeEscalation: address.Of(false),
		RunAsUser:                address.Of(runAs),
		RunAsGroup:               address.Of(runAs),
		RunAsNonRoot:             address.Of(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}

	seccomp := dk.LogMonitoring().SecCompProfile
	if seccomp != "" {
		securityContext.SeccompProfile = &corev1.SeccompProfile{LocalhostProfile: &seccomp}
	}

	return securityContext
}
