// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import "github.com/Dynatrace/dynatrace-operator/pkg/util/sanitize"

const selfMonitoringNameSuffix = "-self-monitoring"

// IsSelfMonitoringEnabled reports whether the self-monitoring collector was opted in via .spec.selfMonitoring.
func (dk *DynaKube) IsSelfMonitoringEnabled() bool {
	return dk.Spec.SelfMonitoring != nil
}

// SelfMonitoringName is the name shared by all objects that make up the self-monitoring collector.
func (dk *DynaKube) SelfMonitoringName() string {
	return dk.Name + selfMonitoringNameSuffix
}

func (s *SelfMonitoringSpec) SanitizedArgs() []string {
	return sanitize.CommandLineArgs(s.Args)
}
