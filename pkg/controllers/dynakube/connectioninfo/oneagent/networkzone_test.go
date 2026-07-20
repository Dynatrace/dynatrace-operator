package oaconnectioninfo

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHasStaleNetworkZoneEndpoints(t *testing.T) {
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
		assert.False(t, hasStaleNetworkZoneEndpoints(nil, "anything"))
	})

	t.Run("ActiveGate disabled → not stale", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dkName, Namespace: namespace},
			Spec:       dynakube.DynaKubeSpec{NetworkZone: "restricted-zone"},
			Status: dynakube.DynaKubeStatus{
				ActiveGate: activegate.Status{ServiceIPs: []string{"10.0.0.1"}},
			},
		}

		endpoints := "https://10.0.0.1:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("no NetworkZone configured → not stale (best effort skip)", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.2"})
		dk.Spec.NetworkZone = ""

		// Stale IP that would normally trigger; gate must suppress it because no
		// network zone is configured.
		endpoints := "https://10.0.0.1:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("no AG ServiceIPs → not stale (best effort skip)", func(t *testing.T) {
		dk := newDynaKubeWithAG(nil)
		endpoints := "https://10.0.0.1:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("empty endpoints → stale (no ServiceIP can be present)", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1"})
		assert.True(t, hasStaleNetworkZoneEndpoints(dk, ""))
	})

	t.Run("ServiceIP present alongside local AG DNS endpoint → not stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1"})
		endpoints := "https://10.0.0.1:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("ServiceIP present alongside unrelated endpoints → not stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1"})
		endpoints := "https://1.2.3.4:443/communication;https://10.0.0.1:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("IPv6 ServiceIP present (bracketed in endpoint URL) → not stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"2001:db8::1"})
		endpoints := "https://[2001:db8::1]:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("ServiceIP missing from endpoints → stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.2"})
		endpoints := "https://10.0.0.1:443/communication;https://" + localServiceHost + ":443/communication"
		assert.True(t, hasStaleNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("endpoints contain no IP-based entries at all → stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1"})
		endpoints := "https://other-activegate.dynatrace:443/communication;https://" + localServiceHost + ":443/communication"
		assert.True(t, hasStaleNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("dual-stack: all ServiceIPs present → not stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1", "2001:db8::1"})
		endpoints := "https://10.0.0.1:443/communication;https://[2001:db8::1]:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("dual-stack: one ServiceIP missing → stale", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1", "2001:db8::1"})
		// Cluster still advertises the previous IPv6 address.
		endpoints := "https://10.0.0.1:443/communication;https://[2001:db8::2]:443/communication;https://" + localServiceHost + ":443/communication"
		assert.True(t, hasStaleNetworkZoneEndpoints(dk, endpoints))
	})

	t.Run("unparseable endpoint string → not stale (best effort skip)", func(t *testing.T) {
		dk := newDynaKubeWithAG([]string{"10.0.0.1"})
		// "garbage" makes NewOACommunicationHosts return an error; the function defers
		// rather than blocking deployment on an input shape it cannot reason about.
		endpoints := "garbage;https://10.0.0.1:443/communication;https://" + localServiceHost + ":443/communication"
		assert.False(t, hasStaleNetworkZoneEndpoints(dk, endpoints))
	})
}
