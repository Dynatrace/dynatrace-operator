package dynakube

import (
	"context"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	errorConflictingNamespaceSelector = `The DynaKube's specification tries to inject into namespaces where another Dynakube already injects into, which is not supported.
Make sure the namespaceSelector doesn't conflict with other Dynakubes namespaceSelector
`
	errorConflictingNamespaceSelectorNoSelector = `The DynaKube does not specificy namespaces where it should inject into while another Dynakube already injects into namespaces, which is not supported.
Make sure you have a namespaceSelector doesn't conflict with other Dynakubes namespaceSelector
`
	errorNamespaceSelectorMatchLabelsViolateLabelSpec = "The DynaKube's namespaceSelector contains matchLabels that are not conform to spec. A valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character."
)

func conflictingNamespaceSelector(ctx context.Context, dv *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	if !dynakube.NeedAppInjection() {
		return ""
	}
	dkMapper := mapper.NewDynakubeMapper(ctx, dv.clt, dv.apiReader, dynakube.Namespace, dynakube)
	_, err := dkMapper.MatchingNamespaces()
	if err != nil && err.Error() == mapper.ErrorConflictingNamespace {
		if dynakube.NamespaceSelector().MatchExpressions == nil && dynakube.NamespaceSelector().MatchLabels == nil {
			log.Info("requested dynakube has conflicting namespaceSelector", "name", dynakube.Name, "namespace", dynakube.Namespace)
			return errorConflictingNamespaceSelectorNoSelector
		} else {
			log.Info("requested dynakube has conflicting namespaceSelector", "name", dynakube.Name, "namespace", dynakube.Namespace)
			return errorConflictingNamespaceSelector
		}
	}
	return ""
}

func nameSpaceSelectorMatchLabelsViolateLabelSpec(ctx context.Context, _ *dynakubeValidator, dynakube *dynatracev1beta1.DynaKube) string {
	var errs []string
	matchLabels := dynakube.NamespaceSelector().MatchLabels
	for _, v := range matchLabels {
		if v != "" {
			errs = validation.IsValidLabelValue(v)
		}
	}
	if len(errs) == 0 {
		return ""
	}
	return errorNamespaceSelectorMatchLabelsViolateLabelSpec
}
