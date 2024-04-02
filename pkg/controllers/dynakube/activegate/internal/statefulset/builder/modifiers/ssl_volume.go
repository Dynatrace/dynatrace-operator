package modifiers

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = SSLVolumeModifier{}
var _ volumeMountModifier = SSLVolumeModifier{}
var _ builder.Modifier = SSLVolumeModifier{}

func NewSSLVolumeModifier(dynakube dynatracev1beta1.DynaKube) SSLVolumeModifier {
	return SSLVolumeModifier{
		dynakube: dynakube,
	}
}

type SSLVolumeModifier struct {
	dynakube dynatracev1beta1.DynaKube
}

func (mod SSLVolumeModifier) Enabled() bool {
	return mod.dynakube.HasActiveGateCaCert() || mod.dynakube.Spec.TrustedCAs != ""
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
