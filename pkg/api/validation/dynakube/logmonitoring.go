package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	warningLogMonitoringIgnoredTemplate    = "The Dynakube's `spec.templates.logMonitoring` section is skipped as the `spec.oneagent` section is also configured."
	errorLogMonitoringMissingImage         = `The Dynakube's specification specifies standalone Log monitoring, but no image repository/tag is configured.`
	errorLogMonitoringWithoutK8SMonitoring = "The Dynakube's specification specifies Log monitoring without an ActiveGate with kubernetes-monitoring enabled or the automatic-kubernetes-monitoring feature flag. This is not allowed as Kubernetes settings are needed for Log monitoring."
)

func logMonitoringWithoutK8SMonitoring(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.LogMonitoring().IsEnabled() && !dk.ActiveGate().IsKubernetesMonitoringEnabled() {
		if !dk.FeatureAutomaticKubernetesApiMonitoring() {
			return errorLogMonitoringWithoutK8SMonitoring
		}

		return ""
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
