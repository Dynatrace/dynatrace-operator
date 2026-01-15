package istio

import (
	"strconv"
	"strings"
	"testing"

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

	t.Run("generate with hostname", func(t *testing.T) {
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
					Name:     protocolHTTPS + "-" + strconv.Itoa(testPort),
					Number:   testPort,
					Protocol: strings.ToUpper(protocolHTTPS),
				}},
				Resolution: istio.ServiceEntry_DNS,
			},
		}
		commHosts1 := []CommunicationHost{{
			Host:     testHost,
			Port:     testPort,
			Protocol: protocolHTTPS,
		}}
		result := buildServiceEntryFQDNs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
		assert.Equal(t, expected, result)

		commHosts2 := []CommunicationHost{{
			Host:     testHost1,
			Port:     testPort1,
			Protocol: protocolHTTPS,
		}}
		result = buildServiceEntryFQDNs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts2)
		assert.NotEqual(t, expected, result)
	})
	t.Run("generate with two different hostnames and same port", func(t *testing.T) {
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
					Name:     protocolHTTPS + "-" + strconv.Itoa(testPort),
					Number:   testPort,
					Protocol: strings.ToUpper(protocolHTTPS),
				}},
				Resolution: istio.ServiceEntry_DNS,
			},
		}
		commHosts1 := []CommunicationHost{{
			Host:     testHost,
			Port:     testPort,
			Protocol: protocolHTTPS,
		},
			{
				Host:     testHost1,
				Port:     testPort,
				Protocol: protocolHTTPS,
			}}
		result := buildServiceEntryFQDNs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
		assert.Equal(t, expected, result)
	})
	t.Run("generate with Ip", func(t *testing.T) {
		const testIP = "42.42.42.42"
		expected := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels:    buildTestLabels(),
			},
			Spec: istio.ServiceEntry{
				Hosts:     []string{ignoredSubdomain},
				Addresses: []string{testIP + subnetMask},
				Location:  istio.ServiceEntry_MESH_EXTERNAL,
				Ports: []*istio.ServicePort{{
					Name:     protocolTCP + "-" + strconv.Itoa(testPort),
					Number:   testPort,
					Protocol: protocolTCP,
				}},
				Resolution: istio.ServiceEntry_NONE,
			},
		}
		commHosts1 := []CommunicationHost{{
			Host:     testIP,
			Port:     testPort,
			Protocol: protocolHTTPS,
		}}
		result := buildServiceEntryIPs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
		assert.Equal(t, expected, result)

		commHosts2 := []CommunicationHost{{
			Host:     testIP,
			Port:     testPort1,
			Protocol: protocolHTTPS,
		}}
		result = buildServiceEntryIPs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts2)
		assert.NotEqual(t, expected, result)
	})
	t.Run("generate with two different Ips and same ports", func(t *testing.T) {
		const (
			testIP  = "42.42.42.42"
			testIP1 = "42.42.42.43"
		)

		expected := &istiov1beta1.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels:    buildTestLabels(),
			},
			Spec: istio.ServiceEntry{
				Hosts:     []string{ignoredSubdomain},
				Addresses: []string{testIP + subnetMask, testIP1 + subnetMask},
				Location:  istio.ServiceEntry_MESH_EXTERNAL,
				Ports: []*istio.ServicePort{{
					Name:     protocolTCP + "-" + strconv.Itoa(testPort),
					Number:   testPort,
					Protocol: protocolTCP,
				}},
				Resolution: istio.ServiceEntry_NONE,
			},
		}
		commHosts1 := []CommunicationHost{{
			Host:     testIP,
			Port:     testPort,
			Protocol: protocolHTTPS,
		},
			{
				Host:     testIP1,
				Port:     testPort,
				Protocol: protocolHTTPS,
			}}
		result := buildServiceEntryIPs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
		assert.Equal(t, expected, result)
	})
}

func TestBuildServiceEntryForHostname(t *testing.T) {
	expected := buildExpectedServiceEntryForHostname(t)
	commHosts1 := []CommunicationHost{{
		Host:     testHost1,
		Port:     testPort1,
		Protocol: protocolHTTP,
	}}
	result := buildServiceEntryFQDNs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
	assert.Equal(t, expected, result)

	commHosts2 := []CommunicationHost{{
		Host:     testHost2,
		Port:     testPort2,
		Protocol: protocolHTTP,
	}}
	result = buildServiceEntryFQDNs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts2)
	assert.NotEqual(t, expected, result)
}

func TestBuildServiceEntryIp(t *testing.T) {
	expected := buildExpectedServiceEntryForIP(t)
	commHosts1 := []CommunicationHost{{
		Host: testIP1,
		Port: testPort1,
	}}
	result := buildServiceEntryIPs(buildObjectMeta(testName, testNamespace, buildTestLabels()), commHosts1)
	assert.Equal(t, expected, result)

	commHosts2 := []CommunicationHost{{
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
				Name:     protocolHTTP + "-" + testPort1String,
				Number:   testPort1,
				Protocol: strings.ToUpper(protocolHTTP),
			}},
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_DNS,
		},
	}
}

func buildExpectedServiceEntryForIP(_ *testing.T) *istiov1beta1.ServiceEntry {
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
				Name:     protocolTCP + "-" + testPort1String,
				Number:   testPort1,
				Protocol: protocolTCP,
			}},
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_NONE,
		},
	}
}
