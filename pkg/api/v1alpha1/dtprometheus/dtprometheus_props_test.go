// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDtPrometheus_GetDynaKubeName(t *testing.T) {
	dtp := &DtPrometheus{Spec: DtPrometheusSpec{DynaKubeName: "dynakube"}}

	assert.Equal(t, "dynakube", dtp.GetDynaKubeName())
}

func TestDtPrometheus_IsDynatracePresetEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{"disabled", false, false},
		{"enabled", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtp := &DtPrometheus{Spec: DtPrometheusSpec{DynatracePreset: DynatracePresetSpec{Enabled: tt.enabled}}}
			assert.Equal(t, tt.expected, dtp.IsDynatracePresetEnabled())
		})
	}
}

func TestDtPrometheus_IsTLSEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{"disabled", false, false},
		{"enabled", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtp := &DtPrometheus{Spec: DtPrometheusSpec{TLS: TLSSpec{Enabled: tt.enabled}}}
			assert.Equal(t, tt.expected, dtp.IsTLSEnabled())
		})
	}
}

func TestDtPrometheus_Conditions(t *testing.T) {
	dtp := &DtPrometheus{}

	conditions := dtp.Conditions()
	*conditions = append(*conditions, metav1.Condition{Type: "Ready"})

	assert.Same(t, &dtp.Status.Conditions, dtp.Conditions())
	assert.Len(t, dtp.Status.Conditions, 1)
	assert.Equal(t, "Ready", dtp.Status.Conditions[0].Type)
}

func TestDtPrometheus_ComponentAccessors(t *testing.T) {
	dtp := &DtPrometheus{}
	dtp.Name = "dtprom"
	dtp.Spec.Gateway.Replicas = new(int32(3))
	dtp.Spec.Scraper.Replicas = new(int32(4))
	dtp.Spec.TargetAllocator.Replicas = new(int32(5))
	dtp.Spec.SelfMonitoring.Enabled = true

	t.Run("gateway wraps the spec and derives the name", func(t *testing.T) {
		assert.Equal(t, "dtprom-gateway", dtp.Gateway().GetName())
		assert.Equal(t, int32(3), dtp.Gateway().GetReplicas())
	})

	t.Run("scraper wraps the spec and derives the name", func(t *testing.T) {
		assert.Equal(t, "dtprom-scraper", dtp.Scraper().GetName())
		assert.Equal(t, int32(4), dtp.Scraper().GetReplicas())
	})

	t.Run("target allocator wraps the spec and derives the name", func(t *testing.T) {
		assert.Equal(t, "dtprom-target-allocator", dtp.TargetAllocator().GetName())
		assert.Equal(t, int32(5), dtp.TargetAllocator().GetReplicas())
	})

	t.Run("self-monitoring wraps the spec and derives the name", func(t *testing.T) {
		assert.Equal(t, "dtprom-self-monitoring", dtp.SelfMonitoring().GetName())
		assert.True(t, dtp.SelfMonitoring().IsEnabled())
	})
}
