// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/stretchr/testify/assert"
)

func TestPrometheusMonitoringStatus_SetPhase(t *testing.T) {
	t.Run("sets the phase and reports a change", func(t *testing.T) {
		pmStatus := &PrometheusMonitoringStatus{}

		changed := pmStatus.SetPhase(status.Running)

		assert.True(t, changed)
		assert.Equal(t, status.Running, pmStatus.Phase)
	})

	t.Run("reports no change when the phase is unchanged", func(t *testing.T) {
		pmStatus := &PrometheusMonitoringStatus{Phase: status.Running}

		changed := pmStatus.SetPhase(status.Running)

		assert.False(t, changed)
		assert.Equal(t, status.Running, pmStatus.Phase)
	})
}
