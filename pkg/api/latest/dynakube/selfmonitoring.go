// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"net"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/sanitize"
)

const (
	selfMonitoringNameSuffix = "-self-monitoring"

	// SelfMonitoringOTLPHTTPPort is the port of the self-monitoring collector's OTLP/HTTP receiver.
	SelfMonitoringOTLPHTTPPort = 4318
)

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

// SelfMonitoringOTLPEndpoint is the base URL of the self-monitoring collector's OTLP/HTTP receiver,
// to be used by other components to export their own telemetry.
func (dk *DynaKube) SelfMonitoringOTLPEndpoint() string {
	return "http://" + net.JoinHostPort(dk.SelfMonitoringName(), strconv.Itoa(SelfMonitoringOTLPHTTPPort))
}
