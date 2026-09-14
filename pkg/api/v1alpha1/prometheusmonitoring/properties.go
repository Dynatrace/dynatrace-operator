// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package prometheusmonitoring

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Conditions returns a pointer to the status conditions slice so callers can
// use meta.SetStatusCondition and friends.
func (pm *PrometheusMonitoring) Conditions() *[]metav1.Condition {
	return &pm.Status.Conditions
}

// TargetAllocator returns the Target Allocator accessor wrapping the target
// allocator spec and the owning PrometheusMonitoring name.
func (pm *PrometheusMonitoring) TargetAllocator() *TargetAllocator {
	return NewTargetAllocator(&pm.Spec.TargetAllocator, pm.Name)
}

// Scraper returns the scraper pool accessor wrapping the scraper spec and the
// owning PrometheusMonitoring name.
func (pm *PrometheusMonitoring) Scraper() *Scraper {
	return NewScraper(&pm.Spec.Scraper, pm.Name)
}

// Gateway returns the gateway pool accessor wrapping the gateway spec and the
// owning PrometheusMonitoring name.
func (pm *PrometheusMonitoring) Gateway() *Gateway {
	return NewGateway(&pm.Spec.Gateway, pm.Name)
}
