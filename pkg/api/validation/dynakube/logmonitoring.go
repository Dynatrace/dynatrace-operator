package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	warningLogMonitoringIgnoredTemplate = "The Dynakube's `spec.templates.logMonitoring` section is skipped as the `spec.oneagent` section is also configured."
)

func ignoredLogMonitoringTemplate(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.LogMonitoring().IsStandalone() {
		return ""
	}

	if dk.NeedsOneAgent() && dk.LogMonitoring().TemplateSpec != nil {
		return warningLogMonitoringIgnoredTemplate
	}

	return ""
}
