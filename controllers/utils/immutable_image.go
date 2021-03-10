package utils

import (
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	corev1 "k8s.io/api/core/v1"
)

// SetUseImmutableImageStatus updates the status' UseImmutableImage field to indicate whether the Operator should use
// immutable images or not.
func SetUseImmutableImageStatus(instance *dynatracev1alpha1.DynaKube, fs *dynatracev1alpha1.FullStackSpec) bool {
	if fs.UseImmutableImage == instance.Status.OneAgent.UseImmutableImage {
		return false
	}

	instance.Status.OneAgent.UseImmutableImage = fs.UseImmutableImage
	return true
}

func BuildPullSecret(instance *dynatracev1alpha1.DynaKube) corev1.LocalObjectReference {
	return corev1.LocalObjectReference{
		Name: buildPullSecretName(instance),
	}
}

func buildPullSecretName(instance *dynatracev1alpha1.DynaKube) string {
	name := instance.Name + dtpullsecret.PullSecretSuffix
	if instance.Spec.CustomPullSecret != "" {
		name = instance.Spec.CustomPullSecret
	}
	return name
}
