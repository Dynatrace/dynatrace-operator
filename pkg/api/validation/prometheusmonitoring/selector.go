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
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const warningOverlappingCustomResourceSelector = "PrometheusMonitoring(s) %s in this namespace use the same customResourceSelector and customResourceNamespaceSelector. Their target allocators pick up the same ServiceMonitors, PodMonitors, ScrapeConfigs and Probes, so metrics are scraped multiple times."

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
	return equalSelector(a.CustomResourceSelector, b.CustomResourceSelector) &&
		equalSelector(a.CustomResourceNamespaceSelector, b.CustomResourceNamespaceSelector)
}

func equalSelector(a, b *metav1.LabelSelector) bool {
	if a == nil || b == nil {
		return a == b
	}

	return equality.Semantic.DeepEqual(a, b)
}
