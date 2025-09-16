package otlpexporterconfiguration

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:generate=true
type Spec struct {
	// The namespaces where you want Dynatrace Operator to inject OTLP exporter configuration.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace Selector",xDescriptors="urn:alm:descriptor:com.tectonic.ui:selector:core:v1:Namespace"
	NamespaceSelector metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// The OpenTelemetry signals that should be configured to be sent via OTLP
	Signals SignalConfiguration `json:"signals,omitempty"`

	// When enabled, existing environment variables for the configuration of the OTLP exporter will be overridden.
	// Disabled by default.
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable override of existing OTLP environment variables",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	OverrideEnvVars *bool `json:"overrideEnvVars,omitempty"`
}

// +kubebuilder:object:generate=true
type SignalConfiguration struct {
	// Enables the automatic OTLP exporter configuration for Metrics
	// see https://opentelemetry.io/docs/specs/otel/protocol/exporter/#endpoint-urls-for-otlphttp
	Metrics *MetricsConfiguration `json:"metrics,omitempty"`
	// Enables the automatic OTLP exporter configuration for Traces
	// see https://opentelemetry.io/docs/specs/otel/protocol/exporter/#endpoint-urls-for-otlphttp
	Traces *TracesConfiguration `json:"traces,omitempty"`
	// Enables the automatic OTLP exporter configuration for Logs
	// see https://opentelemetry.io/docs/specs/otel/protocol/exporter/#endpoint-urls-for-otlphttp
	Logs *LogsConfiguration `json:"logs,omitempty"`
}

// +kubebuilder:object:generate=true
type MetricsConfiguration struct{}

// +kubebuilder:object:generate=true
type TracesConfiguration struct{}

// +kubebuilder:object:generate=true
type LogsConfiguration struct{}
