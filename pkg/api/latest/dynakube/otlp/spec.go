// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otlp

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ExporterConfiguration struct {
	Spec                     *ExporterConfigurationSpec
	globalResourceAttributes map[string]string
}

// +kubebuilder:object:generate=true

type ExporterConfigurationSpec struct {

	// The OpenTelemetry signals that should be configured to be sent via OTLP
	Signals SignalConfiguration `json:"signals,omitempty"`

	// When enabled, existing environment variables for the configuration of the OTLP exporter will be overridden.
	// Disabled by default.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable override of existing OTLP environment variables",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	OverrideEnvVars *bool `json:"overrideEnvVars,omitempty"`
	// The namespaces where you want Dynatrace Operator to inject OTLP exporter configuration.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace Selector",xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:core:v1:Namespace"
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// Additional resource attributes that are merged on top of the global spec.resourceAttributes.
	// If the same key exists in both, the value from additionalResourceAttributes takes precedence.
	// +kubebuilder:validation:Optional
	AdditionalResourceAttributes map[string]string `json:"additionalResourceAttributes,omitempty"`
}

// +kubebuilder:object:generate=true

type SignalConfiguration struct {
	// Enables the automatic OTLP exporter configuration for Metrics
	// see https://opentelemetry.io/docs/specs/otel/protocol/exporter/#endpoint-urls-for-otlphttp
	Metrics *MetricsSignal `json:"metrics,omitempty"`
	// Enables the automatic OTLP exporter configuration for Traces
	// see https://opentelemetry.io/docs/specs/otel/protocol/exporter/#endpoint-urls-for-otlphttp
	Traces *TracesSignal `json:"traces,omitempty"`
	// Enables the automatic OTLP exporter configuration for Logs
	// see https://opentelemetry.io/docs/specs/otel/protocol/exporter/#endpoint-urls-for-otlphttp
	Logs *LogsSignal `json:"logs,omitempty"`
}

// +kubebuilder:object:generate=true

type MetricsSignal struct{}

// +kubebuilder:object:generate=true

type TracesSignal struct{}

// +kubebuilder:object:generate=true

type LogsSignal struct{}
