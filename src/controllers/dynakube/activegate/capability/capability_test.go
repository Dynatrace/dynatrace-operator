package capability

import (
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
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

var capabilities = []dynatracev1.CapabilityDisplayName{
	dynatracev1.RoutingCapability.DisplayName,
	dynatracev1.KubeMonCapability.DisplayName,
	dynatracev1.MetricsIngestCapability.DisplayName,
	dynatracev1.DynatraceApiCapability.DisplayName,
}

func buildDynakube(capabilities []dynatracev1.CapabilityDisplayName) *dynatracev1.DynaKube {
	return &dynatracev1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace, Name: testName,
		},
		Spec: dynatracev1.DynaKubeSpec{
			APIURL: testApiUrl,
			ActiveGate: dynatracev1.ActiveGateSpec{
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
		dynakube := buildDynakube(capabilities)
		mc := NewMultiCapability(dynakube)
		require.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgName, mc.ArgName())
	})
	t.Run(`creates new multicapability without capabilities set in dynakube`, func(t *testing.T) {
		var emptyCapabilites []dynatracev1.CapabilityDisplayName
		dynakube := buildDynakube(emptyCapabilites)
		mc := NewMultiCapability(dynakube)
		require.NotNil(t, mc)
		assert.False(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, "", mc.ArgName())
	})
}
