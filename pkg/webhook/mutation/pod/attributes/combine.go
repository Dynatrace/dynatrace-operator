// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package attributes

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8spod"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
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

const (
	InvalidMetadataAnnotationSize = "Value of the metadata enrichment annotation exceeds the maximum allowed size of %d, actual size: %d"
)

func (attrs *Pod) ApplyJSONAnnotationToPod(pod *corev1.Pod) error {
	json, err := attrs.combineForJSONAnnotation()
	if err != nil {
		return err
	}

	metadataSizeLimit := k8senv.GetMetadaSizeLimit().Resolve(nil).ResolvedValue //nolint:staticcheck
	if len(json) > metadataSizeLimit {
		errMsg := fmt.Sprintf(InvalidMetadataAnnotationSize, metadataSizeLimit, len(json))

		return dtwebhook.MutatorError{
			Err:      errors.New(errMsg),
			Annotate: setNotInjectedAnnotationFunc(errMsg),
		}
	}

	k8spod.SetAnnotationIfNotExists(pod, metadataenrichment.Annotation, json)

	return nil
}

func setNotInjectedAnnotationFunc(reason string) func(*corev1.Pod) {
	return func(pod *corev1.Pod) {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}

		pod.Annotations[dtwebhook.AnnotationDynatraceInjected] = "false"
		pod.Annotations[dtwebhook.AnnotationDynatraceReason] = reason
	}
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
