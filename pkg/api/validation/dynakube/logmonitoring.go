package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
)

const (
	warningLogMonitoringIgnoredTemplate = "The Dynakube's `spec.templates.logMonitoring` section is skipped as the `spec.oneagent` section is also configured."
	errorLogMonitoringMissingImage      = `The Dynakube's specification specifies standalone Log monitoring, but no image repository/tag is configured.`
)

func ignoredLogMonitoringTemplate(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.LogMonitoring().IsStandalone() {
		return ""
	}

	if dk.OneAgent().IsDaemonsetRequired() && dk.LogMonitoring().TemplateSpec != nil {
		return warningLogMonitoringIgnoredTemplate
	}

	return ""
}

func missingLogMonitoringImage(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if !dk.LogMonitoring().IsStandalone() {
		return ""
	}

	if dk.FF().IsPublicRegistry() {
		return ""
	}

	// For new DynaKubes (status not yet set), check the token secret directly.
	hasPlatformToken, err := token.NewReader(dv.apiReader, dk).HasPlatformToken(ctx)
	if err == nil && hasPlatformToken {
		return ""
	}

	if dk.LogMonitoring().TemplateSpec == nil || !dk.LogMonitoring().ImageRef.HasImage() {
		return errorLogMonitoringMissingImage
	}

	return ""
}
