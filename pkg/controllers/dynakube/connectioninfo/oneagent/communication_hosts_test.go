package oaconnectioninfo

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
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

	expectedCommunicationHosts := []dtclient.CommunicationHost{
		{
			Protocol: "protocol",
			Host:     "host",
			Port:     12345,
		},
	}

	t.Run(`communications host empty`, func(t *testing.T) {
		hosts := GetCommunicationHosts(dynakube)
		assert.Empty(t, hosts)
	})

	t.Run(`communication-hosts field found`, func(t *testing.T) {
		dynakube.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = []dynatracev1beta1.CommunicationHostStatus{
			{
				Protocol: "protocol",
				Host:     "host",
				Port:     12345,
			},
		}

		hosts := GetCommunicationHosts(dynakube)
		assert.NotNil(t, hosts)
		assert.Equal(t, expectedCommunicationHosts[0].Host, hosts[0].Host)
		assert.Equal(t, expectedCommunicationHosts[0].Protocol, hosts[0].Protocol)
		assert.Equal(t, expectedCommunicationHosts[0].Port, hosts[0].Port)
	})
}
