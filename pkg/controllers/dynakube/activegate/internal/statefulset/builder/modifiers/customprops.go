package modifiers

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/container"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ volumeModifier = CustomPropertiesModifier{}
var _ volumeMountModifier = CustomPropertiesModifier{}
var _ builder.Modifier = CustomPropertiesModifier{}

func NewCustomPropertiesModifier(dynakube dynatracev1beta1.DynaKube, capability capability.Capability) CustomPropertiesModifier {
	return CustomPropertiesModifier{
		dynakube:   dynakube,
		capability: capability,
	}
}

type CustomPropertiesModifier struct {
	capability capability.Capability
	dynakube   dynatracev1beta1.DynaKube
}

func (mod CustomPropertiesModifier) Enabled() bool {
	return mod.hasCustomProperties()
}

func (mod CustomPropertiesModifier) Modify(sts *appsv1.StatefulSet) error {
	baseContainer := container.FindContainerInPodSpec(&sts.Spec.Template.Spec, consts.ActiveGateContainerName)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, mod.getVolumes()...)
	baseContainer.VolumeMounts = append(baseContainer.VolumeMounts, mod.getVolumeMounts()...)

	return nil
}

func (mod CustomPropertiesModifier) getVolumes() []corev1.Volume {
	valueFrom := mod.determineCustomPropertiesSource()
	volumes := []corev1.Volume{
		{
			Name: customproperties.VolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: valueFrom,
					Items: []corev1.KeyToPath{
						{Key: customproperties.DataKey, Path: customproperties.DataPath},
					}}},
		},
	}

	return volumes
}

func (mod CustomPropertiesModifier) getVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			ReadOnly:  true,
			Name:      customproperties.VolumeName,
			MountPath: customproperties.MountPath,
			SubPath:   customproperties.DataPath,
		},
	}
}

func (mod CustomPropertiesModifier) hasCustomProperties() bool {
	customProperties := mod.capability.Properties().CustomProperties

	return customProperties != nil &&
		(customProperties.Value != "" ||
			customProperties.ValueFrom != "")
}

func (mod CustomPropertiesModifier) determineCustomPropertiesSource() string {
	if mod.capability.Properties().CustomProperties.ValueFrom == "" {
		return fmt.Sprintf("%s-%s-%s", mod.dynakube.Name, mod.dynakube.ActiveGateServiceAccountOwner(), customproperties.Suffix)
	}

	return mod.capability.Properties().CustomProperties.ValueFrom
}
