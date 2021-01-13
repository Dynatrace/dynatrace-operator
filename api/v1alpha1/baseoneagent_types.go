package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BaseOneAgentSpec struct {
	// If enabled, Istio on the cluster will be configured automatically to allow access to the Dynatrace environment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Enable Istio automatic management"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	EnableIstio bool `json:"enableIstio,omitempty"`

	// Defines if you want to use the immutable image or the installer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Use immutable image"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	UseImmutableImage bool `json:"useImmutableImage,omitempty"`
}

type BaseOneAgentStatus struct {
	// LastClusterVersionProbeTimestamp indicates when the cluster's version was last checked
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Last cluster version probed"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	LastClusterVersionProbeTimestamp *metav1.Time `json:"lastClusterVersionProbeTimestamp,omitempty"`

	// UseImmutableImage is set when an immutable image is currently in use
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Using immutable image"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	UseImmutableImage bool `json:"useImmutableImage,omitempty"`

	// EnvironmentID contains the environment ID corresponding to the API URL
	EnvironmentID string `json:"environmentID,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

const (
	// APITokenConditionType identifies the API Token validity condition
	APITokenConditionType string = "APIToken"

	// PaaSTokenConditionType identifies the PaaS Token validity condition
	PaaSTokenConditionType string = "PaaSToken"
)

// Possible reasons for ApiToken and PaaSToken conditions
const (
	// ReasonTokenReady is set when a token has passed verifications
	ReasonTokenReady string = "TokenReady"

	// ReasonTokenSecretNotFound is set when the referenced secret can't be found
	ReasonTokenSecretNotFound string = "TokenSecretNotFound"

	// ReasonTokenMissing is set when the field is missing on the secret
	ReasonTokenMissing string = "TokenMissing"

	// ReasonTokenUnauthorized is set when a token is unauthorized to query the Dynatrace API
	ReasonTokenUnauthorized string = "TokenUnauthorized"

	// ReasonTokenScopeMissing is set when the token is missing the required scope for the Dynatrace API
	ReasonTokenScopeMissing string = "TokenScopeMissing"

	// ReasonTokenError is set when an unknown error has been found when verifying the token
	ReasonTokenError string = "TokenError"
)
