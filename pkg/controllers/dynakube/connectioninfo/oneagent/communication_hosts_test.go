package oaconnectioninfo

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetCommunicationHosts(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: oneagent.Status{
				ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
					ConnectionInfo: communication.ConnectionInfo{},
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

	t.Run("communications host empty", func(t *testing.T) {
		hosts := GetCommunicationHosts(dk)
		assert.Empty(t, hosts)
	})

	t.Run("communication-hosts field found", func(t *testing.T) {
		dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = []oneagent.CommunicationHostStatus{
			{
				Protocol: "protocol",
				Host:     "host",
				Port:     12345,
			},
		}

		hosts := GetCommunicationHosts(dk)
		assert.NotNil(t, hosts)
		assert.Equal(t, expectedCommunicationHosts[0].Host, hosts[0].Host)
		assert.Equal(t, expectedCommunicationHosts[0].Protocol, hosts[0].Protocol)
		assert.Equal(t, expectedCommunicationHosts[0].Port, hosts[0].Port)
	})
}
