package modifiers

import (
	"path"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8scontainer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = DeploymentPropertiesModifier{}
var _ volumeMountModifier = DeploymentPropertiesModifier{}
var _ builder.Modifier = DeploymentPropertiesModifier{}

func NewDeploymentPropertiesModifier(dk dynakube.DynaKube) DeploymentPropertiesModifier {
	return DeploymentPropertiesModifier{
		dk: dk,
	}
}

type DeploymentPropertiesModifier struct {
	dk dynakube.DynaKube
}

func (mod DeploymentPropertiesModifier) Enabled() bool {
	return true
}

func (mod DeploymentPropertiesModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := k8scontainer.FindInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	return nil
}

func (mod DeploymentPropertiesModifier) getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: consts.DeploymentPropertiesVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: mod.dk.ActiveGate().GetDeploymentPropertiesConfigMapName(),
					},
					Items: []corev1.KeyToPath{
						{
							Key:  consts.DeploymentPropertiesFileName,
							Path: consts.DeploymentPropertiesFileName,
						},
					},
				},
			},
		},
	}
}

func (mod DeploymentPropertiesModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      consts.DeploymentPropertiesVolumeName,
			MountPath: getMountPath(),
			SubPath:   consts.DeploymentPropertiesFileName,
		},
	}
}

func getMountPath() string {
	return path.Join(consts.DeploymentPropertiesBasePath, consts.DeploymentPropertiesFileName)
}
