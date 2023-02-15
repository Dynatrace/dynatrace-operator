package capability

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace     = "test-namespace"
	testName          = "test-name"
	testApiUrl        = "https://demo.dev.dynatracelabs.com/api"
	expectedShortName = "activegate"
	expectedArgName   = "MSGrouter,kubernetes_monitoring,metrics_ingest,restInterface,synthetic,beacon_forwarder,beacon_forwarder_synthetic"
)

func getTestInstance() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: v1.ObjectMeta{
			Namespace: testNamespace, Name: testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.RoutingCapability.DisplayName,
					dynatracev1beta1.KubeMonCapability.DisplayName,
					dynatracev1beta1.MetricsIngestCapability.DisplayName,
					dynatracev1beta1.DynatraceApiCapability.DisplayName,
					dynatracev1beta1.SyntheticCapability.DisplayName,
				},
			},
		},
	}
}

func getTestInstanceWithoutCapabilites() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: v1.ObjectMeta{
			Namespace: testNamespace, Name: testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL:     testApiUrl,
			ActiveGate: dynatracev1beta1.ActiveGateSpec{},
		},
	}
}

func TestBuildProxySecretName(t *testing.T) {
	t.Run(`test if build works`, func(t *testing.T) {
		expectedProxySecretName := "dynatrace-activegate-internal-proxy"
		actualProxySecretName := BuildProxySecretName()
		assert.NotEmpty(t, actualProxySecretName)
		assert.Equal(t, expectedProxySecretName, actualProxySecretName)
	})
}

func TestBuildServiceName(t *testing.T) {
	t.Run(`test building the service name`, func(t *testing.T) {
		expectedServiceName := "testName-testModule"
		actualServiceName := BuildServiceName("testName", "testModule")
		assert.NotEmpty(t, actualServiceName)
		assert.Equal(t, expectedServiceName, actualServiceName)
	})
}

func TestNewMultiCapability(t *testing.T) {
	t.Run(`test new multicapability`, func(t *testing.T) {
		dk := getTestInstance()
		mc := NewMultiCapability(dk)
		assert.NotNil(t, mc)
		assert.True(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, expectedArgName, mc.ArgName())
	})
	t.Run(`test new multicapability with no capabilities set`, func(t *testing.T) {
		dk := getTestInstanceWithoutCapabilites()
		mc := NewMultiCapability(dk)
		assert.NotNil(t, mc)
		assert.False(t, mc.Enabled())
		assert.Equal(t, expectedShortName, mc.ShortName())
		assert.Equal(t, "", mc.ArgName())
	})
}
