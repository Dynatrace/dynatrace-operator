// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus/gateway"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus/scraper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus/selfmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus/targetallocator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetDynaKubeName returns the name of the referenced DynaKube.
func (dtp *DtPrometheus) GetDynaKubeName() string {
	return dtp.Spec.DynaKubeName
}

// IsDynatracePresetEnabled reports whether the built-in annotation-based
// ScrapeConfig should be created.
func (dtp *DtPrometheus) IsDynatracePresetEnabled() bool {
	return dtp.Spec.DynatracePreset.Enabled
}

// IsTLSEnabled reports whether operator-managed TLS between components is enabled.
func (dtp *DtPrometheus) IsTLSEnabled() bool {
	return dtp.Spec.TLS.Enabled
}

// GetTLSSecret returns the name of the user-provided TLS secret, or an empty
// string when the operator should generate a self-signed certificate.
func (dtp *DtPrometheus) GetTLSSecret() string {
	return dtp.Spec.TLS.Secret
}

// Conditions returns a pointer to the status conditions slice so callers can
// use meta.SetStatusCondition and friends.
func (dtp *DtPrometheus) Conditions() *[]metav1.Condition {
	return &dtp.Status.Conditions
}

// TargetAllocator returns the Target Allocator accessor wrapping the target
// allocator spec and the owning DtPrometheus name.
func (dtp *DtPrometheus) TargetAllocator() *targetallocator.TargetAllocator {
	return targetallocator.New(&dtp.Spec.TargetAllocator, dtp.Name)
}

// Scraper returns the scraper pool accessor wrapping the scraper spec and the
// owning DtPrometheus name.
func (dtp *DtPrometheus) Scraper() *scraper.Scraper {
	return scraper.New(&dtp.Spec.Scraper, dtp.Name)
}

// Gateway returns the gateway pool accessor wrapping the gateway spec and the
// owning DtPrometheus name.
func (dtp *DtPrometheus) Gateway() *gateway.Gateway {
	return gateway.New(&dtp.Spec.Gateway, dtp.Name)
}

// SelfMonitoring returns the self-monitoring accessor wrapping the self-monitoring
// spec and the owning DtPrometheus name.
func (dtp *DtPrometheus) SelfMonitoring() *selfmonitoring.SelfMonitoring {
	return selfmonitoring.New(&dtp.Spec.SelfMonitoring, dtp.Name)
}
