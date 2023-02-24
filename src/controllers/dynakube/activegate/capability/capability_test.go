package capability

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testNamespace     = "test-namespace"
	testName          = "test-name"
	testApiUrl        = "https://demo.dev.dynatracelabs.com/api"
	expectedShortName = "activegate"
	expectedArgName   = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface,synthetic,beacon_forwarder,beacon_forwarder_synthetic"
)

var capabilites = []dynatracev1beta1.CapabilityDisplayName{
	dynatracev1beta1.RoutingCapability.DisplayName,
	dynatracev1beta1.KubeMonCapability.DisplayName,
	dynatracev1beta1.MetricsIngestCapability.DisplayName,
	dynatracev1beta1.DynatraceApiCapability.DisplayName,
	dynatracev1beta1.SyntheticCapability.DisplayName,
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
		expectedProxySecretName := "dynatrace-activegate-internal-proxy"
		actualProxySecretName := BuildProxySecretName()
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
		dk := buildDynakube(capabilites)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgName, mc.ArgName())
	})
	t.Run(`creates new multicapability without capabilities set in dynakube`, func(t *testing.T) {
		var emptyCapabilites []dynatracev1beta1.CapabilityDisplayName
		dk := buildDynakube(emptyCapabilites)
		mc := NewMultiCapability(dk)
		require.NotNil(t, mc)
		assert.False(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, "", mc.ArgName())
	})
}
