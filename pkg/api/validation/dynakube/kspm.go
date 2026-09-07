// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"k8s.io/utils/ptr"
)

const (
	errorTooManyKubernetesMonitoringReplicas = `The Dynakube's specification specifies KSPM, but has more than one replica. Only one ActiveGate or kubernetesMonitoring replica is allowed in combination with KSPM.`
	errorKSPMMissingKubernetesMonitoring     = "The Dynakube's specification specifies KSPM, but requires either a `kubernetesMonitoring` spec with `registration` configured or an ActiveGate with `kubernetes-monitoring` and `automatic-kubernetes-api-monitoring` enabled."
	errorKSPMMissingImage                    = `The Dynakube's specification specifies KSPM, but no image repository/tag is configured.`
	warningKSPMNoHostPaths                   = `The Dynakube's specification specifies KSPM, but no MappedHostPaths are configured.`
	errorKSPMRootHostPath                    = `The Dynakube's specification specifies KSPM, use either '/' or specific path(s) on the MappedHostPath list.`
	errorKSPMRelativeHostPath                = `The Dynakube's specification specifies KSPM, relative path found on the MappedHostPath list. Use absolute paths only. Relative path: %s`
)

func tooManyKubernetesMonitoringReplicas(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	kubemonHasTooManyReplicas := dk.IsKubemonEnabled() && ptr.Deref(dk.KubernetesMonitoring().Replicas, 0) > 1
	activeGateHasTooManyReplicas := dk.ActiveGate().IsKubernetesMonitoringEnabled() && dk.ActiveGate().GetReplicas() > 1

	if dk.KSPM().IsEnabled() && (kubemonHasTooManyReplicas || activeGateHasTooManyReplicas) {
		return errorTooManyKubernetesMonitoringReplicas
	}

	return ""
}

func kspmWithoutKubernetesMonitoringRegistration(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && !dk.IsKubernetesMonitoringRegistrationEnabled() {
		return errorKSPMMissingKubernetesMonitoring
	}

	return ""
}

func missingKSPMImage(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.KSPM().IsEnabled() {
		return ""
	}

	if !dk.KSPM().ImageRef.HasImage() {
		return errorKSPMMissingImage
	}

	return ""
}

func noMappedHostPaths(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.KSPM().IsEnabled() {
		return ""
	}

	if len(dk.KSPM().GetUniqueMappedHostPaths()) == 0 {
		return warningKSPMNoHostPaths
	}

	return ""
}

func mappedHostPathsWithRootPath(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.KSPM().IsEnabled() {
		return ""
	}

	mappedHostPaths := dk.KSPM().GetUniqueMappedHostPaths()

	if slices.Index(mappedHostPaths, "/") != -1 && len(mappedHostPaths) > 1 {
		return errorKSPMRootHostPath
	}

	return ""
}

func relativeMappedHostPaths(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
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
