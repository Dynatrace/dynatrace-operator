package oaconnectioninfo

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHasStaleRestrictedNetworkZoneEndpoints(t *testing.T) {
	const (
		dkName    = "test-dk"
		namespace = "dynatrace"
		// Matches BuildServiceName(dkName) + "." + namespace
		localServiceHost = "test-dk-activegate.dynatrace"
	)

	newDynaKubeWithAG := func(serviceIPs []string) *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
			Spec: dynakube.DynaKubeSpec{
				NetworkZone: "restricted-zone",
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName},
				},
			},
			Status: dynakube.DynaKubeStatus{
				ActiveGate: activegate.Status{
					ServiceIPs: serviceIPs,
				},
			},
		}
	}

	t.Run("nil DynaKube → not stale", func(t *testing.T) {
		assert.False(t, hasStaleRestrictedNetworkZoneEndpoints(nil, "anything"))
	})

	t.Run("ActiveGate disabled → not stale", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
			Spec:       dynakube.DynaKubeSpec{NetworkZone: "restricted-zone"},
			Status: dynakube.DynaKubeStatus{
				ActiveGate: activegate.Status{ServiceIPs: []string{"10.0.0.1"}},
			},
		}

		endpoints := "https://10.0.0.1:443/communication,https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleRestrictedNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("no NetworkZone configured → not stale (best effort skip)", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.2"})
		dk.Spec.NetworkZone = ""

		// Stale IP that would normally trigger; gate must suppress it because no
		// network zone is configured.
		endpoints := "https://10.0.0.1:443/communication,https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleRestrictedNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("no AG ServiceIPs → not stale (best effort skip)", func(t *testing.T) {
		dk := newDynaKubeWithAG(nil)
		endpoints := "https://10.0.0.1:443/communication,https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleRestrictedNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("empty endpoints → not stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1"})
		assert.False(t, hasStaleRestrictedNetworkZoneEndpoints(dk, ""))
	})

	t.Run("single endpoint → not restricted network-zone setup", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1"})
		endpoints := "https://my.live.dynatrace.com:443/communication"
		assert.False(t, hasStaleRestrictedNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("three endpoints → not restricted network-zone setup", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1"})
		endpoints := "https://1.1.1.1:443/communication,https://2.2.2.2:443/communication,https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleRestrictedNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("two endpoints but neither is the local AG service → not detected", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1"})
		endpoints := "https://other-activegate.dynatrace:443/communication,https://1.2.3.4:443/communication"
		assert.False(t, hasStaleRestrictedNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("restricted setup with matching IP → not stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1"})
		endpoints := "https://10.0.0.1:443/communication,https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleRestrictedNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("restricted setup with matching IPv6 → not stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"2001:db8::1"})
		endpoints := "https://[2001:db8::1]:443/communication,https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleRestrictedNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("restricted setup, IP changed since AG was deployed → stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.2"})
		endpoints := "https://10.0.0.1:443/communication,https://" + localServiceHost + ":443/communication"
		assert.True(t, hasStaleRestrictedNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("restricted setup with multiple cluster IPs, one matches → not stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1", "2001:db8::1"})
		endpoints := "https://10.0.0.1:443/communication,https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleRestrictedNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("unparseable endpoint entries do not over-trigger", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1"})
		// Three comma-separated entries, only two parseable → behaves as 2-entry list with both parseables;
		// here the parseable two happen to match the restricted setup with the stale IP.
		endpoints := "garbage,https://10.0.0.2:443/communication,https://" + localServiceHost + ":443/communication"
		assert.True(t, hasStaleRestrictedNetworkZoneEndpoints(dk, endpoints))
	})
}
