package attributes

import (
	"encoding/json"
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8spod"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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
	withPodAnnotations
	withCustom
)

const (
	// withDeprecated is not included; combineAll adds it conditionally.
	caseAll = withWorkloadInfo | withPodInfo | withClusterInfo |
		withContainerAttrs | withDynakube | withNamespaceAnnotations |
		withRules | withPodAnnotations | withCustom

	caseJSONAnnotation = withDynakube | withNamespaceAnnotations |
		withRules | withPodAnnotations | withWorkloadInfo
)

func (attrs *Pod) ApplyJSONAnnotationToPod(pod *corev1.Pod) error {
	json, err := attrs.combineForJSONAnnotation()
	if err != nil {
		return err
	}

	k8spod.SetAnnotationIfNotExists(pod, metadataenrichment.Annotation, json)

	return nil
}

// combine copies maps into a single result in fixed precedence order (low → high).
func (attrs *Pod) combine(c combinationCase, containerAttrs map[string]string) map[string]string {
	type layer struct {
		flag combinationCase
		data map[string]string
	}

	// this slice defines the precedence order (lowest to highest), "customer over built in" and "local wins"-policy
	layers := []layer{
		{withDeprecated, attrs.deprecated},
		{withWorkloadInfo, attrs.workloadInfo},
		{withPodInfo, attrs.podInfo},
		{withClusterInfo, attrs.clusterInfo},
		{withContainerAttrs, containerAttrs},
		{withRules, attrs.rules},
		{withDynakube, attrs.dynakube},
		{withNamespaceAnnotations, attrs.namespaceAnnotations},
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

func (attrs *Pod) combineForJSONAnnotation() (string, error) {
	combined := attrs.combine(caseJSONAnnotation, nil)

	marshaledAnnotations, err := json.Marshal(combined)
	if err != nil {
		return "", errors.Wrapf(err, "could not marshal metadata annotations to JSON")
	}

	return string(marshaledAnnotations), nil
}
