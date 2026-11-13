// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const errorInvalidSelector = "The PrometheusMonitoring %q has an invalid %s: %s"

func invalidSelectors(_ context.Context, _ client.Reader, pm *prometheusmonitoring.PrometheusMonitoring) string {
	targetAllocatorPath := field.NewPath("spec", "targetAllocator")

	if msg := invalidSelector(pm.Name, pm.Spec.TargetAllocator.CustomResourceSelector, targetAllocatorPath.Child("customResourceSelector")); msg != "" {
		return msg
	}

	return invalidSelector(pm.Name, pm.Spec.TargetAllocator.CustomResourceNamespaceSelector, targetAllocatorPath.Child("customResourceNamespaceSelector"))
}

func invalidSelector(pmName string, selector *metav1.LabelSelector, path *field.Path) string {
	errs := metav1validation.ValidateLabelSelector(selector, metav1validation.LabelSelectorValidationOptions{}, path)
	if len(errs) == 0 {
		return ""
	}

	return fmt.Sprintf(errorInvalidSelector, pmName, path.String(), errs.ToAggregate().Error())
}
