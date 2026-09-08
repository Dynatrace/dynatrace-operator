// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	errorMutualExclusiveKubernetesMonitoring      = "The DynaKube configures Kubernetes Monitoring in both `spec.kubernetesMonitoring` and `spec.activeGate.capabilities`. Configure Kubernetes Monitoring in only one of these locations."
	errorMutualExclusiveKubernetesMonitoringValue = "The DynaKube's `spec.kubernetesMonitoring.customProperties` cannot set both `value` and `valueFrom`. Set only one of these fields."
)

func mutualExclusiveKubernetesMonitoring(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.IsKubemonEnabled() && dk.Spec.ActiveGate.IsKubernetesMonitoringEnabled() {
		return errorMutualExclusiveKubernetesMonitoring
	}

	return ""
}

func kubemonMutualExclusiveCustomPropertiesValue(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.IsKubemonEnabled() {
		customProperties := dk.KubernetesMonitoring().CustomProperties
		if customProperties != nil && customProperties.Value != "" && customProperties.ValueFrom != "" {
			return errorMutualExclusiveKubernetesMonitoringValue
		}
	}

	return ""
}
