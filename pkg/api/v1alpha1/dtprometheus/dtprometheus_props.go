// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsDynatracePresetEnabled reports whether the built-in annotation-based
// ScrapeConfig should be created. Disabled unless the section is explicitly present.
func (dtp *DtPrometheus) IsDynatracePresetEnabled() bool {
	return dtp.Spec.DynatracePreset != nil
}

// IsTLSEnabled reports whether operator-managed TLS between components is
// enabled. The TLS section defaults to an empty object (enabled) when omitted;
// it is disabled only when explicitly set to null.
func (dtp *DtPrometheus) IsTLSEnabled() bool {
	return dtp.Spec.TLS != nil
}

// IsSelfMonitoringEnabled reports whether the self-monitoring collector should be
// deployed. The self-monitoring section defaults to an empty object (enabled)
// when omitted; it is disabled only when explicitly set to null.
func (dtp *DtPrometheus) IsSelfMonitoringEnabled() bool {
	return dtp.Spec.SelfMonitoring != nil
}

// Conditions returns a pointer to the status conditions slice so callers can
// use meta.SetStatusCondition and friends.
func (dtp *DtPrometheus) Conditions() *[]metav1.Condition {
	return &dtp.Status.Conditions
}

// TargetAllocator returns the Target Allocator accessor wrapping the target
// allocator spec and the owning DtPrometheus name.
func (dtp *DtPrometheus) TargetAllocator() *TargetAllocator {
	return NewTargetAllocator(&dtp.Spec.TargetAllocator, dtp.Name)
}

// Scraper returns the scraper pool accessor wrapping the scraper spec and the
// owning DtPrometheus name.
func (dtp *DtPrometheus) Scraper() *Scraper {
	return NewScraper(&dtp.Spec.Scraper, dtp.Name)
}

// Gateway returns the gateway pool accessor wrapping the gateway spec and the
// owning DtPrometheus name.
func (dtp *DtPrometheus) Gateway() *Gateway {
	return NewGateway(&dtp.Spec.Gateway, dtp.Name)
}
