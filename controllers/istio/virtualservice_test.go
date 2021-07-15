package istio

import (
	"github.com/stretchr/testify/assert"
	istio "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

const (
	testPort1 = 1234
	testHost1 = "test-host-1"
	testPort2 = 5678
	testHost2 = "test-host-2"
)

func TestVirtualServiceGeneration(t *testing.T) {
	const (
		testName      = "com1"
		testNamespace = "dynatrace"
		testHost      = "comtest.com"
		testPort      = 8888
	)

	t.Run("generate for tls connection", func(t *testing.T) {
		expected := &istiov1alpha3.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
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
		result := buildVirtualService(testName, testNamespace, testHost, protocolHttps, testPort)

		assert.EqualValues(t, expected, result)
	})
	t.Run("generate for http connection", func(t *testing.T) {
		expected := &istiov1alpha3.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
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
		result := buildVirtualService(testName, testNamespace, testHost, protocolHttp, testPort)

		assert.EqualValues(t, expected, result)
	})
	t.Run("generate for invalid protocol", func(t *testing.T) {
		const invalidHost = "42.42.42.42"
		assert.Nil(t, buildVirtualService(testName, testNamespace, invalidHost, protocolHttp, testPort))
	})
}

func TestBuildVirtualServiceHttpRoute(t *testing.T) {
	expected := buildExpectedVirtualServiceHttpRoute(t)
	result := buildVirtualServiceHttpRoute(testPort1, testHost1)

	assert.Equal(t, len(expected), len(result))
	assert.True(t, reflect.DeepEqual(expected, result))

	result = buildVirtualServiceHttpRoute(testPort2, testHost2)

	assert.Equal(t, len(expected), len(result))
	assert.False(t, reflect.DeepEqual(expected, result))
}

func TestVirtualServiceTLSRoute(t *testing.T) {
	expected := buildExpectedVirtualServiceTLSRoute(t)
	result := buildVirtualServiceTLSRoute(testHost1, testPort1)

	assert.Equal(t, len(expected), len(result))
	assert.True(t, reflect.DeepEqual(expected, result))

	result = buildVirtualServiceTLSRoute(testHost2, testPort2)

	assert.Equal(t, len(expected), len(result))
	assert.False(t, reflect.DeepEqual(expected, result))
}

func TestBuildVirtualServiceSpec(t *testing.T) {
	t.Run(`is http route correctly set if protocol is "http"`, func(t *testing.T) {
		expected := buildExpectedVirtualServiceSpecHttp(t)
		result := buildVirtualServiceSpec(testHost1, protocolHttp, testPort1)

		assert.True(t, reflect.DeepEqual(expected, result))

		result = buildVirtualServiceSpec(testHost2, protocolHttp, testPort2)

		assert.False(t, reflect.DeepEqual(expected, result))
	})
	t.Run(`is TLS route correctly set if protocol is "https"`, func(t *testing.T) {
		expected := buildExpectedVirtualServiceSpecTls(t)
		result := buildVirtualServiceSpec(testHost1, protocolHttps, testPort1)

		assert.True(t, reflect.DeepEqual(expected, result))

		result = buildVirtualServiceSpec(testHost2, protocolHttps, testPort2)

		assert.False(t, reflect.DeepEqual(expected, result))
	})
}

func buildExpectedVirtualServiceHttpRoute(_ *testing.T) []*istio.HTTPRoute {
	return []*istio.HTTPRoute{{
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
	}}
}

func buildExpectedVirtualServiceTLSRoute(_ *testing.T) []*istio.TLSRoute {
	return []*istio.TLSRoute{{
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
	}}
}

func buildExpectedVirtualServiceSpecHttp(t *testing.T) istio.VirtualService {
	return istio.VirtualService{
		Hosts: []string{testHost1},
		Http:  buildExpectedVirtualServiceHttpRoute(t),
	}
}

func buildExpectedVirtualServiceSpecTls(t *testing.T) istio.VirtualService {
	return istio.VirtualService{
		Hosts: []string{testHost1},
		Tls:   buildExpectedVirtualServiceTLSRoute(t),
	}
}
