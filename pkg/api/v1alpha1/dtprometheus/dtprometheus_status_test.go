// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/stretchr/testify/assert"
)

func TestDtPrometheusStatus_SetPhase(t *testing.T) {
	t.Run("sets the phase and reports a change", func(t *testing.T) {
		dtpStatus := &DtPrometheusStatus{}

		changed := dtpStatus.SetPhase(status.Running)

		assert.True(t, changed)
		assert.Equal(t, status.Running, dtpStatus.Phase)
	})

	t.Run("reports no change when the phase is unchanged", func(t *testing.T) {
		dtpStatus := &DtPrometheusStatus{Phase: status.Running}

		changed := dtpStatus.SetPhase(status.Running)

		assert.False(t, changed)
		assert.Equal(t, status.Running, dtpStatus.Phase)
	})
}
