// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInvalidSelectors(t *testing.T) {
	clt := fake.NewClient()
	invalid := &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
		{Key: "team", Operator: "Bogus", Values: []string{"a"}},
	}}
	valid := &metav1.LabelSelector{MatchLabels: map[string]string{"team": "a"}}

	t.Run("nil and valid selectors are accepted", func(t *testing.T) {
		pm := newPM("pm", "ns", valid)
		pm.Spec.TargetAllocator.CustomResourceNamespaceSelector = &metav1.LabelSelector{}

		assert.Empty(t, invalidSelectors(t.Context(), clt, pm))
		assert.Empty(t, invalidSelectors(t.Context(), clt, newPM("pm", "ns", nil)))
	})

	t.Run("invalid customResourceSelector is rejected", func(t *testing.T) {
		msg := invalidSelectors(t.Context(), clt, newPM("pm", "ns", invalid))

		assert.Contains(t, msg, "spec.targetAllocator.customResourceSelector")
		assert.Contains(t, msg, `"pm"`)
	})

	t.Run("invalid customResourceNamespaceSelector is rejected", func(t *testing.T) {
		pm := newPM("pm", "ns", valid)
		pm.Spec.TargetAllocator.CustomResourceNamespaceSelector = invalid

		assert.Contains(t, invalidSelectors(t.Context(), clt, pm), "spec.targetAllocator.customResourceNamespaceSelector")
	})
}
