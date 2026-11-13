// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const warningOverlappingCustomResourceSelector = "PrometheusMonitoring(s) %s in this namespace have an overlapping customResourceSelector and customResourceNamespaceSelector. Their target allocators may pick up the same ServiceMonitors, PodMonitors, ScrapeConfigs and Probes, so metrics may be scraped multiple times."

// overlappingSelectorWarning returns a warning if another PrometheusMonitoring in the same namespace
// uses the same custom resource selectors as pm.
func overlappingSelectorWarning(ctx context.Context, apiReader client.Reader, pm *prometheusmonitoring.PrometheusMonitoring) string {
	log := logd.FromContext(ctx)

	list := &prometheusmonitoring.PrometheusMonitoringList{}
	if err := apiReader.List(ctx, list, client.InNamespace(pm.Namespace)); err != nil {
		log.Info("could not list PrometheusMonitorings to check for overlapping selectors", "err", err.Error())

		return ""
	}

	var names []string

	for _, other := range list.Items {
		if other.Name == pm.Name {
			continue
		}

		if sameSelectors(&pm.Spec.TargetAllocator, &other.Spec.TargetAllocator) {
			names = append(names, other.Name)
		}
	}

	if len(names) == 0 {
		return ""
	}

	slices.Sort(names)

	return fmt.Sprintf(warningOverlappingCustomResourceSelector, strings.Join(names, ", "))
}

func sameSelectors(a, b *prometheusmonitoring.TargetAllocatorSpec) bool {
	return selectorsOverlap(a.CustomResourceSelector, b.CustomResourceSelector) &&
		selectorsOverlap(a.CustomResourceNamespaceSelector, b.CustomResourceNamespaceSelector)
}

// selectorsOverlap reports whether the two selectors can match the same resources.
// A nil selector is treated like an empty one: the target allocator defaults an omitted namespace selector to all namespaces,
// and the CRD defaults an omitted customResourceSelector, so nil never has a special meaning.
// The selectors overlap if the requirements of one are a subset of the requirements of the other.
//
// This is a heuristic and only detects selectors whose requirements are textually contained in each other
// (values are compared as a sorted set per requirement). It does not detect partially overlapping requirements
// on the same key, e.g. "team In [a]" and "team In [a, b]", nor selectors on different keys that can still match the same resource.
func selectorsOverlap(a, b *metav1.LabelSelector) bool {
	reqsA, errA := requirements(a)
	reqsB, errB := requirements(b)

	if errA != nil || errB != nil {
		return false
	}

	return isSubset(reqsA, reqsB) || isSubset(reqsB, reqsA)
}

// requirements normalizes the selector into a set of comparable requirements, the order of values does not matter.
func requirements(s *metav1.LabelSelector) (map[string]struct{}, error) {
	if s == nil {
		s = &metav1.LabelSelector{}
	}

	selector, err := metav1.LabelSelectorAsSelector(s)
	if err != nil {
		return nil, err
	}

	selectorReqs, _ := selector.Requirements()
	reqs := make(map[string]struct{}, len(selectorReqs))

	for _, req := range selectorReqs {
		op := req.Operator()
		if op == selection.Equals || op == selection.DoubleEquals {
			op = selection.In
		}

		reqs[fmt.Sprintf("%s|%s|%s", req.Key(), op, strings.Join(req.Values().List(), ","))] = struct{}{}
	}

	return reqs, nil
}

func isSubset(sub, super map[string]struct{}) bool {
	for req := range sub {
		if _, ok := super[req]; !ok {
			return false
		}
	}

	return true
}
