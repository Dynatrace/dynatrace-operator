package activegate

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseCommunicationHostsFromActiveGateEndpoints(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "test-name",
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: dynakube.OneAgentStatus{
				ConnectionInfoStatus: dynakube.OneAgentConnectionInfoStatus{
					ConnectionInfo: communication.ConnectionInfo{},
				},
			},
		},
	}

	t.Run(`endpoints empty`, func(t *testing.T) {
		hosts := parseCommunicationHostFromActiveGateEndpoints("")
		assert.Empty(t, hosts)
	})

	t.Run(`activegate endpoint set`, func(t *testing.T) {
		dk.Status.ActiveGate.ConnectionInfo.Endpoints = "https://abcd123.some.activegate.endpointurl.com:443"

		hosts := GetEndpointsAsCommunicationHosts(dk)
		assert.Len(t, hosts, 1)
		assert.Equal(t, "abcd123.some.activegate.endpointurl.com", hosts[0].Host)
		assert.Equal(t, "https", hosts[0].Protocol)
		assert.Equal(t, uint32(443), hosts[0].Port)
	})
	t.Run(`activegate multiple endpoints set`, func(t *testing.T) {
		dk.Status.ActiveGate.ConnectionInfo.Endpoints = "https://abcd123.some.activegate.endpointurl.com:443,https://efg5678.some.other.activegate.endpointurl.com"

		hosts := GetEndpointsAsCommunicationHosts(dk)
		assert.Len(t, hosts, 2)
		hostNames := []string{hosts[0].Host, hosts[1].Host}
		assert.Contains(t, hostNames, "abcd123.some.activegate.endpointurl.com")
		assert.Contains(t, hostNames, "efg5678.some.other.activegate.endpointurl.com")
	})
	t.Run(`activegate duplicate endpoints set`, func(t *testing.T) {
		dk.Status.ActiveGate.ConnectionInfo.Endpoints = "https://abcd123.some.activegate.endpointurl.com:443,https://abcd123.some.activegate.endpointurl.com:443,https://abcd123.some.activegate.endpointurl.com:443"

		hosts := GetEndpointsAsCommunicationHosts(dk)
		assert.Len(t, hosts, 1)
		assert.Equal(t, "abcd123.some.activegate.endpointurl.com", hosts[0].Host)
		assert.Equal(t, "https", hosts[0].Protocol)
		assert.Equal(t, uint32(443), hosts[0].Port)
	})
}
