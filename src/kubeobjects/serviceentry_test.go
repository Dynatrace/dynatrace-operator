package kubeobjects

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	istio "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName        = "test-name"
	testPort1String = "1234"
)

func TestServiceEntryGeneration(t *testing.T) {
	const (
		testName      = "com1"
		testNamespace = "dynatrace"
		testHost      = "comtest.com"
		testPort      = 9999
	)

	t.Run(`generate with hostname`, func(t *testing.T) {
		expected := &istiov1alpha3.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: istio.ServiceEntry{
				Hosts:    []string{testHost},
				Location: istio.ServiceEntry_MESH_EXTERNAL,
				Ports: []*istio.Port{{
					Name:     ProtocolHttps + "-" + strconv.Itoa(testPort),
					Number:   testPort,
					Protocol: strings.ToUpper(ProtocolHttps),
				}},
				Resolution: istio.ServiceEntry_DNS,
			},
		}
		result := buildServiceEntry(testName, testNamespace, testHost, ProtocolHttps, testPort)
		assert.EqualValues(t, expected, result)

		result = buildServiceEntry(testName, testNamespace, testHost1, ProtocolHttps, testPort1)
		assert.NotEqualValues(t, expected, result)
	})
	t.Run(`generate with Ip`, func(t *testing.T) {
		const testIp = "42.42.42.42"
		expected := &istiov1alpha3.ServiceEntry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: istio.ServiceEntry{
				Hosts:     []string{IgnoredSubdomain},
				Addresses: []string{testIp + SubnetMask},
				Location:  istio.ServiceEntry_MESH_EXTERNAL,
				Ports: []*istio.Port{{
					Name:     ProtocolTcp + "-" + strconv.Itoa(testPort),
					Number:   testPort,
					Protocol: ProtocolTcp,
				}},
				Resolution: istio.ServiceEntry_NONE,
			},
		}
		result := buildServiceEntry(testName, testNamespace, testIp, ProtocolHttps, testPort)
		assert.EqualValues(t, expected, result)

		result = buildServiceEntry(testName, testNamespace, testIp, ProtocolHttps, testPort1)
		assert.NotEqualValues(t, expected, result)
	})
}

func TestBuildServiceEntryForHostname(t *testing.T) {
	expected := buildExpectedServiceEntryForHostname(t)
	result := buildServiceEntryFQDN(testName, testNamespace, testHost1, ProtocolHttp, testPort1)
	assert.EqualValues(t, expected, result)

	result = buildServiceEntryFQDN(testName, testNamespace, testHost2, ProtocolHttp, testPort2)
	assert.NotEqualValues(t, expected, result)
}

func TestBuildServiceEntryIp(t *testing.T) {
	expected := buildExpectedServiceEntryForIp(t)
	result := buildServiceEntryIP(testName, testNamespace, testHost1, testPort1)
	assert.EqualValues(t, expected, result)

	result = buildServiceEntryIP(testName, testNamespace, testHost2, testPort2)
	assert.NotEqualValues(t, expected, result)
}

func buildExpectedServiceEntryForHostname(_ *testing.T) *istiov1alpha3.ServiceEntry {
	return &istiov1alpha3.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: istio.ServiceEntry{
			Hosts: []string{testHost1},
			Ports: []*istio.Port{{
				Name:     ProtocolHttp + "-" + testPort1String,
				Number:   testPort1,
				Protocol: strings.ToUpper(ProtocolHttp),
			}},
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_DNS,
		},
	}
}

func buildExpectedServiceEntryForIp(_ *testing.T) *istiov1alpha3.ServiceEntry {
	return &istiov1alpha3.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: istio.ServiceEntry{
			Hosts:     []string{IgnoredSubdomain},
			Addresses: []string{testHost1 + SubnetMask},
			Ports: []*istio.Port{{
				Name:     ProtocolTcp + "-" + testPort1String,
				Number:   testPort1,
				Protocol: ProtocolTcp,
			}},
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_NONE,
		},
	}
}
