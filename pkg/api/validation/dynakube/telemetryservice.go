package validation

import (
	"context"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

const (
	errorTelemetryServiceNotEnoughProtocols  = `DynaKube's specification enables the TelemetryService feature, at least one Protocol has to be specified.`
	errorTelemetryServiceUnknownProtocols    = `DynaKube's specification enables the TelemetryService feature, unsupported protocols found on the Protocols list.`
	errorTelemetryServiceDuplicatedProtocols = `DynaKube's specification enables the TelemetryService feature, duplicated protocols found on the Protocols list.`
)

func emptyTelemetryServiceProtocolsList(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.IsTelemetryServiceEnabled() {
		return ""
	}

	if len(dk.TelemetryServiceProtocols()) == 0 {
		log.Info("requested dynakube specify empty list of Protocols")

		return errorTelemetryServiceNotEnoughProtocols
	}

	return ""
}

func unknownTelemetryServiceProtocols(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.IsTelemetryServiceEnabled() {
		return ""
	}

	var unknownProtocols []string

	for _, protocol := range dk.TelemetryServiceProtocols() {
		if !slices.Contains(dynakube.TelemetryServiceKnownProtocols(), protocol) {
			unknownProtocols = append(unknownProtocols, string(protocol))
		}
	}

	if len(unknownProtocols) > 0 {
		log.Info("requested dynakube specify unknown TelemetryService protocol(s)", "protocols", strings.Join(unknownProtocols, ","))

		return errorTelemetryServiceUnknownProtocols
	}

	return ""
}

func duplicatedTelemetryServiceProtocols(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.IsTelemetryServiceEnabled() {
		return ""
	}

	protocolsOccurrences := map[dynakube.TelemetryServiceProtocol]int{}

	for _, protocol := range dk.TelemetryServiceProtocols() {
		if _, ok := protocolsOccurrences[protocol]; !ok {
			protocolsOccurrences[protocol] = 1
		} else {
			protocolsOccurrences[protocol] += 1
		}
	}

	var duplicatedProtocols []string

	for protocol, count := range protocolsOccurrences {
		if count > 1 {
			duplicatedProtocols = append(duplicatedProtocols, string(protocol))
		}
	}

	if len(duplicatedProtocols) > 0 {
		log.Info("requested dynakube specify duplicated TelemetryService protocol(s)", "protocols", strings.Join(duplicatedProtocols, ","))

		return errorTelemetryServiceDuplicatedProtocols
	}

	return ""
}
