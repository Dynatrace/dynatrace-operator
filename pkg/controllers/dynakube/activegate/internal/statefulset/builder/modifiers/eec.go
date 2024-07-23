package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = EecModifier{}
var _ volumeMountModifier = EecModifier{}
var _ builder.Modifier = EecModifier{}

const (
	eecVolumeName        = "eec-token"
	eecFileName          = "eec.token"
	eecSecretsMountPoint = "/var/lib/dynatrace/secrets/eec/token/" + eecFileName
	eecGatewayMountPoint = consts.GatewayConfigMountPoint + "/" + eecFileName
)

func NewEecVolumeModifier(dk dynakube.DynaKube) EecModifier {
	return EecModifier{
		dk: dk,
	}
}

type EecModifier struct {
	dk dynakube.DynaKube
}

func (mod EecModifier) Enabled() bool {
	return mod.dk.PrometheusEnabled()
}

func (mod EecModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := container.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	return nil
}

func (mod EecModifier) getVolumes() []corev1.Volume {
	mode := int32(420)

	return []corev1.Volume{
		{
			Name: eecVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  extension.GetSecretName(mod.dk.Name),
					DefaultMode: &mode,
				},
			},
		},
	}
}

func (mod EecModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      eecVolumeName,
			MountPath: eecSecretsMountPoint,
			SubPath:   extension.EecTokenSecretKey,
		},
		{
			ReadOnly:  true,
			Name:      eecVolumeName,
			MountPath: eecGatewayMountPoint,
			SubPath:   extension.EecTokenSecretKey,
		},
	}
}
