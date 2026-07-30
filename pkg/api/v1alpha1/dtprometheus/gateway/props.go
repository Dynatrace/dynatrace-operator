// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway

import appsv1 "k8s.io/api/apps/v1"

// NameSuffix is appended to the owning DtPrometheus name to derive the base name
// of the gateway's Kubernetes resources.
const NameSuffix = "-gateway"

// New wraps the given Spec together with the owning DtPrometheus name.
func New(spec *Spec, name string) *Gateway {
	return &Gateway{
		Spec: spec,
		name: name,
	}
}

// SetName sets the owning DtPrometheus name.
func (g *Gateway) SetName(name string) {
	g.name = name
}

// GetName returns the base name for the gateway's Kubernetes resources.
func (g *Gateway) GetName() string {
	return g.name + NameSuffix
}

// GetUpdateStrategy returns the StatefulSet update strategy for the gateway pool.
func (g *Gateway) GetUpdateStrategy() appsv1.StatefulSetUpdateStrategy {
	return g.UpdateStrategy
}
