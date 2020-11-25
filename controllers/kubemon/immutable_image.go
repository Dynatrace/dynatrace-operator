package kubemon

import (
	"fmt"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	corev1 "k8s.io/api/core/v1"
)

func buildImage(instance *dynatracev1alpha1.DynaKube) string {
	if instance.Spec.KubernetesMonitoringSpec.Image == "" {
		return buildActiveGateImage(instance)
	}

	return instance.Spec.KubernetesMonitoringSpec.Image
}

func buildPullSecret(instance *dynatracev1alpha1.DynaKube) corev1.LocalObjectReference {
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

func buildActiveGateImage(instance *dynatracev1alpha1.DynaKube) string {
	registry := buildImageRegistryFromAPIURL(instance.Spec.APIURL)
	fullImageName := fmt.Sprintf("%s/linux/activegate", registry)
	version := instance.Spec.KubernetesMonitoringSpec.ActiveGateVersion
	if version != "" {
		return fmt.Sprintf("%s:%s", fullImageName, version)
	}
	return fullImageName
}

func buildImageRegistryFromAPIURL(apiURL string) string {
	r := strings.TrimPrefix(apiURL, "https://")
	r = strings.TrimPrefix(r, "http://")
	r = strings.TrimSuffix(r, "/api")
	return r
}
