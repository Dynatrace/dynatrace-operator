package validation

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	errorTooManyAGReplicas    = `The Dynakube's specification specifies KSPM, but has more than one ActiveGate replica. Only one ActiveGate replica is allowed in combination with KSPM.`
	errorKSPMMissingKubemon   = "The Dynakube's specification specifies KSPM, but requires an ActiveGate with `kubernetes-monitoring` enabled and the `automatic-kubernetes-api-monitoring` feature flag set to `true`."
	errorKSPMMissingImage     = `The Dynakube's specification specifies KSPM, but no image repository/tag is configured.`
	warningKSPMNoHostPaths    = `The Dynakube's specification specifies KSPM, but no MappedHostPaths are configured.`
	errorKSPMRootHostPath     = `The Dynakube's specification specifies KSPM, use either '/' or specific path(s) on the MappedHostPath list.`
	errorKSPMRelativeHostPath = `The Dynakube's specification specifies KSPM, relative path found on the MappedHostPath list. Use absolute paths only. Relative path: %s`
)

func tooManyAGReplicas(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && dk.ActiveGate().GetReplicas() > 1 {
		return errorTooManyAGReplicas
	}

	return ""
}

func kspmWithoutK8SMonitoring(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && (!dk.ActiveGate().IsKubernetesMonitoringEnabled() || !dk.FF().IsAutomaticK8sAPIMonitoring()) {
		return errorKSPMMissingKubemon
	}

	return ""
}

func missingKSPMImage(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	if !dk.KSPM().IsEnabled() {
		return ""
	}

	if dk.KSPM().ImageRef.Repository == "" || dk.KSPM().ImageRef.Tag == "" {
		return errorKSPMMissingImage
	}

	return ""
}

func noMappedHostPaths(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	if !dk.KSPM().IsEnabled() {
		return ""
	}

	if len(dk.KSPM().GetUniqueMappedHostPaths()) == 0 {
		return warningKSPMNoHostPaths
	}

	return ""
}

func mappedHostPathsWithRootPath(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	if !dk.KSPM().IsEnabled() {
		return ""
	}

	mappedHostPaths := dk.KSPM().GetUniqueMappedHostPaths()

	if slices.Index(mappedHostPaths, "/") != -1 && len(mappedHostPaths) > 1 {
		return errorKSPMRootHostPath
	}

	return ""
}

func relativeMappedHostPaths(_ context.Context, _ *validatorClient, dk *dynakube.DynaKube) string {
	if !dk.KSPM().IsEnabled() {
		return ""
	}

	mappedHostPaths := dk.KSPM().GetUniqueMappedHostPaths()

	for _, path := range mappedHostPaths {
		if !filepath.IsAbs(path) {
			return fmt.Sprintf(errorKSPMRelativeHostPath, path)
		}
	}

	return ""
}
