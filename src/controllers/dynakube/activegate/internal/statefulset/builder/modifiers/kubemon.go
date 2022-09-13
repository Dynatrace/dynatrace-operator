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
	return mod.dynakube.IsKubernetesMonitoringCapabilityEnabled()
}

func (mod KubernetesMonitoringModifier) Modify(sts *appsv1.StatefulSet) {
	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)
	sts.Spec.Template.Spec.InitContainers = append(sts.Spec.Template.Spec.InitContainers, mod.getInitContainers()...)
}

func (mod KubernetesMonitoringModifier) getInitContainers() []corev1.Container {
	return []corev1.Container{
		{
			Name:            initContainerTemplateName,
			Image:           mod.dynakube.ActiveGateImage(),
			ImagePullPolicy: corev1.PullAlways,
			WorkingDir:      k8scrt2jksWorkingDir,
			Command:         []string{"/bin/bash"},
			Args:            []string{"-c", k8scrt2jksPath},
			VolumeMounts: []corev1.VolumeMount{
				{
					ReadOnly:  false,
					Name:      trustStoreVolume,
					MountPath: activeGateSslPath,
				},
			},
			Resources: mod.capability.Properties().Resources,
		},
	}
}

func (mod KubernetesMonitoringModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: trustStoreVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
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
