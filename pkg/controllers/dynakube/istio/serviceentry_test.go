package istio

import (
	"strconv"
	"strings"
	"testing"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	istio "istio.io/api/networking/v1beta1"
	istiov1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName        = "test-name"
	testPort1String = "1234"
	testNamespace   = "dynatrace"
)

func TestServiceEntryGeneration(t *testing.T) {
	const (
		testHost  = "comtest.com"
		testHost1 = "int.comtest.com"

		testPort = 9999
	)

	t.Run(`generate with hostname`, func(t *testing.T) {
		expected := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels:    buildTestLabels(),
			},
			Spec: istio.ServiceEntry{
				Hosts:    []string{testHost},
				Location: istio.ServiceEntry_MESH_EXTERNAL,
				Ports: []*istio.ServicePort{{
					Name:     protocolHttps + "-" + strconv.Itoa(testPort),
					Number:   testPort,
					Protocol: strings.ToUpper(protocolHttps),
				}},
				Resolution: istio.ServiceEntry_DNS,
			},
		}
		commHosts1 := []dtclient.CommunicationHost{{
			Host:     testHost,
			Port:     testPort,
			Protocol: protocolHttps,
		}}
		result := buildServiceEntryFQDNs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
		assert.Equal(t, expected, result)

		commHosts2 := []dtclient.CommunicationHost{{
			Host:     testHost1,
			Port:     testPort1,
			Protocol: protocolHttps,
		}}
		result = buildServiceEntryFQDNs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts2)
		assert.NotEqual(t, expected, result)
	})
	t.Run(`generate with two different hostnames and same port`, func(t *testing.T) {
		expected := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels:    buildTestLabels(),
			},
			Spec: istio.ServiceEntry{
				Hosts:    []string{testHost, testHost1},
				Location: istio.ServiceEntry_MESH_EXTERNAL,
				Ports: []*istio.ServicePort{{
					Name:     protocolHttps + "-" + strconv.Itoa(testPort),
					Number:   testPort,
					Protocol: strings.ToUpper(protocolHttps),
				}},
				Resolution: istio.ServiceEntry_DNS,
			},
		}
		commHosts1 := []dtclient.CommunicationHost{{
			Host:     testHost,
			Port:     testPort,
			Protocol: protocolHttps,
		},
			{
				Host:     testHost1,
				Port:     testPort,
				Protocol: protocolHttps,
			}}
		result := buildServiceEntryFQDNs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
		assert.Equal(t, expected, result)
	})
	t.Run(`generate with Ip`, func(t *testing.T) {
		const testIp = "42.42.42.42"
		expected := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels:    buildTestLabels(),
			},
			Spec: istio.ServiceEntry{
				Hosts:     []string{ignoredSubdomain},
				Addresses: []string{testIp + subnetMask},
				Location:  istio.ServiceEntry_MESH_EXTERNAL,
				Ports: []*istio.ServicePort{{
					Name:     protocolTcp + "-" + strconv.Itoa(testPort),
					Number:   testPort,
					Protocol: protocolTcp,
				}},
				Resolution: istio.ServiceEntry_NONE,
			},
		}
		commHosts1 := []dtclient.CommunicationHost{{
			Host:     testIp,
			Port:     testPort,
			Protocol: protocolHttps,
		}}
		result := buildServiceEntryIPs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
		assert.Equal(t, expected, result)

		commHosts2 := []dtclient.CommunicationHost{{
			Host:     testIp,
			Port:     testPort1,
			Protocol: protocolHttps,
		}}
		result = buildServiceEntryIPs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts2)
		assert.NotEqual(t, expected, result)
	})
	t.Run(`generate with two different Ips and same ports`, func(t *testing.T) {
		const (
			testIp  = "42.42.42.42"
			testIp1 = "42.42.42.43"
		)

		expected := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels:    buildTestLabels(),
			},
			Spec: istio.ServiceEntry{
				Hosts:     []string{ignoredSubdomain},
				Addresses: []string{testIp + subnetMask, testIp1 + subnetMask},
				Location:  istio.ServiceEntry_MESH_EXTERNAL,
				Ports: []*istio.ServicePort{{
					Name:     protocolTcp + "-" + strconv.Itoa(testPort),
					Number:   testPort,
					Protocol: protocolTcp,
				}},
				Resolution: istio.ServiceEntry_NONE,
			},
		}
		commHosts1 := []dtclient.CommunicationHost{{
			Host:     testIp,
			Port:     testPort,
			Protocol: protocolHttps,
		},
			{
				Host:     testIp1,
				Port:     testPort,
				Protocol: protocolHttps,
			}}
		result := buildServiceEntryIPs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
		assert.Equal(t, expected, result)
	})
}

func TestBuildServiceEntryForHostname(t *testing.T) {
	expected := buildExpectedServiceEntryForHostname(t)
	commHosts1 := []dtclient.CommunicationHost{{
		Host:     testHost1,
		Port:     testPort1,
		Protocol: protocolHttp,
	}}
	result := buildServiceEntryFQDNs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
	assert.Equal(t, expected, result)

	commHosts2 := []dtclient.CommunicationHost{{
		Host:     testHost2,
		Port:     testPort2,
		Protocol: protocolHttp,
	}}
	result = buildServiceEntryFQDNs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts2)
	assert.NotEqual(t, expected, result)
}

func TestBuildServiceEntryIp(t *testing.T) {
	expected := buildExpectedServiceEntryForIp(t)
	commHosts1 := []dtclient.CommunicationHost{{
		Host: testIP1,
		Port: testPort1,
	}}
	result := buildServiceEntryIPs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
	assert.Equal(t, expected, result)

	commHosts2 := []dtclient.CommunicationHost{{
		Host: testIP2,
		Port: testPort2,
	}}
	result = buildServiceEntryIPs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts2)
	assert.NotEqual(t, expected, result)
}

func buildExpectedServiceEntryForHostname(_ *testing.T) *istiov1beta1.ServiceEntry {
	return &istiov1beta1.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
			Labels:    buildTestLabels(),
		},
		Spec: istio.ServiceEntry{
			Hosts: []string{testHost1},
			Ports: []*istio.ServicePort{{
				Name:     protocolHttp + "-" + testPort1String,
				Number:   testPort1,
				Protocol: strings.ToUpper(protocolHttp),
			}},
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_DNS,
		},
	}
}

func buildExpectedServiceEntryForIp(_ *testing.T) *istiov1beta1.ServiceEntry {
	return &istiov1beta1.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
			Labels:    buildTestLabels(),
		},
		Spec: istio.ServiceEntry{
			Hosts:     []string{ignoredSubdomain},
			Addresses: []string{"42.42.42.42" + subnetMask},
			Ports: []*istio.ServicePort{{
				Name:     protocolTcp + "-" + testPort1String,
				Number:   testPort1,
				Protocol: protocolTcp,
			}},
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_NONE,
		},
	}
}
