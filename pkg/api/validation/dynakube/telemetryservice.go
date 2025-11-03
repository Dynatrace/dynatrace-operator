package validation

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorTelemetryIngestNotEnoughProtocols  = `DynaKube's specification enables the TelemetryIngest feature, at least one Protocol has to be specified.`
	errorTelemetryIngestUnknownProtocols    = `DynaKube's specification enables the TelemetryIngest feature, unsupported protocols found on the Protocols list.`
	errorTelemetryIngestDuplicatedProtocols = `DynaKube's specification enables the TelemetryIngest feature, duplicated protocols found on the Protocols list.`
	errorTelemetryIngestNoDNS1053Label      = `DynaKube's specification enables the TelemetryIngest feature, the telemetry service name violates DNS-1035.
    [The length limit for the name is %d. Additionally a DNS-1035 name must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character (e.g. 'my-name',  or 'abc-123', regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?')]
	`
	errorTelemetryIngestServiceNameInUse     = `The DynaKube's specification enables the TelemetryIngest feature, the telemetry service name is already used by other Dynakube.`
	errorTelemetryIngestForbiddenServiceName = `The DynaKube's specification enables the TelemetryIngest feature, the telemetry service name is incorrect because of forbidden suffix.`
	errorOtelCollectorMissingImage           = `The Dynakube's specification specifies the OTel Collector, but no image repository/tag is configured.`
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

func conflictingTelemetryIngestServiceNames(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryIngest().IsEnabled() {
		return ""
	}

	dkList := &dynakube.DynaKubeList{}
	if err := dv.apiReader.List(ctx, dkList, &client.ListOptions{Namespace: dk.Namespace}); err != nil {
		log.Info("error occurred while listing dynakubes", "err", err.Error())

		return ""
	}

	dkServiceName := dk.TelemetryIngest().GetServiceName()

	for _, otherDk := range dkList.Items {
		if otherDk.Name == dk.Name {
			continue
		}

		if !otherDk.TelemetryIngest().IsEnabled() {
			continue
		}

		otherDkServiceName := otherDk.TelemetryIngest().GetServiceName()

		if otherDkServiceName == dkServiceName {
			log.Info(errorTelemetryIngestServiceNameInUse, "other dynakube name", otherDk.Name, "other telemetry service name", otherDkServiceName, "namespace", otherDk.Namespace)

			return fmt.Sprintf("%s Conflicting Dynakube: %s. Conflicting telemetry service name: %s", errorTelemetryIngestServiceNameInUse, otherDk.Name, otherDkServiceName)
		}
	}

	return ""
}

func forbiddenTelemetryIngestServiceNameSuffix(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryIngest().IsEnabled() {
		return ""
	}

	if dk.TelemetryIngest().ServiceName == "" {
		return ""
	}

	if strings.HasSuffix(dk.TelemetryIngest().ServiceName, consts.ExtensionsControllerSuffix) ||
		strings.HasSuffix(dk.TelemetryIngest().ServiceName, telemetryingest.ServiceNameSuffix) ||
		strings.HasSuffix(dk.TelemetryIngest().ServiceName, "-"+agconsts.MultiActiveGateName) ||
		strings.HasSuffix(dk.TelemetryIngest().ServiceName, "-webhook") {
		log.Info(errorTelemetryIngestForbiddenServiceName, "telemetry service name", dk.TelemetryIngest().ServiceName)

		return fmt.Sprintf("%s Telemetry service name: %s", errorTelemetryIngestForbiddenServiceName, dk.TelemetryIngest().ServiceName)
	}

	return ""
}

func missingOtelCollectorImage(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryIngest().IsEnabled() && !dk.Extensions().IsPrometheusEnabled() {
		return ""
	}

	if dk.Spec.Templates.OpenTelemetryCollector.ImageRef.Repository == "" || dk.Spec.Templates.OpenTelemetryCollector.ImageRef.Tag == "" {
		return errorOtelCollectorMissingImage
	}

	return ""
}
