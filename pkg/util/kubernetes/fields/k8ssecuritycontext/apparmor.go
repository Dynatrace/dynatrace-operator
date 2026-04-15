package k8ssecuritycontext

import (
	"maps"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/version"
	corev1 "k8s.io/api/core/v1"
)

func isAppArmorRewriteEnabled() bool {
	const minAppArmorRewriteKubernetesVersion = 31

	return version.GetMinorVersion() >= minAppArmorRewriteKubernetesVersion
}

// GetAppArmorProfile builds the AppArmorProfile from the annotations and container name.
// Returns nil if no AppArmor annotation key for the container is present or the operator runs on Kubernetes 1.30 or lower.
//
// This function uses the version cache, so tests need to make sure to call [version.DisableCacheForTest].
func GetAppArmorProfile(annotations map[string]string, containerName string) *corev1.AppArmorProfile {
	var profile *corev1.AppArmorProfile
	if isAppArmorRewriteEnabled() {
		profile = getProfileFromPodAnnotations(annotations, containerName)
	}

	return profile
}

// RemoveAppArmorAnnotation returns a copy of the annotation without AppArmor annotations for the specified containers.
// If no AppArmor annotations are found or the operator runs on Kubernetes 1.30 or lower, returns the input without modification.
//
// This function uses the version cache, so tests need to make sure to call [version.DisableCacheForTest].
func RemoveAppArmorAnnotation(annotations map[string]string, containerNames ...string) map[string]string {
	if isAppArmorRewriteEnabled() {
		var modified map[string]string

		for _, containerName := range containerNames {
			key := corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + containerName
			if _, ok := annotations[key]; ok {
				if modified == nil {
					modified = maps.Clone(annotations)
				}

				delete(modified, key)
			}
		}

		if modified != nil {
			return modified
		}
	}

	return annotations
}

// getProfileFromPodAnnotations gets the AppArmor profile to use with container from
// (deprecated) pod annotations.
//
// Source: https://github.com/kubernetes/kubernetes/blob/v1.35.3/pkg/security/apparmor/helpers.go#L74
func getProfileFromPodAnnotations(annotations map[string]string, containerName string) *corev1.AppArmorProfile {
	val, ok := annotations[corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix+containerName]
	if !ok {
		return nil
	}

	switch {
	case val == corev1.DeprecatedAppArmorBetaProfileRuntimeDefault:
		return &corev1.AppArmorProfile{Type: corev1.AppArmorProfileTypeRuntimeDefault}

	case val == corev1.DeprecatedAppArmorBetaProfileNameUnconfined:
		return &corev1.AppArmorProfile{Type: corev1.AppArmorProfileTypeUnconfined}

	case strings.HasPrefix(val, corev1.DeprecatedAppArmorBetaProfileNamePrefix):
		// Note: an invalid empty localhost profile will be rejected by kubelet admission.
		profileName := strings.TrimPrefix(val, corev1.DeprecatedAppArmorBetaProfileNamePrefix)

		return &corev1.AppArmorProfile{
			Type:             corev1.AppArmorProfileTypeLocalhost,
			LocalhostProfile: &profileName,
		}

	default:
		// Invalid annotation.
		return nil
	}
}
