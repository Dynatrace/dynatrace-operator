package istio

import (
	"reflect"
	"testing"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	istio "istio.io/api/networking/v1beta1"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testPort1 = 1234
	testHost1 = "test-host-1"
	testIP1   = "42.42.42.42"
	testPort2 = 5678
	testHost2 = "test-host-2"
	testIP2   = "66.249.65.40"
)

func TestVirtualServiceGeneration(t *testing.T) {
	const (
		testName      = "com1"
		testNamespace = "dynatrace"
		testHost      = "comtest.com"
		testPort      = 8888
	)

	t.Run("generate for tls connection", func(t *testing.T) {
		expected := &istiov1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels:    buildTestLabels(),
			},
			Spec: istio.VirtualService{
				Hosts: []string{testHost},
				Tls: []*istio.TLSRoute{{
					Match: []*istio.TLSMatchAttributes{{
						Port:     testPort,
						SniHosts: []string{testHost}},
					},
					Route: []*istio.RouteDestination{{
						Destination: &istio.Destination{
							Host: testHost,
							Port: &istio.PortSelector{Number: testPort},
						}},
					},
				}}},
		}
		commHosts := []dtclient.CommunicationHost{{
			Host:     testHost,
			Port:     testPort,
			Protocol: protocolHttps,
		}}
		result := buildVirtualService(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts)

		assert.EqualValues(t, expected, result)
	})
	t.Run("generate for http connection", func(t *testing.T) {
		expected := &istiov1beta1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels:    buildTestLabels(),
			},
			Spec: istio.VirtualService{
				Hosts: []string{testHost},
				Http: []*istio.HTTPRoute{{
					Match: []*istio.HTTPMatchRequest{{
						Port: testPort,
					}},
					Route: []*istio.HTTPRouteDestination{{
						Destination: &istio.Destination{
							Host: testHost,
							Port: &istio.PortSelector{Number: testPort},
						}},
					},
				}}},
		}
		commHosts := []dtclient.CommunicationHost{{
			Host:     testHost,
			Port:     testPort,
			Protocol: protocolHttp,
		}}
		result := buildVirtualService(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts)

		assert.EqualValues(t, expected, result)
	})
	t.Run("generate for invalid protocol", func(t *testing.T) {
		commHosts := []dtclient.CommunicationHost{{
			Host:     "42.42.42.42",
			Port:     testPort,
			Protocol: protocolHttp,
		}}
		assert.Nil(t, buildVirtualService(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts))
	})
}

func TestBuildVirtualServiceHttpRoute(t *testing.T) {
	expected := buildExpectedVirtualServiceHttpRoute(t)
	result := buildVirtualServiceHttpRoute(testHost1, testPort1)

	assert.True(t, reflect.DeepEqual(expected, result))

	result = buildVirtualServiceHttpRoute(testHost2, testPort2)

	assert.False(t, reflect.DeepEqual(expected, result))
}

func TestVirtualServiceTLSRoute(t *testing.T) {
	expected := buildExpectedVirtualServiceTLSRoute(t)
	result := buildVirtualServiceTLSRoute(testHost1, testPort1)

	assert.True(t, reflect.DeepEqual(expected, result))

	result = buildVirtualServiceTLSRoute(testHost2, testPort2)

	assert.False(t, reflect.DeepEqual(expected, result))
}

func TestBuildVirtualServiceSpec(t *testing.T) {
	t.Run(`is http route correctly set if protocol is "http"`, func(t *testing.T) {
		expected := buildExpectedVirtualServiceSpecHttp(t)
		result := buildVirtualServiceSpec([]dtclient.CommunicationHost{
			{Host: testHost1, Port: testPort1, Protocol: protocolHttp},
		})

		assert.True(t, reflect.DeepEqual(expected.DeepCopy(), result.DeepCopy()))

		result = buildVirtualServiceSpec([]dtclient.CommunicationHost{
			{Host: testHost2, Port: testPort2, Protocol: protocolHttp},
		})

		assert.False(t, reflect.DeepEqual(expected.DeepCopy(), result.DeepCopy()))
	})
	t.Run(`is TLS route correctly set if protocol is "https"`, func(t *testing.T) {
		expected := buildExpectedVirtualServiceSpecTls(t)
		result := buildVirtualServiceSpec([]dtclient.CommunicationHost{
			{Host: testHost1, Port: testPort1, Protocol: protocolHttps},
		})

		assert.True(t, reflect.DeepEqual(expected.DeepCopy(), result.DeepCopy()))

		result = buildVirtualServiceSpec([]dtclient.CommunicationHost{
			{Host: testHost2, Port: testPort2, Protocol: protocolHttps},
		})

		assert.False(t, reflect.DeepEqual(expected.DeepCopy(), result.DeepCopy()))
	})
}

func buildExpectedVirtualServiceHttpRoute(_ *testing.T) *istio.HTTPRoute {
	return &istio.HTTPRoute{
		Match: []*istio.HTTPMatchRequest{{
			Port: testPort1,
		}},
		Route: []*istio.HTTPRouteDestination{{
			Destination: &istio.Destination{
				Host: testHost1,
				Port: &istio.PortSelector{
					Number: testPort1,
				},
			},
		}},
	}
}

func buildExpectedVirtualServiceTLSRoute(_ *testing.T) *istio.TLSRoute {
	return &istio.TLSRoute{
		Match: []*istio.TLSMatchAttributes{{
			SniHosts: []string{testHost1},
			Port:     testPort1,
		}},
		Route: []*istio.RouteDestination{{
			Destination: &istio.Destination{
				Host: testHost1,
				Port: &istio.PortSelector{
					Number: testPort1,
				},
			},
		}},
	}
}

func buildExpectedVirtualServiceSpecHttp(t *testing.T) istio.VirtualService {
	return istio.VirtualService{
		Hosts: []string{testHost1},
		Http:  []*istio.HTTPRoute{buildExpectedVirtualServiceHttpRoute(t)},
	}
}

func buildExpectedVirtualServiceSpecTls(t *testing.T) istio.VirtualService {
	return istio.VirtualService{
		Hosts: []string{testHost1},
		Tls:   []*istio.TLSRoute{buildExpectedVirtualServiceTLSRoute(t)},
	}
}

func buildTestLabels() map[string]string {
	return map[string]string{
		"test": "test",
	}
}
