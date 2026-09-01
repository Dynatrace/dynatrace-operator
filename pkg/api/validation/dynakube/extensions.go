// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
)

const (
	warningExtensionsWithoutK8SMonitoringNew = "The DynaKube configures extensions, which require Kubernetes Monitoring with cluster registration. Configure `spec.kubernetesMonitoring` and add its `registration` section, or configure an ActiveGate with the `kubernetes-monitoring` capability and enable the `automatic-kubernetes-api-monitoring` feature flag."

	warningExtensionsWithoutK8SMonitoringOld = "The Dynakube is configured with extensions without an ActiveGate with `kubernetes-monitoring` enabled or the `automatic-kubernetes-api-monitoring` feature flag. You need to ensure that Kubernetes monitoring is setup for this cluster."
)

func extensionsWithoutK8SMonitoring(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.Extensions().IsAnyEnabled() && !dk.IsKubernetesMonitoringRegistrationEnabled() {
		if k8senv.IsKubemonOperandEnabled() {
			return warningExtensionsWithoutK8SMonitoringNew
		}

		return warningExtensionsWithoutK8SMonitoringOld
	}

	return ""
}
