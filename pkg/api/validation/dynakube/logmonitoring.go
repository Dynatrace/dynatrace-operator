package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	warningLogMonitoringIgnoredTemplate = "The DynaKube's specification tries to configure LogMonitoring Template section and OneAgent at the same time, in which case the LogMonitoring Template section is ignored"
)

func ignoredLogMonitoringTemplates(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.LogMonitoring().IsStandalone() {
		return ""
	}

	if dk.NeedsOneAgent() && dk.LogMonitoring().TemplateSpec != nil {
		return warningLogMonitoringIgnoredTemplate
	}

	return ""
}
