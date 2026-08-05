// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	errorConflictingNamespaceSelector = `The DynaKube's specification tries to inject into namespaces where another Dynakube already injects into, which is not supported.
Make sure the namespaceSelector doesn't conflict with other Dynakubes namespaceSelector`

	errorInvalidOneAgentNamespaceSelector           = "OneAgent namespaceSelector contains invalid matchLabels or matchExpressions."
	errorInvalidMetadataEnrichmentNamespaceSelector = "Metadata enrichment namespaceSelector contains invalid matchLabels or matchExpressions."
	errorInvalidOTLPExporterNamespaceSelector       = "OTLP exporter configuration namespaceSelector contains invalid matchLabels or matchExpressions."
)

func conflictingNamespaceSelector(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	log := logd.FromContext(ctx)

	if !dk.OneAgent().IsAppInjectionNeeded() &&
		!dk.MetadataEnrichment().IsEnabled() &&
		!dk.OTLPExporterConfiguration().IsEnabled() {
		return ""
	}

	dkMapper := mapper.NewDynakubeMapper(ctx, nil, dv.apiReader, dk.Namespace, dk)

	_, err := dkMapper.MatchingNamespaces()
	if err != nil && err.Error() == mapper.ErrorConflictingNamespace {
		log.Info("requested dynakube has conflicting namespaceSelector")

		return errorConflictingNamespaceSelector
	}

	return ""
}

func invalidOneAgentNamespaceSelector(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	var (
		oneAgentSelector *metav1.LabelSelector
		oneAgentMode     string
	)

	switch oa := dk.OneAgent(); {
	case oa.IsCloudNativeFullstackMode():
		oneAgentSelector = &oa.CloudNativeFullStack.NamespaceSelector
		oneAgentMode = "cloudNativeFullStack"
	case oa.IsApplicationMonitoringMode():
		oneAgentSelector = &oa.ApplicationMonitoring.NamespaceSelector
		oneAgentMode = "applicationMonitoring"
	}

	errs := validation.ValidateLabelSelector(oneAgentSelector, validation.LabelSelectorValidationOptions{}, field.NewPath("spec", "oneAgent", oneAgentMode, "namespaceSelector"))
	if len(errs) == 0 {
		return ""
	}

	logErrorList(ctx, errorInvalidOneAgentNamespaceSelector, errs)

	return errorInvalidOneAgentNamespaceSelector
}

func invalidMetadataNamespaceSelectors(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	errs := validation.ValidateLabelSelector(
		&dk.Spec.MetadataEnrichment.NamespaceSelector, validation.LabelSelectorValidationOptions{},
		field.NewPath("spec", "metadataEnrichment", "namespaceSelector"),
	)

	if len(errs) == 0 {
		return ""
	}

	logErrorList(ctx, errorInvalidMetadataEnrichmentNamespaceSelector, errs)

	return errorInvalidMetadataEnrichmentNamespaceSelector
}

func invalidOTLPExporterNamespaceSelector(ctx context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	if dk.Spec.OTLPExporterConfiguration == nil {
		return ""
	}

	errs := validation.ValidateLabelSelector(
		&dk.Spec.OTLPExporterConfiguration.NamespaceSelector, validation.LabelSelectorValidationOptions{},
		field.NewPath("spec", "metadataEnrichment", "namespaceSelector"),
	)

	if len(errs) == 0 {
		return ""
	}

	logErrorList(ctx, errorInvalidOTLPExporterNamespaceSelector, errs)

	return errorInvalidOTLPExporterNamespaceSelector
}

func logErrorList(ctx context.Context, msg string, errs field.ErrorList) {
	logd.FromContext(ctx).Info(
		msg,
		// circumvent zap error "details" formatting, we just want the string
		"details", errs.ToAggregate().Error(),
	)
}
