package capability

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace                              = "test-namespace"
	testName                                   = "test-name"
	testApiUrl                                 = "https://demo.dev.dynatracelabs.com/api"
	expectedShortName                          = "activegate"
	expectedArgName                            = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface"
	expectedArgNameWithDebugging               = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface,debugging"
	expectedArgNameWithExtensions              = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface,extension_controller"
	expectedArgNameWithExtensionsOnly          = "extension_controller"
	expectedArgNameWithOTLPingest              = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface,log_analytics_collector,generic_ingest,otlp_ingest"
	expectedArgNameWithOTLPingestOnly          = "log_analytics_collector,generic_ingest,otlp_ingest"
	expectedArgNameWithExtensionsAndOTLPingest = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface,extension_controller,log_analytics_collector,generic_ingest,otlp_ingest"
	expectedArgNameWithTelemetryIngest         = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface,log_analytics_collector,generic_ingest,otlp_ingest"
)

var capabilities = []activegate.CapabilityDisplayName{
	activegate.RoutingCapability.DisplayName,
	activegate.KubeMonCapability.DisplayName,
	activegate.MetricsIngestCapability.DisplayName,
	activegate.DynatraceApiCapability.DisplayName,
}

func buildDynakube(capabilities []activegate.CapabilityDisplayName, enableExtensions bool, enableOTLPingest bool, enableTelemetryIngest bool) *dynakube.DynaKube {
	extensionsSpec := &dynakube.ExtensionsSpec{}
	if !enableExtensions {
		extensionsSpec = nil
	}

	telemetryIngestSpec := &telemetryingest.Spec{}
	if !enableTelemetryIngest {
		telemetryIngestSpec = nil
	}

	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace, Name: testName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testApiUrl,
			ActiveGate: activegate.Spec{
				Capabilities: capabilities,
			},
			Extensions:       extensionsSpec,
			EnableOTLPingest: enableOTLPingest,
			TelemetryIngest:  telemetryIngestSpec,
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
		dk := buildDynakube(capabilities, false, false, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgName, mc.ArgName())
	})
	t.Run(`creates new multicapability without capabilities set in dynakube`, func(t *testing.T) {
		var emptyCapabilites []activegate.CapabilityDisplayName
		dk := buildDynakube(emptyCapabilites, false, false, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.False(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Empty(t, mc.ArgName())
	})
}

func TestNewMultiCapabilityWithExtensions(t *testing.T) {
	t.Run(`creates new multicapability with Extensions enabled`, func(t *testing.T) {
		dk := buildDynakube(capabilities, true, false, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgNameWithExtensions, mc.ArgName())
	})
	t.Run(`creates new multicapability without capabilities set in dynakube and Extensions enabled`, func(t *testing.T) {
		var emptyCapabilites []activegate.CapabilityDisplayName
		dk := buildDynakube(emptyCapabilites, true, false, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgNameWithExtensionsOnly, mc.ArgName())
	})
}

func TestNewMultiCapabilityWithOTLPingest(t *testing.T) {
	t.Run(`creates new multicapability with OTLPingest enabled`, func(t *testing.T) {
		dk := buildDynakube(capabilities, false, true, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgNameWithOTLPingest, mc.ArgName())
	})
	t.Run(`creates new multicapability without capabilities set in dynakube and OTLPingest enabled`, func(t *testing.T) {
		var emptyCapabilites []activegate.CapabilityDisplayName
		dk := buildDynakube(emptyCapabilites, false, true, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgNameWithOTLPingestOnly, mc.ArgName())
	})
}

func TestNewMultiCapabilityWithExtensionsAndOTLPingest(t *testing.T) {
	t.Run(`creates new multicapability with Extensions and OTLPingest enabled`, func(t *testing.T) {
		dk := buildDynakube(capabilities, true, true, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgNameWithExtensionsAndOTLPingest, mc.ArgName())
	})
}

func TestNewMultiCapabilityWithTelemetryIngest(t *testing.T) {
	t.Run(`creates new multicapability with TelemetryIngest enabled`, func(t *testing.T) {
		dk := buildDynakube(capabilities, false, false, true)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgNameWithTelemetryIngest, mc.ArgName())
	})
	t.Run(`creates new multicapability without capabilities set in dynakube and TelemetryIngest enabled`, func(t *testing.T) {
		var emptyCapabilites []activegate.CapabilityDisplayName
		dk := buildDynakube(emptyCapabilites, false, false, true)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.False(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Empty(t, mc.ArgName())
	})
}

func TestNewMultiCapabilityWithOTLPingestAndTelemetryIngest(t *testing.T) {
	t.Run(`creates new multicapability without capabilities set in dynakube and with OTLPingest and TelemetryIngest enabled`, func(t *testing.T) {
		var emptyCapabilites []activegate.CapabilityDisplayName
		dk := buildDynakube(emptyCapabilites, false, true, true)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgNameWithOTLPingestOnly, mc.ArgName())
	})
}

func TestNewMultiCapabilityWithDebugging(t *testing.T) {
	t.Run(`creates new multicapability with debugging capability enabled`, func(t *testing.T) {
		dk := buildDynakube(append(capabilities, activegate.DebuggingCapability.DisplayName), false, false, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgNameWithDebugging, mc.ArgName())
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
	type capabilityBuilder func(*dynakube.DynaKube) Capability

	type testCase struct {
		title       string
		dk          *dynakube.DynaKube
		capability  capabilityBuilder
		expectedDNS string
	}

	testCases := []testCase{
		{
			title: "DNSEntryPoint for ActiveGate routing capability",
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynakube.DynaKubeSpec{
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.RoutingCapability.DisplayName,
						},
					},
				},
				Status: dynakube.DynaKubeStatus{
					ActiveGate: activegate.Status{
						ServiceIPs: []string{"1.2.3.4"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication,https://dynakube-activegate.dynatrace:443/communication",
		},
		{
			title: "DNSEntryPoint with multiple service IPs",
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynakube.DynaKubeSpec{
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.RoutingCapability.DisplayName,
						},
					},
				},
				Status: dynakube.DynaKubeStatus{
					ActiveGate: activegate.Status{
						ServiceIPs: []string{"1.2.3.4", "4.3.2.1"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication,https://4.3.2.1:443/communication,https://dynakube-activegate.dynatrace:443/communication",
		},
		{
			title: "DNSEntryPoint with multiple service IPs, dual-stack",
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynakube.DynaKubeSpec{
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.RoutingCapability.DisplayName,
						},
					},
				},
				Status: dynakube.DynaKubeStatus{
					ActiveGate: activegate.Status{
						ServiceIPs: []string{"1.2.3.4", "2600:2d00:0:4:f9b7:bd67:1d97:5994", "4.3.2.1", "2600:2d00:0:4:f9b7:bd67:1d97:5996"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication,https://[2600:2d00:0:4:f9b7:bd67:1d97:5994]:443/communication,https://4.3.2.1:443/communication,https://[2600:2d00:0:4:f9b7:bd67:1d97:5996]:443/communication,https://dynakube-activegate.dynatrace:443/communication",
		},
		{
			title: "DNSEntryPoint for ActiveGate k8s monitoring capability",
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynakube.DynaKubeSpec{
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
				},
				Status: dynakube.DynaKubeStatus{
					ActiveGate: activegate.Status{
						ServiceIPs: []string{"1.2.3.4"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication",
		},
		{
			title: "DNSEntryPoint for ActiveGate routing+kubemon capabilities",
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynakube.DynaKubeSpec{
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
							activegate.RoutingCapability.DisplayName,
						},
					},
				},
				Status: dynakube.DynaKubeStatus{
					ActiveGate: activegate.Status{
						ServiceIPs: []string{"1.2.3.4"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication,https://dynakube-activegate.dynatrace:443/communication",
		},
		{
			title: "DNSEntryPoint for deprecated routing ActiveGate",
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynakube.DynaKubeSpec{
					ActiveGate: activegate.Spec{
						Capabilities: capabilities,
					},
				},
				Status: dynakube.DynaKubeStatus{
					ActiveGate: activegate.Status{
						ServiceIPs: []string{"1.2.3.4"},
					},
				},
			},
			capability:  NewMultiCapability,
			expectedDNS: "https://1.2.3.4:443/communication,https://dynakube-activegate.dynatrace:443/communication",
		},
		{
			title: "DNSEntryPoint for deprecated kubernetes monitoring ActiveGate",
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynakube.DynaKubeSpec{},
				Status: dynakube.DynaKubeStatus{
					ActiveGate: activegate.Status{
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
