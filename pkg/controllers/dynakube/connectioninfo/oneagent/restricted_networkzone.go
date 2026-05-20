package oaconnectioninfo

import (
	"net/url"
	"slices"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
)

// restrictedNetworkZoneEndpointCount is the number of OneAgent endpoints the Dynatrace
// cluster returns when the tenant lives in a restricted network-zone served only by a
// single local ActiveGate: one IP-based endpoint plus the AG Service's DNS entry point.
//
// The ActiveGate's Service is a ClusterIP service, with a default(unset) ipFamilyPolicy which is SingleStack.
// If we used DualStack, then it would be possible to have 2 IP-based endpoints (ipv4/ipv6).
// This logic/const expects that we are always using a SingleStack Service.
const restrictedNetworkZoneEndpointCount = 2

// hasStaleRestrictedNetworkZoneEndpoints performs a best-effort detection of a restricted
// network-zone setup served by the local ActiveGate Service.
//
// The Dynatrace cluster decides whether a tenant lives in a restricted network-zone; the
// Operator cannot know for certain. We approximate the setup by checking whether the
// returned endpoint list contains exactly two entries and one of them points at the local
// AG Service's DNS name (https://<dk-name>-activegate.<namespace>:<port>/communication).
// When that pattern matches, the other endpoint should be the AG Service's current
// ClusterIP. If it is not, the cluster is still advertising a stale IP (typically right
// after the AG Service IP changed and before the AG has re-registered) and propagating
// those endpoints to the OneAgent would leave it unable to reach the AG.
//
// Returns true only when the setup is detected AND the IP endpoint is stale. The check
// is intentionally conservative – if any inputs are missing or unexpected, the function
// returns false so the non-restricted scenarios are not affected.
func hasStaleRestrictedNetworkZoneEndpoints(dk *dynakube.DynaKube, endpoints string) bool {
	if dk == nil || dk.Spec.NetworkZone == "" || !dk.ActiveGate().IsRoutingEnabled() || len(dk.Status.ActiveGate.ServiceIPs) == 0 {
		return false
	}

	hosts := parseEndpointHosts(endpoints)
	if len(hosts) != restrictedNetworkZoneEndpointCount {
		return false
	}

	localServiceHost := capability.BuildServiceHostname(*dk)

	var (
		ipHost              string
		hasLocalServiceName bool
	)

	for _, host := range hosts {
		if host == localServiceHost {
			hasLocalServiceName = true
		} else {
			ipHost = host
		}
	}

	// This is the "best effort" part.
	// if it doesn't contain our expected local AG Service host, then it could be a different AG, local or not.
	// Therefore, we can't expect that the `ipHost` is the local AG Service's IP.
	//
	// These endpoints maybe from a different DynaKube in the cluster using the same network-zone, it could be anything.
	if !hasLocalServiceName || ipHost == "" {
		return false
	}

	return !slices.Contains(dk.Status.ActiveGate.ServiceIPs, ipHost)
}

// parseEndpointHosts splits the cluster-provided comma-separated endpoint string and
// returns the host portion (without port) of each parseable entry. Unparseable entries
// are silently skipped so the best-effort check does not over-trigger on unexpected
// formats.
func parseEndpointHosts(endpoints string) []string {
	if endpoints == "" {
		return nil
	}

	hosts := make([]string, 0)

	for endpoint := range strings.SplitSeq(endpoints, ",") {
		u, err := url.Parse(endpoint)
		if err != nil {
			continue
		}

		host := u.Hostname()
		if host == "" {
			continue
		}

		hosts = append(hosts, host)
	}

	return hosts
}
