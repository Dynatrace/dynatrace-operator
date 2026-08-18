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
	errorTooManyKubernetesMonitoringReplicas  = `The Dynakube's specification specifies KSPM, but has more than one replica. Only one ActiveGate or kubernetesMonitoring replica is allowed in combination with KSPM.`
	errorKSPMMissingKubernetesMonitoring      = "The Dynakube's specification specifies KSPM, but requires either a `kubernetesMonitoring` spec or an ActiveGate with `kubernetes-monitoring` enabled."
	errorKSPMMissingAutomaticK8sAPIMonitoring = "The Dynakube's specification specifies KSPM with an ActiveGate `kubernetes-monitoring` capability, but the `automatic-kubernetes-api-monitoring` feature flag is not set to `true`. It is required for KSPM to function correctly."
	errorKSPMMissingRegistration              = "The Dynakube's specification specifies KSPM together with `kubernetesMonitoring`, but `kubernetesMonitoring.registration` is not set. It is required to register the Kubernetes cluster in Dynatrace, which KSPM depends on."
	errorKSPMMissingImage                     = `The Dynakube's specification specifies KSPM, but no image repository/tag is configured.`
	warningKSPMNoHostPaths                    = `The Dynakube's specification specifies KSPM, but no MappedHostPaths are configured.`
	errorKSPMRootHostPath                     = `The Dynakube's specification specifies KSPM, use either '/' or specific path(s) on the MappedHostPath list.`
	errorKSPMRelativeHostPath                 = `The Dynakube's specification specifies KSPM, relative path found on the MappedHostPath list. Use absolute paths only. Relative path: %s`
)

func tooManyKubernetesMonitoringReplicas(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	kubemonHasTooManyReplicas := dk.IsKubemonEnabled() && ptr.Deref(dk.KubernetesMonitoring().Replicas, 0) > 1
	activeGateHasTooManyReplicas := dk.ActiveGate().IsKubernetesMonitoringEnabled() && dk.ActiveGate().GetReplicas() > 1

	if dk.KSPM().IsEnabled() && (kubemonHasTooManyReplicas || activeGateHasTooManyReplicas) {
		return errorTooManyKubernetesMonitoringReplicas
	}

	return ""
}

func kspmWithoutK8sMonitoring(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && !dk.IsKubernetesMonitoringEnabled() {
		return errorKSPMMissingKubernetesMonitoring
	}

	return ""
}

func kspmWithoutAutomaticK8sAPIMonitoring(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && dk.ActiveGate().IsKubernetesMonitoringEnabled() && !dk.FF().IsAutomaticK8sAPIMonitoring() {
		return errorKSPMMissingAutomaticK8sAPIMonitoring
	}

	return ""
}

func kspmWithoutRegistration(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.KSPM().IsEnabled() && dk.IsKubemonEnabled() && !dk.KubernetesMonitoring().IsRegistrationEnabled() {
		return errorKSPMMissingRegistration
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
