// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func newPM(name, namespace string, selector *metav1.LabelSelector) *prometheusmonitoring.PrometheusMonitoring {
	return &prometheusmonitoring.PrometheusMonitoring{
		Name:      name,
		Namespace: namespace,
		Spec: prometheusmonitoring.PrometheusMonitoringSpec{
			TargetAllocator: prometheusmonitoring.TargetAllocatorSpec{CustomResourceSelector: selector},
		},
	}
}

func TestOverlappingSelectorWarning(t *testing.T) {
	selectorA := &metav1.LabelSelector{MatchLabels: map[string]string{"team": "a"}}
	selectorB := &metav1.LabelSelector{MatchLabels: map[string]string{"team": "b"}}

	t.Run("same selector in same namespace warns", func(t *testing.T) {
		clt := fake.NewClient(newPM("other", "ns", selectorA.DeepCopy()))

		msg := overlappingSelectorWarning(t.Context(), clt, newPM("pm", "ns", selectorA))

		assert.Contains(t, msg, "other")
	})

	t.Run("same name (update of itself) does not warn", func(t *testing.T) {
		clt := fake.NewClient(newPM("pm", "ns", selectorA))

		assert.Empty(t, overlappingSelectorWarning(t.Context(), clt, newPM("pm", "ns", selectorA)))
	})

	t.Run("different selector does not warn", func(t *testing.T) {
		clt := fake.NewClient(newPM("other", "ns", selectorB))

		assert.Empty(t, overlappingSelectorWarning(t.Context(), clt, newPM("pm", "ns", selectorA)))
	})

	t.Run("same selector in other namespace does not warn", func(t *testing.T) {
		clt := fake.NewClient(newPM("other", "other-ns", selectorA))

		assert.Empty(t, overlappingSelectorWarning(t.Context(), clt, newPM("pm", "ns", selectorA)))
	})

	t.Run("both unset selectors warn", func(t *testing.T) {
		clt := fake.NewClient(newPM("other", "ns", nil))

		assert.NotEmpty(t, overlappingSelectorWarning(t.Context(), clt, newPM("pm", "ns", nil)))
	})

	t.Run("different dynakubes without any selector warn", func(t *testing.T) {
		other := newPM("other", "ns", nil)
		other.Spec.DynaKubeName = "dk-2"
		pm := newPM("pm", "ns", nil)
		pm.Spec.DynaKubeName = "dk-1"
		clt := fake.NewClient(other)

		msg := overlappingSelectorWarning(t.Context(), clt, pm)

		assert.Contains(t, msg, "other")
	})

	t.Run("same selector but different namespace selectors does not warn", func(t *testing.T) {
		other := newPM("other", "ns", selectorA)
		other.Spec.TargetAllocator.CustomResourceNamespaceSelector = selectorB
		pm := newPM("pm", "ns", selectorA)
		pm.Spec.TargetAllocator.CustomResourceNamespaceSelector = selectorA
		clt := fake.NewClient(other)

		assert.Empty(t, overlappingSelectorWarning(t.Context(), clt, pm))
	})

	t.Run("subset of labels warns", func(t *testing.T) {
		superset := &metav1.LabelSelector{MatchLabels: map[string]string{"team": "a", "env": "prod"}}
		clt := fake.NewClient(newPM("other", "ns", superset))

		assert.NotEmpty(t, overlappingSelectorWarning(t.Context(), clt, newPM("pm", "ns", selectorA)))
	})

	t.Run("order of expression values is ignored", func(t *testing.T) {
		exprA := &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "team", Operator: metav1.LabelSelectorOpIn, Values: []string{"a", "b"}},
		}}
		exprB := &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "team", Operator: metav1.LabelSelectorOpIn, Values: []string{"b", "a"}},
		}}
		clt := fake.NewClient(newPM("other", "ns", exprB))

		assert.NotEmpty(t, overlappingSelectorWarning(t.Context(), clt, newPM("pm", "ns", exprA)))
	})

	t.Run("nil and empty namespace selector warn", func(t *testing.T) {
		other := newPM("other", "ns", selectorA)
		other.Spec.TargetAllocator.CustomResourceNamespaceSelector = &metav1.LabelSelector{}
		clt := fake.NewClient(other)

		assert.NotEmpty(t, overlappingSelectorWarning(t.Context(), clt, newPM("pm", "ns", selectorA)))
	})

	t.Run("nil namespace selector overlaps with a restrictive one", func(t *testing.T) {
		other := newPM("other", "ns", selectorA)
		other.Spec.TargetAllocator.CustomResourceNamespaceSelector = selectorB
		clt := fake.NewClient(other)

		assert.NotEmpty(t, overlappingSelectorWarning(t.Context(), clt, newPM("pm", "ns", selectorA)))
	})

	t.Run("list error is ignored", func(t *testing.T) {
		clt := fake.NewClientWithInterceptors(interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return assert.AnError
			},
		})

		assert.Empty(t, overlappingSelectorWarning(t.Context(), clt, newPM("pm", "ns", selectorA)))
	})

	t.Run("surfaced through ValidateCreate", func(t *testing.T) {
		validator := &Validator{apiReader: fake.NewClient(newPM("other", "ns", selectorA))}

		warnings, err := validator.ValidateCreate(t.Context(), newPM("pm", "ns", selectorA))

		require.NoError(t, err)
		assert.Len(t, warnings, 1)
	})
}
