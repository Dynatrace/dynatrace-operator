package capability

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace                      = "test-namespace"
	testName                           = "test-name"
	testAPIURL                         = "https://demo.dev.dynatracelabs.com/api"
	expectedShortName                  = "activegate"
	expectedArgName                    = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface"
	expectedArgNameWithDebugging       = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface,debugging"
	expectedArgNameWithExtensions      = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface,extension_controller"
	expectedArgNameWithExtensionsOnly  = "extension_controller"
	expectedArgNameWithTelemetryIngest = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface,log_analytics_collector,generic_ingest,otlp_ingest"
)

var capabilities = []activegate.CapabilityDisplayName{
	activegate.RoutingCapability.DisplayName,
	activegate.KubeMonCapability.DisplayName,
	activegate.MetricsIngestCapability.DisplayName,
	activegate.DynatraceAPICapability.DisplayName,
}

func buildDynakube(capabilities []activegate.CapabilityDisplayName, enableExtensions bool, enableTelemetryIngest bool) *dynakube.DynaKube {
	extensionsSpec := &extensions.Spec{PrometheusSpec: &extensions.PrometheusSpec{}, Databases: []extensions.Database{}}
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
			APIURL: testAPIURL,
			ActiveGate: activegate.Spec{
				Capabilities: capabilities,
			},
			Extensions:      extensionsSpec,
			TelemetryIngest: telemetryIngestSpec,
		},
	}
}

func TestBuildProxySecretName(t *testing.T) {
	t.Run("correct secret name", func(t *testing.T) {
		expectedProxySecretName := "someDK-internal-proxy"
		actualProxySecretName := proxy.BuildSecretName("someDK")
		require.NotEmpty(t, actualProxySecretName)
		assert.Equal(t, expectedProxySecretName, actualProxySecretName)
	})
}

func TestBuildServiceName(t *testing.T) {
	t.Run("build service name", func(t *testing.T) {
		expectedServiceName := "testName-" + consts.MultiActiveGateName
		actualServiceName := BuildServiceName("testName")
		require.NotEmpty(t, actualServiceName)
		assert.Equal(t, expectedServiceName, actualServiceName)
	})
}

func TestNewMultiCapability(t *testing.T) {
	t.Run("creates new multicapability", func(t *testing.T) {
		dk := buildDynakube(capabilities, false, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedArgName, mc.ArgName())
	})
	t.Run("creates new multicapability without capabilities set in dynakube", func(t *testing.T) {
		var emptyCapabilites []activegate.CapabilityDisplayName
		dk := buildDynakube(emptyCapabilites, false, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.False(t, mc.Enabled())
		assert.Empty(t, mc.ArgName())
	})
}

func TestNewMultiCapabilityWithExtensions(t *testing.T) {
	t.Run("creates new multicapability with Extensions enabled", func(t *testing.T) {
		dk := buildDynakube(capabilities, true, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedArgNameWithExtensions, mc.ArgName())
	})
	t.Run("creates new multicapability without capabilities set in dynakube and Extensions enabled", func(t *testing.T) {
		var emptyCapabilites []activegate.CapabilityDisplayName
		dk := buildDynakube(emptyCapabilites, true, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedArgNameWithExtensionsOnly, mc.ArgName())
	})
}

func TestNewMultiCapabilityWithTelemetryIngest(t *testing.T) {
	t.Run("creates new multicapability with TelemetryIngest enabled", func(t *testing.T) {
		dk := buildDynakube(capabilities, false, true)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedArgNameWithTelemetryIngest, mc.ArgName())
	})
	t.Run("creates new multicapability without capabilities set in dynakube and TelemetryIngest enabled", func(t *testing.T) {
		var emptyCapabilites []activegate.CapabilityDisplayName
		dk := buildDynakube(emptyCapabilites, false, true)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.False(t, mc.Enabled())
		assert.Empty(t, mc.ArgName())
	})
}

func TestNewMultiCapabilityWithDebugging(t *testing.T) {
	t.Run("creates new multicapability with debugging capability enabled", func(t *testing.T) {
		dk := buildDynakube(append(capabilities, activegate.DebuggingCapability.DisplayName), false, false)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedArgNameWithDebugging, mc.ArgName())
	})
}

func TestBuildServiceDomainNameForDNSEntryPoint(t *testing.T) {
	actual := buildServiceDomainName("test-name", "test-namespace")
	assert.NotEmpty(t, actual)

	expected := "test-name-activegate.test-namespace:443"
	assert.Equal(t, expected, actual)

	testStringName := "this---dynakube_string"
	testNamespace := "this_is---namespace_string"
	expected = "this---dynakube_string-activegate.this_is---namespace_string:443"
	actual = buildServiceDomainName(testStringName, testNamespace)
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
			dnsEntryPoint := BuildDNSEntryPoint(*test.dk)
			assert.Equal(t, test.expectedDNS, dnsEntryPoint)
		})
	}
}
