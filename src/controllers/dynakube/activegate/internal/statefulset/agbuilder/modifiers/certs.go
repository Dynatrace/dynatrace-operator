package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = CertificatesModifier{}
var _ volumeMountModifier = CertificatesModifier{}
var _ agbuilder.Modifier = CertificatesModifier{}

func NewCertificatesModifier(dynakube dynatracev1beta1.DynaKube) CertificatesModifier {
	return CertificatesModifier{
		dynakube: dynakube,
	}
}

type CertificatesModifier struct {
	dynakube dynatracev1beta1.DynaKube
}

func (mod CertificatesModifier) Enabled() bool {
	return mod.dynakube.HasActiveGateCaCert()
}

func (mod CertificatesModifier) Modify(sts *appsv1.StatefulSet) {
	baseContainer := kubeobjects.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)
}

func (mod CertificatesModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: consts.GatewaySslVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func (mod CertificatesModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  false,
			Name:      consts.GatewaySslVolumeName,
			MountPath: consts.GatewaySslMountPoint,
		},
	}
}
