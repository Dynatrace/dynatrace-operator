// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	warningExtensionsWithoutK8SMonitoring = "The DynaKube configures extensions, which require Kubernetes Monitoring with cluster registration. Configure `spec.kubernetesMonitoring` and add its `registration` section, or configure an ActiveGate with the `kubernetes-monitoring` capability and enable the `automatic-kubernetes-api-monitoring` feature flag."
)

func extensionsWithoutK8SMonitoring(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.Extensions().IsAnyEnabled() && !dk.IsKubernetesMonitoringRegistrationEnabled() {
		return warningExtensionsWithoutK8SMonitoring
	}

	return ""
}
