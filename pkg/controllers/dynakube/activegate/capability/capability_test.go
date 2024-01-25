package capability

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
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

var capabilities = []dynatracev1beta1.CapabilityDisplayName{
	dynatracev1beta1.RoutingCapability.DisplayName,
	dynatracev1beta1.KubeMonCapability.DisplayName,
	dynatracev1beta1.MetricsIngestCapability.DisplayName,
	dynatracev1beta1.DynatraceApiCapability.DisplayName,
}

func buildDynakube(capabilities []dynatracev1beta1.CapabilityDisplayName) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace, Name: testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
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
		var emptyCapabilites []dynatracev1beta1.CapabilityDisplayName
		dynakube := buildDynakube(emptyCapabilites)
		mc := NewMultiCapability(dynakube)
		require.NotNil(t, mc)
		assert.False(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, "", mc.ArgName())
	})
}

func TestBuildServiceHostNameForDNSEntryPoint(t *testing.T) {
	actual := BuildServiceHostName("test-name", "test-component-feature")
	assert.NotEmpty(t, actual)

	expected := "$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_HOST):$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_PORT)"
	assert.Equal(t, expected, actual)

	testStringName := "this---test_string"
	testStringFeature := "SHOULD--_--PaRsEcORrEcTlY"
	expected = "$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_HOST):$(THIS___TEST_STRING_SHOULD_____PARSECORRECTLY_SERVICE_PORT)"
	actual = BuildServiceHostName(testStringName, testStringFeature)
	assert.Equal(t, expected, actual)
}

func TestBuildServiceDomainNameForDNSEntryPoint(t *testing.T) {
	actual := BuildServiceDomainName("test-name", "test-namespace", "test-component-feature")
	assert.NotEmpty(t, actual)

	expected := "test-name-test-component-feature.test-namespace:$(TEST_NAME_TEST_COMPONENT_FEATURE_SERVICE_PORT)"
	assert.Equal(t, expected, actual)

	testStringName := "this---dynakube_string"
	testNamespace := "this_is---namespace_string"
	testStringFeature := "SHOULD--_--PaRsEcORrEcTlY"
	expected = "this---dynakube_string-SHOULD--_--PaRsEcORrEcTlY.this_is---namespace_string:$(THIS___DYNAKUBE_STRING_SHOULD_____PARSECORRECTLY_SERVICE_PORT)"
	actual = BuildServiceDomainName(testStringName, testNamespace, testStringFeature)
	assert.Equal(t, expected, actual)
}
