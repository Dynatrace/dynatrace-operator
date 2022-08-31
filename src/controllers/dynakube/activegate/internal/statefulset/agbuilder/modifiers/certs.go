package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	agbuilderTypes "github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder/internal/types"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func NewCertificatesModifier(dynakube dynatracev1beta1.DynaKube) agbuilderTypes.Modifier {
	return AuthTokenModifier{
		dynakube: dynakube,
	}
}

type CertificatesModifier struct {
	dynakube dynatracev1beta1.DynaKube
}

func (mod CertificatesModifier) Modify(sts *appsv1.StatefulSet) {
	if !mod.dynakube.HasActiveGateCaCert() {
		return
	}

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
