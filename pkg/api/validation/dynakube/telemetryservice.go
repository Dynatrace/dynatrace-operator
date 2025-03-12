package validation

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	errorTelemetryIngestNotEnoughProtocols  = `DynaKube's specification enables the TelemetryIngest feature, at least one Protocol has to be specified.`
	errorTelemetryIngestUnknownProtocols    = `DynaKube's specification enables the TelemetryIngest feature, unsupported protocols found on the Protocols list.`
	errorTelemetryIngestDuplicatedProtocols = `DynaKube's specification enables the TelemetryIngest feature, duplicated protocols found on the Protocols list.`
	errorTelemetryIngestNoDNS1053Label      = `DynaKube's specification enables the TelemetryIngest feature, the telemetry service name violates DNS-1035.
    [The length limit for the name is %d. Additionally a DNS-1035 name must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character (e.g. 'my-name',  or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')]
	`
)

func emptyTelemetryIngestProtocolsList(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryIngest().IsEnabled() {
		return ""
	}

	if len(dk.TelemetryIngest().GetProtocols()) == 0 {
		log.Info("requested dynakube specify empty list of Protocols")

		return errorTelemetryIngestNotEnoughProtocols
	}

	return ""
}

func unknownTelemetryIngestProtocols(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryIngest().IsEnabled() {
		return ""
	}

	var unknownProtocols []string

	for _, protocol := range dk.TelemetryIngest().GetProtocols() {
		if !slices.Contains(otelcgen.RegisteredProtocols, protocol) {
			unknownProtocols = append(unknownProtocols, string(protocol))
		}
	}

	if len(unknownProtocols) > 0 {
		log.Info("requested dynakube specify unknown TelemetryIngest protocol(s)", "protocols", strings.Join(unknownProtocols, ","))

		return errorTelemetryIngestUnknownProtocols
	}

	return ""
}

func duplicatedTelemetryIngestProtocols(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryIngest().IsEnabled() {
		return ""
	}

	protocolsOccurrences := map[otelcgen.Protocol]int{}

	for _, protocol := range dk.TelemetryIngest().GetProtocols() {
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
		log.Info("requested dynakube specify duplicated TelemetryIngest protocol(s)", "protocols", strings.Join(duplicatedProtocols, ","))

		return errorTelemetryIngestDuplicatedProtocols
	}

	return ""
}

func invalidTelemetryIngestName(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryIngest().IsEnabled() {
		return ""
	}

	var errs []string

	if dk.TelemetryIngest().ServiceName != "" {
		errs = validation.IsDNS1035Label(dk.Spec.TelemetryIngest.ServiceName)
	}

	if len(errs) == 0 {
		return ""
	}

	return invalidTelemetryIngestNameErrorMessage()
}

func invalidTelemetryIngestNameErrorMessage() string {
	return fmt.Sprintf(errorTelemetryIngestNoDNS1053Label, validation.DNS1035LabelMaxLength)
}
