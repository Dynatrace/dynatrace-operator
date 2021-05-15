package status

import (
	"fmt"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testUUID = "test-uuid"

	testHost     = "test-host"
	testPort     = uint32(1234)
	testProtocol = "test-protocol"

	testAnotherHost     = "test-another-host"
	testAnotherPort     = uint32(5678)
	testAnotherProtocol = "test-another-protocol"

	testError = "test-error"
)

func TestStatusOptions(t *testing.T) {
	// Checks if StatusOptions struct and its properties exists
	_ = Options{
		Dtc:       &dtclient.MockDynatraceClient{},
		ApiClient: fake.NewClient(),
	}
}

func TestSetDynakubeStatus(t *testing.T) {
	t.Run(`set status`, func(t *testing.T) {
		instance := &dynatracev1alpha1.DynaKube{}
		dtc := &dtclient.MockDynatraceClient{}
		clt := fake.NewClient(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUUID,
			},
		})
		options := Options{
			Dtc:       dtc,
			ApiClient: clt,
		}

		dtc.On("GetCommunicationHostForClient").Return(dtclient.CommunicationHost{
			Protocol: testProtocol,
			Host:     testHost,
			Port:     testPort,
		}, nil)

		dtc.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{
			CommunicationHosts: []dtclient.CommunicationHost{
				{
					Protocol: testProtocol,
					Host:     testHost,
					Port:     testPort,
				},
				{
					Protocol: testAnotherProtocol,
					Host:     testAnotherHost,
					Port:     testAnotherPort,
				},
			},
			TenantUUID: testUUID,
		}, nil)

		err := SetDynakubeStatus(instance, options)

		assert.NoError(t, err)
		assert.Equal(t, testUUID, instance.Status.KubeSystemUUID)
		assert.NotNil(t, instance.Status.CommunicationHostForClient)
		assert.Equal(t, testHost, instance.Status.CommunicationHostForClient.Host)
		assert.Equal(t, testPort, instance.Status.CommunicationHostForClient.Port)
		assert.Equal(t, testProtocol, instance.Status.CommunicationHostForClient.Protocol)
		assert.NotNil(t, instance.Status.ConnectionInfo)
		assert.Equal(t, testUUID, instance.Status.ConnectionInfo.TenantUUID)
		assert.Equal(t, testUUID, instance.Status.EnvironmentID)
		assert.NotNil(t, instance.Status.ConnectionInfo.CommunicationHosts)
		assert.Equal(t, []dynatracev1alpha1.CommunicationHostStatus{
			{
				Protocol: testProtocol,
				Host:     testHost,
				Port:     testPort,
			},
			{
				Protocol: testAnotherProtocol,
				Host:     testAnotherHost,
				Port:     testAnotherPort,
			},
		}, instance.Status.ConnectionInfo.CommunicationHosts)
	})
	t.Run(`error querying kube system uid`, func(t *testing.T) {
		instance := &dynatracev1alpha1.DynaKube{}
		dtc := &dtclient.MockDynatraceClient{}
		clt := fake.NewClient()
		options := Options{
			Dtc:       dtc,
			ApiClient: clt,
		}

		err := SetDynakubeStatus(instance, options)
		assert.EqualError(t, err, "namespaces \"kube-system\" not found")
	})
	t.Run(`error querying communication host for client`, func(t *testing.T) {
		instance := &dynatracev1alpha1.DynaKube{}
		dtc := &dtclient.MockDynatraceClient{}
		clt := fake.NewClient(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUUID,
			},
		})
		options := Options{
			Dtc:       dtc,
			ApiClient: clt,
		}

		dtc.On("GetCommunicationHostForClient").Return(dtclient.CommunicationHost{}, fmt.Errorf(testError))

		err := SetDynakubeStatus(instance, options)
		assert.EqualError(t, err, testError)
	})
	t.Run(`error querying connection info`, func(t *testing.T) {
		instance := &dynatracev1alpha1.DynaKube{}
		dtc := &dtclient.MockDynatraceClient{}
		clt := fake.NewClient(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUUID,
			},
		})
		options := Options{
			Dtc:       dtc,
			ApiClient: clt,
		}

		dtc.On("GetCommunicationHostForClient").Return(dtclient.CommunicationHost{
			Protocol: testProtocol,
			Host:     testHost,
			Port:     testPort,
		}, nil)

		dtc.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{}, fmt.Errorf(testError))

		err := SetDynakubeStatus(instance, options)
		assert.EqualError(t, err, testError)
	})
}
