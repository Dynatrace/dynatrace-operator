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
	"k8s.io/utils/ptr"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpec_IsOverrideEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   OTLPExporterConfiguration
		expected bool
	}{
		{
			name:     "overrideEnvVars not set",
			config:   OTLPExporterConfiguration{},
			expected: false,
		},
		{
			name: "overrideEnvVars disabled",
			config: OTLPExporterConfiguration{Spec: &Spec{
				OverrideEnvVars: ptr.To(false),
			}},
			expected: false,
		},
		{
			name: "overrideEnvVars enabled",
			config: OTLPExporterConfiguration{Spec: &Spec{
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
		config   OTLPExporterConfiguration
		expected bool
	}{
		{
			name:     "metrics disabled",
			config:   OTLPExporterConfiguration{},
			expected: false,
		},
		{
			name: "metrics enabled",
			config: OTLPExporterConfiguration{
				Spec: &Spec{
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
		config   OTLPExporterConfiguration
		expected bool
	}{
		{
			name:     "traces disabled",
			config:   OTLPExporterConfiguration{},
			expected: false,
		},
		{
			name: "traces enabled",
			config: OTLPExporterConfiguration{
				Spec: &Spec{
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
		config   OTLPExporterConfiguration
		expected bool
	}{
		{
			name:     "logs disabled",
			config:   OTLPExporterConfiguration{},
			expected: false,
		},
		{
			name: "logs enabled",
			config: OTLPExporterConfiguration{
				Spec: &Spec{
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
		config   OTLPExporterConfiguration
		expected bool
	}{
		{
			name:     "disabled",
			config:   OTLPExporterConfiguration{},
			expected: false,
		},
		{
			name:     "enabled (traces)",
			config:   OTLPExporterConfiguration{Spec: &Spec{Signals: SignalConfiguration{Traces: &TracesSignal{}}}},
			expected: true,
		},
		{
			name:     "enabled (metrics)",
			config:   OTLPExporterConfiguration{Spec: &Spec{Signals: SignalConfiguration{Metrics: &MetricsSignal{}}}},
			expected: true,
		},
		{
			name:     "enabled (logs)",
			config:   OTLPExporterConfiguration{Spec: &Spec{Signals: SignalConfiguration{Logs: &LogsSignal{}}}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.IsEnabled())
		})
	}
}
