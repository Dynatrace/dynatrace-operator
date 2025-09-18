/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package otlpexporterconfiguration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpec_IsOverrideEnabled(t *testing.T) {
	t.Run("overrideEnvVars not set", func(t *testing.T) {
		s := Spec{}

		assert.False(t, s.IsOverrideEnabled())
	})
	t.Run("overrideEnvVars disabled", func(t *testing.T) {
		enabled := false
		s := Spec{
			OverrideEnvVars: &enabled,
		}

		assert.False(t, s.IsOverrideEnabled())
	})
	t.Run("overrideEnvVars enabled", func(t *testing.T) {
		enabled := true
		s := Spec{
			OverrideEnvVars: &enabled,
		}

		assert.True(t, s.IsOverrideEnabled())
	})
}

func TestSpec_IsMetricsEnabled(t *testing.T) {
	t.Run("metrics disabled", func(t *testing.T) {
		s := Spec{}

		assert.False(t, s.IsMetricsEnabled())
	})
	t.Run("metrics enabled", func(t *testing.T) {
		s := Spec{
			Signals: SignalConfiguration{
				Metrics: &MetricsConfiguration{},
			},
		}

		assert.True(t, s.IsMetricsEnabled())
	})
}

func TestSpec_IsTracesEnabled(t *testing.T) {
	t.Run("traces disabled", func(t *testing.T) {
		s := Spec{}

		assert.False(t, s.IsTracesEnabled())
	})
	t.Run("traces enabled", func(t *testing.T) {
		s := Spec{
			Signals: SignalConfiguration{
				Traces: &TracesConfiguration{},
			},
		}

		assert.True(t, s.IsTracesEnabled())
	})
}

func TestSpec_IsLogsEnabled(t *testing.T) {
	t.Run("logs disabled", func(t *testing.T) {
		s := Spec{}

		assert.False(t, s.IsLogsEnabled())
	})
	t.Run("traces enabled", func(t *testing.T) {
		s := Spec{
			Signals: SignalConfiguration{
				Logs: &LogsConfiguration{},
			},
		}

		assert.True(t, s.IsLogsEnabled())
	})
}
