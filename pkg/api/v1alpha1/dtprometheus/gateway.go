// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import appsv1 "k8s.io/api/apps/v1"

// GatewayNameSuffix is appended to the owning DTPrometheus name to derive the base
// name of the gateway's Kubernetes resources.
const GatewayNameSuffix = "-gateway"

// +kubebuilder:object:generate=false

// Gateway wraps the gateway Spec together with the owning DTPrometheus name so
// derived state (such as Kubernetes resource names) can be computed.
type Gateway struct {
	*GatewaySpec

	name string
}

// GatewaySpec configures the gateway pool StatefulSet (tier 2): a StatefulSet of
// OTel Collectors that run stateful processors and export to Dynatrace via OTLP/HTTP.
type GatewaySpec struct {
	PodSpec `json:",inline"`

	// StatefulSet update strategy for the gateway pool. partition: N means only
	// pods with ordinal >= N are updated; set a non-zero value for canary rollouts.
	// +kubebuilder:validation:Optional
	UpdateStrategy appsv1.StatefulSetUpdateStrategy `json:"updateStrategy,omitzero"`
}

// NewGateway wraps the given Spec together with the owning DTPrometheus name.
func NewGateway(spec *GatewaySpec, name string) *Gateway {
	return &Gateway{
		GatewaySpec: spec,
		name:        name,
	}
}

// SetName sets the owning DTPrometheus name.
func (g *Gateway) SetName(name string) {
	g.name = name
}

// GetStatefulSetName returns the base name for the gateway's StatefulSet.
func (g *Gateway) GetStatefulSetName() string {
	return g.name + GatewayNameSuffix
}
