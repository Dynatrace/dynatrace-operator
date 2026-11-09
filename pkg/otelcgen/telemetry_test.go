// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsTelemetry(t *testing.T) {
	cfg := &Config{Service: ServiceConfig{Telemetry: NewMetricsTelemetry("http://dk-self-monitoring:4318")}}
	c, err := cfg.Marshal()
	require.NoError(t, err)

	expectedOutput, err := os.ReadFile(filepath.Join("testdata", "telemetry_metrics.yaml"))
	require.NoError(t, err)
	assert.YAMLEq(t, string(expectedOutput), string(c))
}
