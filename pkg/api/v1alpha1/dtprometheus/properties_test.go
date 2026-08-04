// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDTPrometheus_IsDynatracePresetEnabled(t *testing.T) {
	tests := []struct {
		name     string
		preset   *DynatracePresetSpec
		expected bool
	}{
		{"disabled when unset", nil, false},
		{"enabled when present", &DynatracePresetSpec{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtp := &DTPrometheus{Spec: DTPrometheusSpec{DynatracePreset: tt.preset}}
			assert.Equal(t, tt.expected, dtp.IsDynatracePresetEnabled())
		})
	}
}

func TestDTPrometheus_IsTLSEnabled(t *testing.T) {
	tests := []struct {
		name     string
		tls      *TLSSpec
		expected bool
	}{
		{"disabled when unset", nil, false},
		{"enabled when present", &TLSSpec{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtp := &DTPrometheus{Spec: DTPrometheusSpec{TLS: tt.tls}}
			assert.Equal(t, tt.expected, dtp.IsTLSEnabled())
		})
	}
}

func TestDTPrometheus_IsSelfMonitoringEnabled(t *testing.T) {
	tests := []struct {
		name        string
		selfMonSpec *SelfMonitoringSpec
		expected    bool
	}{
		{"disabled when unset", nil, false},
		{"enabled when present", &SelfMonitoringSpec{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtp := &DTPrometheus{Spec: DTPrometheusSpec{SelfMonitoring: tt.selfMonSpec}}
			assert.Equal(t, tt.expected, dtp.IsSelfMonitoringEnabled())
		})
	}
}

func TestDTPrometheus_Conditions(t *testing.T) {
	dtp := &DTPrometheus{}

	conditions := dtp.Conditions()
	*conditions = append(*conditions, metav1.Condition{Type: "Ready"})

	assert.Same(t, &dtp.Status.Conditions, dtp.Conditions())
	assert.Len(t, dtp.Status.Conditions, 1)
	assert.Equal(t, "Ready", dtp.Status.Conditions[0].Type)
}

func TestDTPrometheus_ComponentAccessors(t *testing.T) {
	dtp := &DTPrometheus{}
	dtp.Name = "dtprom"

	t.Run("gateway wraps the spec and derives the name", func(t *testing.T) {
		assert.Equal(t, "dtprom-gateway", dtp.Gateway().GetStatefulSetName())
	})

	t.Run("scraper wraps the spec and derives the name", func(t *testing.T) {
		assert.Equal(t, "dtprom-scraper", dtp.Scraper().GetDeploymentName())
	})

	t.Run("target allocator wraps the spec and derives the name", func(t *testing.T) {
		assert.Equal(t, "dtprom-target-allocator", dtp.TargetAllocator().GetDeploymentName())
	})
}
