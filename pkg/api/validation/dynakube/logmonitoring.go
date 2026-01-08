package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

const (
	warningLogMonitoringIgnoredTemplate      = "The Dynakube's `spec.templates.logMonitoring` section is skipped as the `spec.oneagent` section is also configured."
	warningLogMonitoringWithoutK8SMonitoring = "The Dynakube is configured for Log monitoring without an ActiveGate with `kubernetes-monitoring` enabled or the `automatic-kubernetes-api-monitoring` feature flag. You need to ensure that Kubernetes monitoring is setup for this cluster."
	errorLogMonitoringMissingImage           = `The Dynakube's specification specifies standalone Log monitoring, but no image repository/tag is configured.`
)

func logMonitoringWithoutK8SMonitoring(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.LogMonitoring().IsEnabled() && (!dk.ActiveGate().IsKubernetesMonitoringEnabled() || !dk.FF().IsAutomaticK8sAPIMonitoring()) {
		return warningLogMonitoringWithoutK8SMonitoring
	}

	return ""
}

func ignoredLogMonitoringTemplate(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.LogMonitoring().IsStandalone() {
		return ""
	}

	if dk.OneAgent().IsDaemonsetRequired() && dk.LogMonitoring().TemplateSpec != nil {
		return warningLogMonitoringIgnoredTemplate
	}

	return ""
}

func missingLogMonitoringImage(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.LogMonitoring().IsStandalone() {
		return ""
	}

	if dk.LogMonitoring().TemplateSpec == nil ||
		dk.LogMonitoring().ImageRef.Repository == "" || dk.LogMonitoring().ImageRef.Tag == "" {
		return errorLogMonitoringMissingImage
	}

	return ""
}
