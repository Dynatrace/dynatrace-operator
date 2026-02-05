package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
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

func getContainer(dk dynakube.DynaKube, tenantUUID string) corev1.Container {
	securityContext := getBaseSecurityContext(dk)
	securityContext.Capabilities.Add = neededCapabilities

	container := corev1.Container{
		Name:            containerName,
		Image:           dk.LogMonitoring().Template().ImageRef.StringWithDefaults(defaultImageRepo, defaultImageTag),
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts:    getVolumeMounts(tenantUUID),
		Env:             getEnvs(),
		Resources:       dk.LogMonitoring().Template().Resources,
		SecurityContext: &securityContext,
	}

	return container
}

func getInitContainer(dk dynakube.DynaKube, tenantUUID string) corev1.Container {
	securityContext := getBaseSecurityContext(dk)
	securityContext.Capabilities.Add = neededInitCapabilities

	container := corev1.Container{
		Name:            initContainerName,
		Image:           dk.LogMonitoring().Template().ImageRef.StringWithDefaults(defaultImageRepo, defaultImageTag),
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts:    []corev1.VolumeMount{getDTVolumeMounts(tenantUUID)},
		Command:         []string{bootstrapCommand},
		Env:             getInitEnvs(dk),
		Args:            getInitArgs(dk),
		Resources:       dk.LogMonitoring().Template().Resources,
		SecurityContext: &securityContext,
	}

	return container
}

func getBaseSecurityContext(dk dynakube.DynaKube) corev1.SecurityContext {
	securityContext := corev1.SecurityContext{
		Privileged:               ptr.To(false),
		ReadOnlyRootFilesystem:   ptr.To(true),
		AllowPrivilegeEscalation: ptr.To(false),
		RunAsUser:                ptr.To(runAs),
		RunAsGroup:               ptr.To(runAs),
		RunAsNonRoot:             ptr.To(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}

	seccomp := dk.LogMonitoring().Template().SecCompProfile
	if seccomp != "" {
		securityContext.SeccompProfile = &corev1.SeccompProfile{LocalhostProfile: &seccomp}
	}

	if dk.OneAgent().IsPrivilegedNeeded() {
		securityContext.Privileged = ptr.To(true)
		securityContext.AllowPrivilegeEscalation = ptr.To(true)
	}

	return securityContext
}
