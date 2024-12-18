package validation

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"golang.org/x/net/context"
	"slices"
	"strings"
)

const (
	errorTelemetryServiceNotEnoughProtocols = `DynaKube's specification enables the TelemetryService feature, at least one Protocol has to be specified.`
	errorTelemetryServiceTooManyProtocols   = `DynaKube's specification enables the TelemetryService feature, too many Protocols is specified.`
	errorTelemetryServiceUnknownProtocols   = `DynaKube's specification enables the TelemetryService feature, unsupported protocols found on the Protocols list.`
)

func telemetryServiceProtocols(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.IsTelemetryServiceEnabled() {
		return ""
	}

	if len(dk.TelemetryServiceProtocols()) == 0 {
		log.Info("requested dynakube specify empty list of Protocols")

		return errorTelemetryServiceNotEnoughProtocols
	}

	if len(dk.TelemetryServiceProtocols()) > len(dynakube.TelemetryServiceKnownProtocols()) {
		log.Info("requested dynakube specify too many TelemetryService protocol(s).", "specified protocols", len(dk.Spec.TelemetryService.Protocols), "number of supported protocols", len(dynakube.TelemetryServiceKnownProtocols()))

		return errorTelemetryServiceTooManyProtocols
	}

	var unknownProtocols []string
	for _, protocol := range dk.TelemetryServiceProtocols() {
		if !slices.Contains(dynakube.TelemetryServiceKnownProtocols(), protocol) {
			unknownProtocols = append(unknownProtocols, protocol)
		}
	}

	if len(unknownProtocols) > 0 {
		log.Info("requested dynakube specify unknown TelemetryService protocol(s)", "unknown protocols", strings.Join(unknownProtocols, ","))

		return errorTelemetryServiceUnknownProtocols
	}

	return ""
}
