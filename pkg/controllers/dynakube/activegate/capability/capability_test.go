package capability

import (
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace     = "test-namespace"
	testName          = "test-name"
	testApiUrl        = "https://demo.dev.dynatracelabs.com/api"
	expectedShortName = "activegate"
	expectedArgName   = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface"
)

var capabilities = []dynatracev1beta2.CapabilityDisplayName{
	dynatracev1beta2.RoutingCapability.DisplayName,
	dynatracev1beta2.KubeMonCapability.DisplayName,
	dynatracev1beta2.MetricsIngestCapability.DisplayName,
	dynatracev1beta2.DynatraceApiCapability.DisplayName,
}

func buildDynakube(capabilities []dynatracev1beta2.CapabilityDisplayName) *dynatracev1beta2.DynaKube {
	return &dynatracev1beta2.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace, Name: testName,
		},
		Spec: dynatracev1beta2.DynaKubeSpec{
			APIURL: testApiUrl,
			ActiveGate: dynatracev1beta2.ActiveGateSpec{
				Capabilities: capabilities,
			},
		},
	}
}

func TestBuildProxySecretName(t *testing.T) {
	t.Run(`correct secret name`, func(t *testing.T) {
		expectedProxySecretName := "someDK-internal-proxy"
		actualProxySecretName := proxy.BuildSecretName("someDK")
		require.NotEmpty(t, actualProxySecretName)
		assert.Equal(t, expectedProxySecretName, actualProxySecretName)
	})
}

func TestBuildServiceName(t *testing.T) {
	t.Run(`build service name`, func(t *testing.T) {
		expectedServiceName := "testName-testModule"
		actualServiceName := BuildServiceName("testName", "testModule")
		require.NotEmpty(t, actualServiceName)
		assert.Equal(t, expectedServiceName, actualServiceName)
	})
}

func TestNewMultiCapability(t *testing.T) {
	t.Run(`creates new multicapability`, func(t *testing.T) {
		dynakube := buildDynakube(capabilities)
		mc := NewMultiCapability(dynakube)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgName, mc.ArgName())
	})
	t.Run(`creates new multicapability without capabilities set in dynakube`, func(t *testing.T) {
		var emptyCapabilites []dynatracev1beta2.CapabilityDisplayName
		dynakube := buildDynakube(emptyCapabilites)
		mc := NewMultiCapability(dynakube)
		require.NotNil(t, mc)
		assert.False(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, "", mc.ArgName())
	})
}

func TestBuildServiceDomainNameForDNSEntryPoint(t *testing.T) {
	actual := buildServiceDomainName("test-name", "test-namespace", "test-component-feature")
	assert.NotEmpty(t, actual)

	expected := "test-name-test-component-feature.test-namespace:443"
	assert.Equal(t, expected, actual)

	testStringName := "this---dynakube_string"
	testNamespace := "this_is---namespace_string"
	testStringFeature := "SHOULD--_--PaRsEcORrEcTlY"
	expected = "this---dynakube_string-SHOULD--_--PaRsEcORrEcTlY.this_is---namespace_string:443"
	actual = buildServiceDomainName(testStringName, testNamespace, testStringFeature)
	assert.Equal(t, expected, actual)
}

func TestBuildDNSEntryPoint(t *testing.T) {
	type capabilityBuilder func(*dynatracev1beta2.DynaKube) Capability

	type testCase struct {
		title       string
		dk          *dynatracev1beta2.DynaKube
		capability  capabilityBuilder
		expectedDNS string
	}

	testCases := []testCase{
		{
			title: "DNSEntryPoint for ActiveGate routing capability",
			dk: &dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynatracev1beta2.DynaKubeSpec{
					ActiveGate: dynatracev1beta2.ActiveGateSpec{
						Capabilities: []dynatracev1beta2.CapabilityDisplayName{
							dynatracev1beta2.RoutingCapability.DisplayName,
						},
					},
				},
				Status: dynatracev1beta2.DynaKubeStatus{
					ActiveGate: dynatracev1beta2.ActiveGateStatus{
						ServiceIPs: []string{"1.2.3.4"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication,https://dynakube-activegate.dynatrace:443/communication",
		},
		{
			title: "DNSEntryPoint with multiple service IPs",
			dk: &dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynatracev1beta2.DynaKubeSpec{
					ActiveGate: dynatracev1beta2.ActiveGateSpec{
						Capabilities: []dynatracev1beta2.CapabilityDisplayName{
							dynatracev1beta2.RoutingCapability.DisplayName,
						},
					},
				},
				Status: dynatracev1beta2.DynaKubeStatus{
					ActiveGate: dynatracev1beta2.ActiveGateStatus{
						ServiceIPs: []string{"1.2.3.4", "4.3.2.1"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication,https://4.3.2.1:443/communication,https://dynakube-activegate.dynatrace:443/communication",
		},
		{
			title: "DNSEntryPoint with multiple service IPs, dual-stack",
			dk: &dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynatracev1beta2.DynaKubeSpec{
					ActiveGate: dynatracev1beta2.ActiveGateSpec{
						Capabilities: []dynatracev1beta2.CapabilityDisplayName{
							dynatracev1beta2.RoutingCapability.DisplayName,
						},
					},
				},
				Status: dynatracev1beta2.DynaKubeStatus{
					ActiveGate: dynatracev1beta2.ActiveGateStatus{
						ServiceIPs: []string{"1.2.3.4", "2600:2d00:0:4:f9b7:bd67:1d97:5994", "4.3.2.1", "2600:2d00:0:4:f9b7:bd67:1d97:5996"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication,https://[2600:2d00:0:4:f9b7:bd67:1d97:5994]:443/communication,https://4.3.2.1:443/communication,https://[2600:2d00:0:4:f9b7:bd67:1d97:5996]:443/communication,https://dynakube-activegate.dynatrace:443/communication",
		},
		{
			title: "DNSEntryPoint for ActiveGate k8s monitoring capability",
			dk: &dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynatracev1beta2.DynaKubeSpec{
					ActiveGate: dynatracev1beta2.ActiveGateSpec{
						Capabilities: []dynatracev1beta2.CapabilityDisplayName{
							dynatracev1beta2.KubeMonCapability.DisplayName,
						},
					},
				},
				Status: dynatracev1beta2.DynaKubeStatus{
					ActiveGate: dynatracev1beta2.ActiveGateStatus{
						ServiceIPs: []string{"1.2.3.4"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication",
		},
		{
			title: "DNSEntryPoint for ActiveGate routing+kubemon capabilities",
			dk: &dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynatracev1beta2.DynaKubeSpec{
					ActiveGate: dynatracev1beta2.ActiveGateSpec{
						Capabilities: []dynatracev1beta2.CapabilityDisplayName{
							dynatracev1beta2.KubeMonCapability.DisplayName,
							dynatracev1beta2.RoutingCapability.DisplayName,
						},
					},
				},
				Status: dynatracev1beta2.DynaKubeStatus{
					ActiveGate: dynatracev1beta2.ActiveGateStatus{
						ServiceIPs: []string{"1.2.3.4"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication,https://dynakube-activegate.dynatrace:443/communication",
		},
		{
			title: "DNSEntryPoint for deprecated routing ActiveGate",
			dk: &dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynatracev1beta2.DynaKubeSpec{
					ActiveGate: dynatracev1beta2.ActiveGateSpec{
						Capabilities: capabilities,
					},
				},
				Status: dynatracev1beta2.DynaKubeStatus{
					ActiveGate: dynatracev1beta2.ActiveGateStatus{
						ServiceIPs: []string{"1.2.3.4"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication,https://dynakube-activegate.dynatrace:443/communication",
		},
		{
			title: "DNSEntryPoint for deprecated kubernetes monitoring ActiveGate",
			dk: &dynatracev1beta2.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynatracev1beta2.DynaKubeSpec{},
				Status: dynatracev1beta2.DynaKubeStatus{
					ActiveGate: dynatracev1beta2.ActiveGateStatus{
						ServiceIPs: []string{"1.2.3.4"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication",
		},
	}
	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			capability := test.capability(test.dk)
			dnsEntryPoint := BuildDNSEntryPoint(*test.dk, capability)
			assert.Equal(t, test.expectedDNS, dnsEntryPoint)
		})
	}
}
