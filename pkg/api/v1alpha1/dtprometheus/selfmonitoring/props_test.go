// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package selfmonitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	spec := &Spec{}

	selfMonitoring := New(spec, "dtprom")

	assert.Same(t, spec, selfMonitoring.Spec)
	assert.Equal(t, "dtprom"+NameSuffix, selfMonitoring.GetName())
}

func TestSelfMonitoring_GetName(t *testing.T) {
	selfMonitoring := New(&Spec{}, "dtprom")

	assert.Equal(t, "dtprom-self-monitoring", selfMonitoring.GetName())
}

func TestSelfMonitoring_SetName(t *testing.T) {
	selfMonitoring := New(&Spec{}, "old")

	selfMonitoring.SetName("new")

	assert.Equal(t, "new-self-monitoring", selfMonitoring.GetName())
}

func TestSelfMonitoring_IsEnabled(t *testing.T) {
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
			selfMonitoring := New(&Spec{Enabled: tt.enabled}, "dtprom")
			assert.Equal(t, tt.expected, selfMonitoring.IsEnabled())
		})
	}
}
