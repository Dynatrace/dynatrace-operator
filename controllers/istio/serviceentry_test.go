package istio

import (
	"github.com/stretchr/testify/assert"
	istio "istio.io/api/networking/v1alpha3"
	istiov1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

const (
	testName        = "test-name"
	testNamespace   = "test-namespace"
	testPort1String = "1234"
)

func TestBuildServiceEntryForHostname(t *testing.T) {
	expected := &istiov1alpha3.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: istio.ServiceEntry{
			Hosts: []string{testHost1},
			Ports: []*istio.Port{{
				Name:     protocolHttp + "-" + testPort1String,
				Number:   testPort1,
				Protocol: strings.ToUpper(protocolHttp),
			}},
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_DNS,
		},
	}
	result := buildServiceEntryFQDN(testName, testNamespace, testHost1, protocolHttp, testPort1)
	assert.EqualValues(t, expected, result)
}

func TestBuildServiceEntryIp(t *testing.T) {
	expected := buildExpectedServiceEntryForIp(t)
	result := buildServiceEntryIP(testName, testNamespace, testHost1, testPort1)
	assert.EqualValues(t, expected, result)

	result = buildServiceEntryIP(testName, testNamespace, testHost2, testPort2)
	assert.NotEqualValues(t, expected, result)
}

func buildExpectedServiceEntryForIp(_ *testing.T) *istiov1alpha3.ServiceEntry {
	return &istiov1alpha3.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: istio.ServiceEntry{
			Hosts:     []string{ignoredSubdomain},
			Addresses: []string{testHost1 + subnetMask},
			Ports: []*istio.Port{{
				Name:     protocolTcp + "-" + testPort1String,
				Number:   testPort1,
				Protocol: protocolTcp,
			}},
			Location:   istio.ServiceEntry_MESH_EXTERNAL,
			Resolution: istio.ServiceEntry_NONE,
		},
	}
}
