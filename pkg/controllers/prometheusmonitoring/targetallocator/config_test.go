// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTelemetry(t *testing.T) {
	t.Run("self-monitoring disabled", func(t *testing.T) {
		assert.Nil(t, buildTelemetry(&dynakube.DynaKube{Name: "dk"}))
	})

	t.Run("self-monitoring enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Name: "dk",
			Spec: dynakube.DynaKubeSpec{SelfMonitoring: &dynakube.SelfMonitoringSpec{}},
		}

		tel := buildTelemetry(dk)

		require.NotNil(t, tel)
		assert.Equal(t, "normal", tel.Metrics.Level)
		require.Len(t, tel.Metrics.Readers, 1)

		otlp := tel.Metrics.Readers[0].Periodic.Exporter.OTLPHTTP
		assert.Equal(t, "http://dk-self-monitoring:4318", otlp.Endpoint)
		assert.Equal(t, "delta", otlp.TemporalityPreference)
	})
}
