package modifiers

import (
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

var _ volumeModifier = KubernetesMonitoringModifier{}
var _ volumeMountModifier = KubernetesMonitoringModifier{}
var _ initContainerModifier = KubernetesMonitoringModifier{}
var _ builder.Modifier = KubernetesMonitoringModifier{}

const (
	trustStoreVolume          = "truststore-volume"
	activeGateCacertsPath     = "/opt/dynatrace/gateway/jre/lib/security/cacerts"
	k8sCertificateFile        = "k8s-local.jks"
	k8scrt2jksPath            = "/opt/dynatrace/gateway/k8scrt2jks.sh"
	activeGateSslPath         = "/var/lib/dynatrace/gateway/ssl"
	k8scrt2jksWorkingDir      = "/var/lib/dynatrace/gateway"
	initContainerTemplateName = "certificate-loader"

	certLoaderWorkDirVolume = "cert-tmp"
)

func NewKubernetesMonitoringModifier(dk dynakube.DynaKube, capability capability.Capability) KubernetesMonitoringModifier {
	return KubernetesMonitoringModifier{
		dk:         dk,
		capability: capability,
	}
}

type KubernetesMonitoringModifier struct {
	capability capability.Capability
	dk         dynakube.DynaKube
}

func (mod KubernetesMonitoringModifier) Enabled() bool {
	return mod.dk.ActiveGate().IsKubernetesMonitoringEnabled()
}

func (mod KubernetesMonitoringModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := k8scontainer.FindInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)
	sts.Spec.Template.Spec.InitContainers = append(sts.Spec.Template.Spec.InitContainers, mod.getInitContainers()...)
	sts.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(true)

	return nil
}

func (mod KubernetesMonitoringModifier) getInitContainers() []corev1.Container {
	volumeMounts := slices.Concat([]corev1.VolumeMount{
		{
			ReadOnly:  false,
			Name:      trustStoreVolume,
			MountPath: activeGateSslPath,
		},
	}, mod.getReadOnlyInitVolumeMounts())

	return []corev1.Container{
		{
			Name:            initContainerTemplateName,
			Image:           mod.dk.ActiveGate().GetImage(),
			ImagePullPolicy: mod.dk.ActiveGate().GetPullPolicy(),
			WorkingDir:      k8scrt2jksWorkingDir,
			Command:         []string{"/bin/bash"},
			Args:            []string{"-c", k8scrt2jksPath},
			VolumeMounts:    volumeMounts,
			Resources:       mod.capability.Properties().Resources,
			SecurityContext: GetSecurityContext(true),
		},
	}
}

func (mod KubernetesMonitoringModifier) getVolumes() []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: trustStoreVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	return append(volumes, mod.getReadOnlyInitVolumes()...)
}

func (mod KubernetesMonitoringModifier) getReadOnlyInitVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name:         certLoaderWorkDirVolume,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
	}
}

func (mod KubernetesMonitoringModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      trustStoreVolume,
			MountPath: activeGateCacertsPath,
			SubPath:   k8sCertificateFile,
		},
	}
}

func (mod KubernetesMonitoringModifier) getReadOnlyInitVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  false,
			Name:      certLoaderWorkDirVolume,
			MountPath: k8scrt2jksWorkingDir,
		},
	}
}

func GetSecurityContext(readOnlyRootFileSystem bool) *corev1.SecurityContext {
	securityContext := corev1.SecurityContext{
		Privileged:               ptr.To(false),
		AllowPrivilegeEscalation: ptr.To(false),
		RunAsNonRoot:             ptr.To(true),
		RunAsUser:                ptr.To(consts.DockerImageUser),
		RunAsGroup:               ptr.To(consts.DockerImageGroup),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		ReadOnlyRootFilesystem: ptr.To(readOnlyRootFileSystem),
	}

	return &securityContext
}
