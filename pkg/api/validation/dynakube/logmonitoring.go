package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	warningLogMonitoringIgnoredTemplate = "The Dynakube's `spec.templates.logMonitoring` section is skipped as the `spec.oneagent` section is also configured."
	errorLogMonitoringMissingImage      = `The Dynakube's specification specifies standalone Log monitoring, but no image repository/tag is configured.`
)

func ignoredLogMonitoringTemplate(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.LogMonitoring().IsStandalone() {
		return ""
	}

	if dk.OneAgent().NeedsOneAgent() && dk.LogMonitoring().TemplateSpec != nil {
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
