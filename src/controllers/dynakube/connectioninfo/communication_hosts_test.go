package connectioninfo

import (
	"context"
	"encoding/json"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetCommunicationHosts(t *testing.T) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		}}

	communicationHosts := []dtclient.CommunicationHost{
		{
			Protocol: "protocol",
			Host:     "host",
			Port:     12345,
		},
	}

	t.Run(`connectioninfo config map not found`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()

		hosts, err := GetCommunicationHosts(context.TODO(), fakeClient, fakeClient, dynakube)
		assert.Error(t, err)
		assert.Nil(t, hosts)
	})

	t.Run(`communication-hosts field not found`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dynakube.OneAgentConnectionInfoConfigMapName(),
				Namespace: testNamespace,
			}}).Build()

		hosts, err := GetCommunicationHosts(context.TODO(), fakeClient, fakeClient, dynakube)
		assert.Error(t, err)
		assert.Nil(t, hosts)
	})

	t.Run(`communication-hosts field found`, func(t *testing.T) {
		connectionInfoConfigMap, err := buildConnectionInfoConfigMap(dynakube, communicationHosts)
		require.NoError(t, err, "failed to create connectioninfo config map")
		fakeClient := fake.NewClientBuilder().WithObjects(connectionInfoConfigMap).Build()

		hosts, err := GetCommunicationHosts(context.TODO(), fakeClient, fakeClient, dynakube)
		assert.NoError(t, err)
		assert.NotNil(t, hosts)
		assert.Equal(t, communicationHosts, hosts)
	})
}

func buildConnectionInfoConfigMap(dynakube *dynatracev1beta1.DynaKube, communicationHosts []dtclient.CommunicationHost) (*corev1.ConfigMap, error) {
	communicationHostsBytes, err := json.Marshal(communicationHosts)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakube.OneAgentConnectionInfoConfigMapName(),
			Namespace: testNamespace,
		},
		Data: map[string]string{
			TenantUUIDName:             "tenantUUID",
			CommunicationEndpointsName: "tenantEndpoints",
			CommunicationHosts:         string(communicationHostsBytes),
		},
	}, nil
}
