package attributes

import (
	"encoding/json"
	"maps"

	"github.com/pkg/errors"
)

type combinationCase uint

const (
	withDeprecated combinationCase = 1 << iota
	withWorkloadInfo
	withPodInfo
	withClusterInfo
	withContainerAttrs
	withDynakube
	withNamespaceAnnotations
	withRules
	withRulesPropagate
	withPodAnnotations
	withCustom
)

const (
	caseAll = withWorkloadInfo | withPodInfo | withClusterInfo |
		withContainerAttrs | withDynakube | withNamespaceAnnotations |
		withRules | withRulesPropagate | withPodAnnotations | withCustom

	caseMetadataAnnotations = withWorkloadInfo | withDynakube |
		withNamespaceAnnotations | withRulesPropagate

	caseJSONAnnotation = withDynakube | withNamespaceAnnotations |
		withRules | withRulesPropagate | withPodAnnotations
)

// combine copies maps into a single result in fixed precedence order (low → high).
// Only maps whose bit is set in c are included.
func (attrs *Pod) combine(c combinationCase, containerAttrs map[string]string) map[string]string {
	type layer struct {
		flag combinationCase
		data map[string]string
	}

	layers := []layer{
		{withDeprecated, attrs.deprecated},
		{withWorkloadInfo, attrs.workloadInfo},
		{withPodInfo, attrs.podInfo},
		{withClusterInfo, attrs.clusterInfo},
		{withContainerAttrs, containerAttrs},
		{withDynakube, attrs.dynakube},
		{withNamespaceAnnotations, attrs.namespaceAnnotations},
		{withRules, attrs.rules},
		{withRulesPropagate, attrs.rulesPropagate},
		{withPodAnnotations, attrs.podAnnotations},
		{withCustom, attrs.custom},
	}

	combined := make(map[string]string)

	for _, l := range layers {
		if c&l.flag != 0 {
			maps.Copy(combined, l.data)
		}
	}

	return combined
}

func flattenContainerAttrs(containerAttrs []Container) map[string]string {
	m := make(map[string]string)
	for _, c := range containerAttrs {
		maps.Copy(m, c.ToMap())
	}

	return m
}

func (attrs *Pod) combineAll(containerAttrs ...Container) map[string]string {
	c := caseAll
	if attrs.useDeprecated {
		c |= withDeprecated
	}

	return attrs.combine(c, flattenContainerAttrs(containerAttrs))
}

func (attrs *Pod) combineForMetadataAnnotations() map[string]string {
	return attrs.combine(caseMetadataAnnotations, nil)
}

func (attrs *Pod) combineForJSONAnnotation() (string, error) {
	combined := attrs.combine(caseJSONAnnotation, nil)

	marshaledAnnotations, err := json.Marshal(combined)
	if err != nil {
		return "", errors.WithMessage(errors.WithStack(err), "could not marshal metadata annotations to JSON")
	}

	return string(marshaledAnnotations), nil
}
