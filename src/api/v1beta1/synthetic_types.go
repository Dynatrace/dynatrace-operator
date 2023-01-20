package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

// properties to deploy Synthetic Monitoring
type SyntheticSpec struct {
	// private synthetic location
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Location Entity Id",order=1,xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	LocationEntityId string `json:"locationEntityId"`

	// synthetic node type
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Type",order=2,xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	NodeType string `json:"nodeType,omitempty"`

	// optional: environment variables for Synthetic Engine
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Environment variables",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Env []corev1.EnvVar `json:"env,omitempty"`

	DynaMetrics DynaMetricSpec `json:"dynaMetric,omitempty"`
	Autoscaler  AutoscalerSpec `json:"autoscaler,omitempty"`
}

// to publish metrics from Dynatrace Observability
type DynaMetricSpec struct {
	// credentials to query the metrics
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DynaMetric Token",order=20,xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	Token string `json:"token"`

	// optional: environment variables for External Metric Server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Environment variables",order=21,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:hidden"}
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// to accommodate pods with accordance to the load observed for Synthetic Engine
type AutoscalerSpec struct {
	// lower bound for replicas
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Min Replicas",order=30,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
	MinReplicas int32 `json:"minReplicas,omitempty"`

	// upper bound for replicas
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Max Replicas",order=31,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
	MaxReplicas int32 `json:"maxReplicas,omitempty"`

	// query to collect the load metric from Dynatrace Tenant
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Max Replicas",order=32,xDescriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
	DynaQuery string `json:"dynaQuery,omitempty"`
}
