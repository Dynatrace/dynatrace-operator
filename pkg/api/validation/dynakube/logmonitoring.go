package validation

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/sanitize"
)

const (
	warningLogMonitoringIgnoredTemplate = "The Dynakube's `spec.templates.logMonitoring` section is skipped as the `spec.oneagent` section is also configured."
	errorLogMonitoringMissingImage      = `The Dynakube's specification specifies standalone Log monitoring, but no image repository/tag is configured.`
	errorInvalidLogmonArgument          = "The DynaKube' spec.templates.logMonitoring.args contains invalid arguments. Make sure to remove forbidden characters (newline, tab, carriage return, null) from the value in your custom resource."
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

	if dk.LogMonitoring().TemplateSpec == nil || !dk.LogMonitoring().ImageRef.HasImage() {
		return errorLogMonitoringMissingImage
	}

	return ""
}

func invalidLogmonArguments(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	args := dk.LogMonitoring().Template().Args
	for _, arg := range args {
		if strings.ContainsAny(arg, sanitize.InvalidCommandLineCharset) {
			return errorInvalidLogmonArgument
		}
	}

	return ""
}
