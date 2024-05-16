package modifiers

import (
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = EecModifier{}
var _ volumeMountModifier = EecModifier{}
var _ builder.Modifier = EecModifier{}

const (
	eecVolumeName = "eec-token"
	eecDir        = "/var/lib/dynatrace/secrets/eec"
	eecFile       = "eec.token"
)

func NewEecVolumeModifier(dynakube dynatracev1beta2.DynaKube) EecModifier {
	return EecModifier{
		dynakube: dynakube,
	}
}

type EecModifier struct {
	dynakube dynatracev1beta2.DynaKube
}

func (mod EecModifier) Enabled() bool {
	return true
}

func (mod EecModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := container.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	baseContainer.Command = []string{"/bin/sh"}
	baseContainer.Args = []string{
		"-c",
		"echo abc ; cp -v " + eecDir + "/" + eecFile + " /var/lib/dynatrace/gateway/config/ ; /opt/dynatrace/gateway/entrypoint.sh",
	}

	return nil
}

func (mod EecModifier) getVolumes() []corev1.Volume {
	mode := int32(420)
	return []corev1.Volume{
		{
			Name: eecVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  "eec-ag-token",
					DefaultMode: &mode,
					Items: []corev1.KeyToPath{
						{
							Key:  "eec-token",
							Path: eecFile,
							Mode: &mode,
						},
					},
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
			MountPath: eecDir,
		},
	}
}
