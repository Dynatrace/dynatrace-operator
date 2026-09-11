// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrometheusMonitoringStatus defines the observed state of PrometheusMonitoring.
type PrometheusMonitoringStatus struct { //nolint:revive
	// Defines the current state (Running, Deploying, Error, ...)
	Phase status.DeploymentPhase `json:"phase,omitempty"`

	Gateway         GatewayStatus         `json:"gateway,omitempty"`
	Scraper         ScraperStatus         `json:"scraper,omitempty"`
	TargetAllocator TargetAllocatorStatus `json:"targetAllocator,omitempty"`

	// Conditions includes status about the current state of the instance
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type GatewayStatus struct {
	// Image URI of the gateway currently deployed.
	ResolvedImage string `json:"image,omitempty"`
}

type ScraperStatus struct {
	// Image URI of the scraper currently deployed.
	ResolvedImage string `json:"image,omitempty"`
}

type TargetAllocatorStatus struct {
	// Image URI of the target allocator currently deployed.
	ResolvedImage string `json:"image,omitempty"`
}

// SetPhase sets the status phase on the PrometheusMonitoring object.
func (pms *PrometheusMonitoringStatus) SetPhase(phase status.DeploymentPhase) bool {
	upd := phase != pms.Phase
	pms.Phase = phase

	return upd
}
