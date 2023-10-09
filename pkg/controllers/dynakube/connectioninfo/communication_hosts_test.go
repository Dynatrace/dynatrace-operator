package connectioninfo

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetCommunicationHosts(t *testing.T) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			OneAgent: dynatracev1beta1.OneAgentStatus{
				ConnectionInfoStatus: dynatracev1beta1.OneAgentConnectionInfoStatus{
					ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{},
				},
			},
		},
	}

	expectedCommunicationHosts := []dynatrace.CommunicationHost{
		{
			Protocol: "protocol",
			Host:     "host",
			Port:     12345,
		},
	}

	t.Run(`communications host empty`, func(t *testing.T) {
		hosts := GetOneAgentCommunicationHosts(dynakube)
		assert.Len(t, hosts, 0)
	})

	t.Run(`communication-hosts field found`, func(t *testing.T) {
		dynakube.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = []dynatracev1beta1.CommunicationHostStatus{
			{
				Protocol: "protocol",
				Host:     "host",
				Port:     12345,
			},
		}

		hosts := GetOneAgentCommunicationHosts(dynakube)
		assert.NotNil(t, hosts)
		assert.Equal(t, expectedCommunicationHosts[0].Host, hosts[0].Host)
		assert.Equal(t, expectedCommunicationHosts[0].Protocol, hosts[0].Protocol)
		assert.Equal(t, expectedCommunicationHosts[0].Port, hosts[0].Port)
	})
}

func TestParseCommunicationHostsFromActiveGateEndpoints(t *testing.T) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			OneAgent: dynatracev1beta1.OneAgentStatus{
				ConnectionInfoStatus: dynatracev1beta1.OneAgentConnectionInfoStatus{
					ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{},
				},
			},
		},
	}

	t.Run(`endpoints empty`, func(t *testing.T) {
		hosts := parseCommunicationHostFromActiveGateEndpoints("")
		assert.Len(t, hosts, 0)
	})

	t.Run(`activegate endpoint set`, func(t *testing.T) {
		dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints = "https://abcd123.some.activegate.endpointurl.com:443"

		hosts := GetActiveGateEndpointsAsCommunicationHosts(dynakube)
		assert.Equal(t, 1, len(hosts))
		assert.Equal(t, "abcd123.some.activegate.endpointurl.com", hosts[0].Host)
		assert.Equal(t, "https", hosts[0].Protocol)
		assert.Equal(t, uint32(443), hosts[0].Port)
	})
	t.Run(`activegate multiple endpoints set`, func(t *testing.T) {
		dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints = "https://abcd123.some.activegate.endpointurl.com:443,https://efg5678.some.other.activegate.endpointurl.com"

		hosts := GetActiveGateEndpointsAsCommunicationHosts(dynakube)
		assert.Equal(t, 2, len(hosts))
		hostNames := []string{hosts[0].Host, hosts[1].Host}
		assert.Contains(t, hostNames, "abcd123.some.activegate.endpointurl.com")
		assert.Contains(t, hostNames, "efg5678.some.other.activegate.endpointurl.com")
	})
	t.Run(`activegate duplicate endpoints set`, func(t *testing.T) {
		dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints = "https://abcd123.some.activegate.endpointurl.com:443,https://abcd123.some.activegate.endpointurl.com:443,https://abcd123.some.activegate.endpointurl.com:443"

		hosts := GetActiveGateEndpointsAsCommunicationHosts(dynakube)
		assert.Equal(t, 1, len(hosts))
		assert.Equal(t, "abcd123.some.activegate.endpointurl.com", hosts[0].Host)
		assert.Equal(t, "https", hosts[0].Protocol)
		assert.Equal(t, uint32(443), hosts[0].Port)
	})
}
