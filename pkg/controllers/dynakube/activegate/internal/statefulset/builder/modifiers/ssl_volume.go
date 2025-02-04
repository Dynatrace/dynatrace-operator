package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = SSLVolumeModifier{}
var _ volumeMountModifier = SSLVolumeModifier{}
var _ builder.Modifier = SSLVolumeModifier{}

func NewSSLVolumeModifier(dk dynakube.DynaKube) SSLVolumeModifier {
	return SSLVolumeModifier{
		dk: dk,
	}
}

type SSLVolumeModifier struct {
	dk dynakube.DynaKube
}

func (mod SSLVolumeModifier) Enabled() bool {
	return mod.dk.ActiveGate().HasCaCert() || mod.dk.Spec.TrustedCAs != ""
}

func (mod SSLVolumeModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := container.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	return nil
}

func (mod SSLVolumeModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: consts.GatewaySslVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func (mod SSLVolumeModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  false,
			Name:      consts.GatewaySslVolumeName,
			MountPath: consts.GatewaySslMountPoint,
		},
	}
}
