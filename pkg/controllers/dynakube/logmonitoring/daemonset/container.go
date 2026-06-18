package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8ssecuritycontext"
	corev1 "k8s.io/api/core/v1"
)

const (
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

func getContainer(dk dynakube.DynaKube, tenantUUID string, imageURI string) corev1.Container {
	securityContext := getBaseSecurityContext(dk)
	securityContext.Capabilities.Add = neededCapabilities

	if dk.Spec.Templates.LogMonitoring != nil {
		securityContext.AppArmorProfile = k8ssecuritycontext.GetAppArmorProfile(dk.Spec.Templates.LogMonitoring.Annotations, containerName)
	}

	if imageURI == "" {
		imageURI = dk.LogMonitoring().Template().ImageRef.String()
	}

	container := corev1.Container{
		Name:            containerName,
		Image:           imageURI,
		ImagePullPolicy: dk.LogMonitoring().Template().ImageRef.GetPullPolicy(),
		VolumeMounts:    getVolumeMounts(tenantUUID),
		Env:             getEnvs(),
		Resources:       dk.LogMonitoring().Template().Resources,
		SecurityContext: securityContext,
	}

	return container
}

func getInitContainer(dk dynakube.DynaKube, tenantUUID string, imageURI string) corev1.Container {
	securityContext := getBaseSecurityContext(dk)
	securityContext.Capabilities.Add = neededInitCapabilities

	if dk.Spec.Templates.LogMonitoring != nil {
		securityContext.AppArmorProfile = k8ssecuritycontext.GetAppArmorProfile(dk.Spec.Templates.LogMonitoring.Annotations, initContainerName)
	}

	image := imageURI
	if imageURI == "" {
		image = dk.LogMonitoring().Template().ImageRef.String()
	}

	container := corev1.Container{
		Name:            initContainerName,
		Image:           image,
		ImagePullPolicy: dk.LogMonitoring().Template().ImageRef.GetPullPolicy(),
		VolumeMounts:    []corev1.VolumeMount{getDTVolumeMounts(tenantUUID)},
		Command:         []string{bootstrapCommand},
		Env:             getInitEnvs(dk),
		Args:            getInitArgs(dk),
		Resources:       dk.LogMonitoring().Template().Resources,
		SecurityContext: securityContext,
	}

	return container
}

func getBaseSecurityContext(dk dynakube.DynaKube) *corev1.SecurityContext {
	securityContext := &corev1.SecurityContext{
		Privileged:               new(false),
		ReadOnlyRootFilesystem:   new(true),
		AllowPrivilegeEscalation: new(false),
		RunAsUser:                new(runAs),
		RunAsGroup:               new(runAs),
		RunAsNonRoot:             new(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}

	seccomp := dk.LogMonitoring().Template().SecCompProfile
	if seccomp != "" {
		securityContext.SeccompProfile = &corev1.SeccompProfile{LocalhostProfile: &seccomp}
	}

	if dk.OneAgent().IsPrivilegedNeeded() {
		securityContext.Privileged = new(true)
		securityContext.AllowPrivilegeEscalation = new(true)
	}

	return securityContext
}
