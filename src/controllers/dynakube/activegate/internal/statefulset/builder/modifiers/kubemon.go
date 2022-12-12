package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

func NewKubernetesMonitoringModifier(dynakube dynatracev1beta1.DynaKube, capability capability.Capability) KubernetesMonitoringModifier {
	return KubernetesMonitoringModifier{
		dynakube:   dynakube,
		capability: capability,
	}
}

type KubernetesMonitoringModifier struct {
	dynakube   dynatracev1beta1.DynaKube
	capability capability.Capability
}

func (mod KubernetesMonitoringModifier) Enabled() bool {
	return mod.dynakube.IsKubernetesMonitoringActiveGateEnabled()
}

func (mod KubernetesMonitoringModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)
	sts.Spec.Template.Spec.InitContainers = append(sts.Spec.Template.Spec.InitContainers, mod.getInitContainers()...)
	return nil
}

func (mod KubernetesMonitoringModifier) getInitContainers() []corev1.Container {
	readOnlyRootFs := mod.dynakube.FeatureActiveGateReadOnlyFilesystem()
	volumeMounts := []corev1.VolumeMount{
		{
			ReadOnly:  false,
			Name:      trustStoreVolume,
			MountPath: activeGateSslPath,
		},
	}
	volumeMounts = append(volumeMounts, mod.getReadOnlyInitVolumeMounts()...)

	return []corev1.Container{
		{
			Name:            initContainerTemplateName,
			Image:           mod.dynakube.ActiveGateImage(),
			ImagePullPolicy: corev1.PullAlways,
			WorkingDir:      k8scrt2jksWorkingDir,
			Command:         []string{"/bin/bash"},
			Args:            []string{"-c", k8scrt2jksPath},
			VolumeMounts:    volumeMounts,
			Resources:       mod.capability.Properties().Resources,
			SecurityContext: &corev1.SecurityContext{
				ReadOnlyRootFilesystem: &readOnlyRootFs,
			},
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
	readOnlyRootFs := mod.dynakube.FeatureActiveGateReadOnlyFilesystem()
	if readOnlyRootFs {
		return []corev1.Volume{
			{
				Name:         certLoaderWorkDirVolume,
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			},
		}
	}
	return []corev1.Volume{}
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
	readOnlyRootFs := mod.dynakube.FeatureActiveGateReadOnlyFilesystem()
	if readOnlyRootFs {
		return []corev1.VolumeMount{
			{
				ReadOnly:  false,
				Name:      certLoaderWorkDirVolume,
				MountPath: k8scrt2jksWorkingDir,
			},
		}
	}
	return []corev1.VolumeMount{}
}
