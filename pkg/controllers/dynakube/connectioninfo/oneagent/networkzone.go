// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oaconnectioninfo

import (
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
)

// hasStaleNetworkZoneEndpoints checks whether the OneAgent endpoints returned by the
// Dynatrace cluster still advertise every ClusterIP of the local routing ActiveGate
// Service.
//
// When the DynaKube uses a network-zone served by a routing ActiveGate, the cluster is
// expected to return endpoints that point at this AG. If any of the AG Service's current
// ClusterIPs is missing from the returned endpoint list, the cluster is still
// advertising a stale IP (typically right after the AG Service IPs changed and before
// the AG has re-registered, which can happen if the DynaKube was removed then reapplied)
// and propagating those endpoints to the OneAgent would leave it unable to reach the AG.
// The OneAgent deployment is postponed until the cluster reissues correct endpoints.
//
// The function does NOT try to detect whether the network-zone is actually "restricted"
// (i.e. the cluster suppresses all other endpoints); the cluster owns that decision and
// the Operator cannot verify it. It only enforces what it can: every local AG Service
// IP must appear in the endpoint list if a network-zone is used. In every other case
// the function returns false so the non-network-zone scenarios are not affected. The
// same applies if the endpoint string cannot be parsed at all – we treat that as an
// unrecognized shape and defer to the rest of the system.
//
// Note: the ClusterIP Service we create for the ActiveGate should have a single
// ClusterIP, because we do not set spec.ipFamilyPolicy and that defaults to
// SingleStack. If for some reason the Service ended up DualStack, the ActiveGate
// would register two ClusterIPs, one per IP family, and this function handles
// that unrealistic scenario as well.
func hasStaleNetworkZoneEndpoints(dk *dynakube.DynaKube, endpoints string) bool {
	if dk == nil || dk.Spec.NetworkZone == "" || !dk.ActiveGate().IsRoutingEnabled() || len(dk.Status.ActiveGate.ServiceIPs) == 0 {
		return false
	}

	hosts, err := connectioninfo.ParseOACommunicationHosts(endpoints)
	if err != nil {
		return false
	}

	for _, ip := range dk.Status.ActiveGate.ServiceIPs {
		if !slices.ContainsFunc(hosts, func(h connectioninfo.CommunicationHost) bool { return h.Host == ip }) {
			return true
		}
	}

	return false
}
