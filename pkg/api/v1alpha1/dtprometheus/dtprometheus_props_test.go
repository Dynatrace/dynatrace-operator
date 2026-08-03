// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func TestDtPrometheus_IsSelfMonitoringEnabled(t *testing.T) {
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
			dtp := &DtPrometheus{Spec: DtPrometheusSpec{SelfMonitoring: tt.selfMonSpec}}
			assert.Equal(t, tt.expected, dtp.IsSelfMonitoringEnabled())
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
