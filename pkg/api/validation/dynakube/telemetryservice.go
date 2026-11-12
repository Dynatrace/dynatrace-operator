// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/types"
	dtvalidation "github.com/Dynatrace/dynatrace-operator/pkg/api/validation"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errorTelemetryIngestNotEnoughProtocols   = `DynaKube's specification enables the TelemetryIngest feature, at least one Protocol has to be specified.`
	errorTelemetryIngestUnknownProtocols     = `DynaKube's specification enables the TelemetryIngest feature, unsupported protocols found on the Protocols list.`
	errorTelemetryIngestServiceNameInUse     = `The DynaKube's specification enables the TelemetryIngest feature, the telemetry service name is already used by other Dynakube.`
	errorTelemetryIngestForbiddenServiceName = `The DynaKube's specification enables the TelemetryIngest feature, the telemetry service name is incorrect because of forbidden suffix.`
	errorOTelCollectorMissingImage           = `The Dynakube's specification specifies the OTel Collector, but no image repository/tag is configured.`
	warningOTelCollectorIgnoredTemplate      = "The Dynakube's `spec.templates.otelCollector` section is skipped as the `spec.telemetryIngest` section is not configured. Consider either removing `spec.templates.otelCollector.imageRef` or enabling `spec.telemetryIngest`."
	warningOTelCollectorDeprecatedTLSRefName = "The DynaKube's `spec.templates.otelCollector.tlsRefName` field is deprecated and has no effect. Use `spec.telemetryIngest.tlsRefName` to configure TLS for the OTel Collector."
)

func emptyTelemetryIngestProtocolsList(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	if !dk.TelemetryIngest().IsEnabled() {
		return ""
	}

	if len(dk.TelemetryIngest().GetProtocols()) == 0 {
		log.Info("requested dynakube specify empty list of Protocols")

		return errorTelemetryIngestNotEnoughProtocols
	}

	return ""
}

func unknownTelemetryIngestProtocols(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

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
	return fmt.Sprintf("%s\n    [The length limit for the name is %d.]",
		dtvalidation.DNS1035LabelViolation("DynaKube's specification enables the TelemetryIngest feature, the telemetry service name"),
		validation.DNS1035LabelMaxLength)
}

func conflictingTelemetryIngestServiceNames(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	if !dk.TelemetryIngest().IsEnabled() {
		return ""
	}

	dkList := &dynakube.DynaKubeList{}
	if err := dv.apiReader.List(ctx, dkList, &client.ListOptions{Namespace: dk.Namespace}); err != nil {
		log.Info("error occurred while listing dynakubes", "err", err.Error())

		return ""
	}

	dkServiceName := dk.TelemetryIngest().GetServiceName()

	for _, otherDK := range dkList.Items {
		if otherDK.Name == dk.Name {
			continue
		}

		if !otherDK.TelemetryIngest().IsEnabled() {
			continue
		}

		otherDKServiceName := otherDK.TelemetryIngest().GetServiceName()

		if otherDKServiceName == dkServiceName {
			log.Info(errorTelemetryIngestServiceNameInUse, "other dynakube name", otherDK.Name, "other telemetry service name", otherDKServiceName)

			return fmt.Sprintf("%s Conflicting Dynakube: %s. Conflicting telemetry service name: %s", errorTelemetryIngestServiceNameInUse, otherDK.Name, otherDKServiceName)
		}
	}

	return ""
}

func forbiddenTelemetryIngestServiceNameSuffix(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

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

func missingOTelCollectorImage(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryIngest().IsEnabled() {
		return ""
	}

	if dk.Spec.Templates.OpenTelemetryCollector.ImageRef.String() != "" {
		return ""
	}

	if dk.FF().IsPublicRegistry() {
		return ""
	}

	return errorOTelCollectorMissingImage
}

func ignoredOTelCollectorTemplate(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if !dk.TelemetryIngest().IsEnabled() && dk.Spec.Templates.OpenTelemetryCollector.ImageRef != (types.ImageRef{}) {
		return warningOTelCollectorIgnoredTemplate
	}

	return ""
}

func deprecatedOTelCollectorTLSRefName(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.Spec.Templates.OpenTelemetryCollector.TLSRefName != "" { //nolint:staticcheck // SA1019: intentionally reading deprecated field to warn about it
		return warningOTelCollectorDeprecatedTLSRefName
	}

	return ""
}
