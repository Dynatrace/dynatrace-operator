package capability

import (
	"encoding/json"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testApiUrl = "https://demo.dev.dynatracelabs.com/api"

func testBuildDynaKubeWithAnnotations(instanceName string, statsdEnabled bool, annotations map[string]string) *dynatracev1beta1.DynaKube {
	var capabilities []dynatracev1beta1.CapabilityDisplayName
	if statsdEnabled {
		capabilities = append(capabilities, dynatracev1beta1.StatsdIngestCapability.DisplayName)
	}

	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instanceName,
			Namespace:   "dynatrace",
			Annotations: annotations,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: capabilities,
			},
		},
	}
}

func TestCreateEecConfigMap(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		instance := testBuildDynaKubeWithAnnotations("dynakube", true, map[string]string{
			"internal.operator.dynatrace.com/extensions.debugExtensionDSstatsddisablenamedalivesignals": "false",
			"internal.operator.dynatrace.com/extensions.debugExtensionDSstatsdlogoutboundminttraffic":   "true",
			"internal.operator.dynatrace.com/extensions.debugExtensionDSstatsdcustomloglevel":           "trace",
		})
		runtimeConfig := make(map[string]interface{})

		eecConfigMap := CreateEecConfigMap(instance, "activegate")
		runtimeConfigJson := eecConfigMap.Data["runtimeConfiguration"]

		assert.Equal(t, "dynakube-activegate-eec-config", eecConfigMap.Name)

		require.NotEmpty(t, eecConfigMap.Data)
		require.NoError(t, json.Unmarshal([]byte(runtimeConfigJson), &runtimeConfig))
		assert.Equal(t, 1., runtimeConfig["revision"])
		assert.True(t, runtimeConfig["booleanMap"].(map[string]interface{})["debugExtensionDSstatsdlogoutboundminttraffic"].(bool))
		assert.False(t, runtimeConfig["booleanMap"].(map[string]interface{})["debugExtensionDSstatsddisablenamedalivesignals"].(bool))
		assert.Equal(t, "trace", runtimeConfig["stringMap"].(map[string]interface{})["debugExtensionDSstatsdcustomloglevel"])
		assert.Empty(t, runtimeConfig["longMap"].(map[string]interface{}))
	})

	t.Run("no valid EEC runtime properties, StatsD enabled", func(t *testing.T) {
		instance := testBuildDynaKubeWithAnnotations("dynakube", true, map[string]string{
			"internal.operator.dynatrace.com/debugExtensionDSstatsdlogoutboundminttraffic": "true",
			"debugExtensionDSstatsdcustomloglevel":                                         "info",
		})
		runtimeConfig := make(map[string]interface{})

		eecConfigMap := CreateEecConfigMap(instance, "activegate")
		runtimeConfigJson := eecConfigMap.Data["runtimeConfiguration"]

		assert.Equal(t, "dynakube-activegate-eec-config", eecConfigMap.Name)

		require.NotEmpty(t, eecConfigMap.Data)
		require.NoError(t, json.Unmarshal([]byte(runtimeConfigJson), &runtimeConfig))
		assert.Equal(t, 1., runtimeConfig["revision"])
		assert.Empty(t, runtimeConfig["booleanMap"].(map[string]interface{}))
		assert.Empty(t, runtimeConfig["stringMap"].(map[string]interface{}))
		assert.Empty(t, runtimeConfig["longMap"].(map[string]interface{}))
	})

	t.Run("valid EEC runtime properties but StatsD disabled", func(t *testing.T) {
		instance := testBuildDynaKubeWithAnnotations("dynakube", false, map[string]string{
			"internal.operator.dynatrace.com/extensions.debugExtensionDSstatsddisablenamedalivesignals": "false",
			"internal.operator.dynatrace.com/extensions.debugExtensionDSstatsdlogoutboundminttraffic":   "true",
			"internal.operator.dynatrace.com/extensions.debugExtensionDSstatsdcustomloglevel":           "trace",
		})

		eecConfigMap := CreateEecConfigMap(instance, "activegate")
		assert.Nil(t, eecConfigMap)
	})
}
