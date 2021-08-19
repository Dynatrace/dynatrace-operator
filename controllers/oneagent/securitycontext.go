package oneagent

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	hostRootMount = "host-root"

	oneagentInstallationMountName = "oneagent-installation"
	oneagentInstallationMountPath = "/mnt/volume_storage_mount"

	oneagentReadOnlyMode = "ONEAGENT_READ_ONLY_MODE"
	enableVolumeStorage  = "ONEAGENT_ENABLE_VOLUME_STORAGE"
)

func prepareSecurityContext(unprivileged bool) *corev1.SecurityContext {
	var secCtx *corev1.SecurityContext

	if unprivileged {
		secCtx = getUnprivilegedSecurityContext()
	} else {
		secCtx = getPrivilegedSecurityContext()
	}

	return secCtx
}

func getPrivilegedSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Privileged: boolPointer(true),
	}
}

func getUnprivilegedSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
			Add: []corev1.Capability{
				"CHOWN",
				"DAC_OVERRIDE",
				"DAC_READ_SEARCH",
				"FOWNER",
				"FSETID",
				"KILL",
				"NET_ADMIN",
				"NET_RAW",
				"SETFCAP",
				"SETGID",
				"SETUID",
				"SYS_ADMIN",
				"SYS_CHROOT",
				"SYS_PTRACE",
				"SYS_RESOURCE",
			},
		},
	}
}

func boolPointer(value bool) *bool {
	return &value
}

func int64Pointer(value int64) *int64 {
	return &value
}
