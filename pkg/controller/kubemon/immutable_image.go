package kubemon

import (
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/dtpullsecret"
	corev1 "k8s.io/api/core/v1"
)

func buildImage(instance *v1alpha1.DynaKube) string {
	if instance.Spec.KubernetesMonitoringSpec.Image == "" {
		return buildActiveGateImage(instance)
	}

	return instance.Spec.KubernetesMonitoringSpec.Image
}

func buildPullSecret(instance *v1alpha1.DynaKube) corev1.LocalObjectReference {
	return corev1.LocalObjectReference{
		Name: buildPullSecretName(instance),
	}
}

func buildPullSecretName(instance *v1alpha1.DynaKube) string {
	name := instance.Name + dtpullsecret.PullSecretSuffix
	if instance.Spec.CustomPullSecret != "" {
		name = instance.Spec.CustomPullSecret
	}
	return name
}

func buildActiveGateImage(instance *v1alpha1.DynaKube) string {
	registry := buildImageRegistryFromAPIURL(instance.Spec.APIURL)
	fullImageName := appendPath(registry)
	return appendActiveGateVersion(instance, fullImageName)
}

func appendActiveGateVersion(instance *v1alpha1.DynaKube, fullImageName string) string {
	version := instance.Spec.KubernetesMonitoringSpec.ActiveGateVersion
	if version != "" {
		return fmt.Sprintf("%s:%s", fullImageName, version)
	}
	return fullImageName
}

func appendPath(registry string) string {
	return fmt.Sprintf("%s/linux/activegate", registry)
}

func buildImageRegistryFromAPIURL(apiURL string) string {
	r := strings.TrimPrefix(apiURL, "https://")
	r = strings.TrimSuffix(r, "/api")
	return r
}
