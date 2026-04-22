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

package otlp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestSpec_IsOverrideEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   ExporterConfiguration
		expected bool
	}{
		{
			name:     "overrideEnvVars not set",
			config:   ExporterConfiguration{},
			expected: false,
		},
		{
			name: "overrideEnvVars disabled",
			config: ExporterConfiguration{Spec: &ExporterConfigurationSpec{
				OverrideEnvVars: ptr.To(false),
			}},
			expected: false,
		},
		{
			name: "overrideEnvVars enabled",
			config: ExporterConfiguration{Spec: &ExporterConfigurationSpec{
				OverrideEnvVars: ptr.To(true),
			}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.IsOverrideEnvVarsEnabled())
		})
	}
}

func TestSpec_IsMetricsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   ExporterConfiguration
		expected bool
	}{
		{
			name:     "metrics disabled",
			config:   ExporterConfiguration{},
			expected: false,
		},
		{
			name: "metrics enabled",
			config: ExporterConfiguration{
				Spec: &ExporterConfigurationSpec{
					Signals: SignalConfiguration{
						Metrics: &MetricsSignal{},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.IsMetricsEnabled())
		})
	}
}

func TestSpec_IsTracesEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   ExporterConfiguration
		expected bool
	}{
		{
			name:     "traces disabled",
			config:   ExporterConfiguration{},
			expected: false,
		},
		{
			name: "traces enabled",
			config: ExporterConfiguration{
				Spec: &ExporterConfigurationSpec{
					Signals: SignalConfiguration{
						Traces: &TracesSignal{},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.IsTracesEnabled())
		})
	}
}

func TestSpec_IsLogsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   ExporterConfiguration
		expected bool
	}{
		{
			name:     "logs disabled",
			config:   ExporterConfiguration{},
			expected: false,
		},
		{
			name: "logs enabled",
			config: ExporterConfiguration{
				Spec: &ExporterConfigurationSpec{
					Signals: SignalConfiguration{
						Logs: &LogsSignal{},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.IsLogsEnabled())
		})
	}
}

func TestSpec_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   ExporterConfiguration
		expected bool
	}{
		{
			name:     "disabled",
			config:   ExporterConfiguration{},
			expected: false,
		},
		{
			name:     "enabled (traces)",
			config:   ExporterConfiguration{Spec: &ExporterConfigurationSpec{Signals: SignalConfiguration{Traces: &TracesSignal{}}}},
			expected: true,
		},
		{
			name:     "enabled (metrics)",
			config:   ExporterConfiguration{Spec: &ExporterConfigurationSpec{Signals: SignalConfiguration{Metrics: &MetricsSignal{}}}},
			expected: true,
		},
		{
			name:     "enabled (logs)",
			config:   ExporterConfiguration{Spec: &ExporterConfigurationSpec{Signals: SignalConfiguration{Logs: &LogsSignal{}}}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.IsEnabled())
		})
	}
}

func TestExporterConfiguration_GetResourceAttributes(t *testing.T) {
	tests := []struct {
		name     string
		spec     *ExporterConfigurationSpec
		global   map[string]string
		expected map[string]string
	}{
		{
			name:     "nil spec - no global returns nil",
			spec:     nil,
			global:   nil,
			expected: nil,
		},
		{
			name:     "nil spec - only global returns global",
			spec:     nil,
			global:   map[string]string{"g": "1"},
			expected: map[string]string{"g": "1"},
		},
		{
			name:     "only global set, no additional",
			spec:     &ExporterConfigurationSpec{},
			global:   map[string]string{"a": "1"},
			expected: map[string]string{"a": "1"},
		},
		{
			name: "only additional set, no global",
			spec: &ExporterConfigurationSpec{
				AdditionalResourceAttributes: map[string]string{"x": "10"},
			},
			global:   nil,
			expected: map[string]string{"x": "10"},
		},
		{
			name: "non-overlapping global and additional returns union",
			spec: &ExporterConfigurationSpec{
				AdditionalResourceAttributes: map[string]string{"b": "2"},
			},
			global:   map[string]string{"a": "1"},
			expected: map[string]string{"a": "1", "b": "2"},
		},
		{
			name: "overlapping and non-overlapping: additional wins on overlap, union otherwise",
			spec: &ExporterConfigurationSpec{
				AdditionalResourceAttributes: map[string]string{"b": "2", "shared": "additional"},
			},
			global:   map[string]string{"a": "1", "shared": "global"},
			expected: map[string]string{"a": "1", "b": "2", "shared": "additional"},
		},
		{
			name:     "both nil/empty returns nil",
			spec:     &ExporterConfigurationSpec{},
			global:   nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewExporterConfiguration(tt.spec, tt.global)
			assert.Equal(t, tt.expected, config.GetResourceAttributes())
		})
	}
}
