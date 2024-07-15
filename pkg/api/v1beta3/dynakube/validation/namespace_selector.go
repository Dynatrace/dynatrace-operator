package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	errorConflictingNamespaceSelector = `The DynaKube's specification tries to inject into namespaces where another Dynakube already injects into, which is not supported.
Make sure the namespaceSelector doesn't conflict with other Dynakubes namespaceSelector`

	errorNamespaceSelectorMatchLabelsViolateLabelSpec = "The DynaKube's namespaceSelector contains matchLabels that are not conform to spec."
)

func conflictingNamespaceSelector(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if !dk.NeedAppInjection() && !dk.MetadataEnrichmentEnabled() {
		return ""
	}

	dkMapper := mapper.NewDynakubeMapper(ctx, nil, dv.apiReader, dk.Namespace, dk)

	_, err := dkMapper.MatchingNamespaces()
	if err != nil && err.Error() == mapper.ErrorConflictingNamespace {
		log.Info("requested dynakube has conflicting namespaceSelector", "name", dk.Name, "namespace", dk.Namespace)

		return errorConflictingNamespaceSelector
	}

	return ""
}

func namespaceSelectorViolateLabelSpec(_ context.Context, _ *Validator, dk *dynakube.DynaKube) string {
	errs := validation.ValidateLabelSelector(dk.OneAgentNamespaceSelector(), validation.LabelSelectorValidationOptions{AllowInvalidLabelValueInSelector: false}, field.NewPath("spec", "namespaceSelector"))
	if len(errs) == 0 {
		return ""
	}

	return fmt.Sprintf("%s (%s)", errorNamespaceSelectorMatchLabelsViolateLabelSpec, errs)
}
