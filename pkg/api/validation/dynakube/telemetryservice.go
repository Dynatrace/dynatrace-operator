package validation

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/telemetryservice"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	errorTelemetryServiceNotEnoughProtocols  = `DynaKube's specification enables the TelemetryService feature, at least one Protocol has to be specified.`
	errorTelemetryServiceUnknownProtocols    = `DynaKube's specification enables the TelemetryService feature, unsupported protocols found on the Protocols list.`
	errorTelemetryServiceDuplicatedProtocols = `DynaKube's specification enables the TelemetryService feature, duplicated protocols found on the Protocols list.`
	errorTelemetryServiceNoDNS1053Label      = `DynaKube's specification enables the TelemetryService feature, the telemetry service name violates DNS-1035.
    [The length limit for the name is %d. Additionally a DNS-1035 name must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character (e.g. 'my-name',  or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')]
	`
)

func emptyTelemetryServiceProtocolsList(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryService().IsEnabled() {
		return ""
	}

	if len(dk.TelemetryService().GetProtocols()) == 0 {
		log.Info("requested dynakube specify empty list of Protocols")

		return errorTelemetryServiceNotEnoughProtocols
	}

	return ""
}

func unknownTelemetryServiceProtocols(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryService().IsEnabled() {
		return ""
	}

	var unknownProtocols []string

	for _, protocol := range dk.TelemetryService().GetProtocols() {
		if !slices.Contains(telemetryservice.KnownProtocols(), protocol) {
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
	if !dk.TelemetryService().IsEnabled() {
		return ""
	}

	protocolsOccurrences := map[telemetryservice.Protocol]int{}

	for _, protocol := range dk.TelemetryService().GetProtocols() {
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

func invalidTelemetryServiceName(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryService().IsEnabled() {
		return ""
	}

	var errs []string

	if dk.TelemetryService().ServiceName != "" {
		errs = validation.IsDNS1035Label(dk.Spec.TelemetryService.ServiceName)
	}

	if len(errs) == 0 {
		return ""
	}

	return invalidTelemetryServiceNameErrorMessage()
}

func invalidTelemetryServiceNameErrorMessage() string {
	return fmt.Sprintf(errorTelemetryServiceNoDNS1053Label, validation.DNS1035LabelMaxLength)
}
