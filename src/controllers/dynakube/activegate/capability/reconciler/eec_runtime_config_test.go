package reconciler

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
			dynatracev1beta1.AnnotationFeaturePrefix + "extensions.debugExtensionDSstatsddisablenamedalivesignals": "false",
			dynatracev1beta1.AnnotationFeaturePrefix + "extensions.debugExtensionDSstatsdlogoutboundminttraffic":   "true",
			dynatracev1beta1.AnnotationFeaturePrefix + "extensions.debugExtensionDSstatsdcustomloglevel":           "trace",
		})
		runtimeConfig := NewEecRuntimeConfig()

		eecConfigMap, err := CreateEecConfigMap(instance, "activegate")
		require.NoError(t, err)
		runtimeConfigJson := eecConfigMap.Data["runtimeConfiguration"]

		assert.Equal(t, "dynakube-activegate-eec-config", eecConfigMap.Name)

		require.NotEmpty(t, eecConfigMap.Data)
		require.NoError(t, json.Unmarshal([]byte(runtimeConfigJson), &runtimeConfig))
		assert.Equal(t, 1, runtimeConfig.Revision)
		assert.True(t, runtimeConfig.BooleanMap["debugExtensionDSstatsdlogoutboundminttraffic"])
		assert.False(t, runtimeConfig.BooleanMap["debugExtensionDSstatsddisablenamedalivesignals"])
		assert.Equal(t, "trace", runtimeConfig.StringMap["debugExtensionDSstatsdcustomloglevel"])
		assert.Empty(t, runtimeConfig.LongMap)
	})

	t.Run("no valid EEC runtime properties, StatsD enabled", func(t *testing.T) {
		instance := testBuildDynaKubeWithAnnotations("dynakube", true, map[string]string{
			dynatracev1beta1.AnnotationFeaturePrefix + "debugExtensionDSstatsdlogoutboundminttraffic": "true",
			"debugExtensionDSstatsdcustomloglevel":                                                    "info",
		})
		runtimeConfig := NewEecRuntimeConfig()

		eecConfigMap, err := CreateEecConfigMap(instance, "activegate")
		require.NoError(t, err)

		assert.Equal(t, "dynakube-activegate-eec-config", eecConfigMap.Name)

		require.NotEmpty(t, eecConfigMap.Data)
		require.NoError(t, json.Unmarshal([]byte(eecConfigMap.Data["runtimeConfiguration"]), &runtimeConfig))

		assert.Equal(t, 1, runtimeConfig.Revision)
		assert.Empty(t, runtimeConfig.BooleanMap)
		assert.Empty(t, runtimeConfig.StringMap)
		assert.Empty(t, runtimeConfig.LongMap)
	})

	t.Run("valid EEC runtime properties but StatsD disabled", func(t *testing.T) {
		instance := testBuildDynaKubeWithAnnotations("dynakube", false, map[string]string{
			dynatracev1beta1.AnnotationFeaturePrefix + "extensions.debugExtensiondummylongflag": "17",
		})
		runtimeConfig := NewEecRuntimeConfig()

		eecConfigMap, err := CreateEecConfigMap(instance, "activegate")
		require.NoError(t, err)
		assert.NotNil(t, eecConfigMap)

		assert.Equal(t, "dynakube-activegate-eec-config", eecConfigMap.Name)

		require.NotEmpty(t, eecConfigMap.Data)
		require.NoError(t, json.Unmarshal([]byte(eecConfigMap.Data["runtimeConfiguration"]), &runtimeConfig))

		assert.Equal(t, 1, runtimeConfig.Revision)
		assert.Empty(t, runtimeConfig.BooleanMap)
		assert.Empty(t, runtimeConfig.StringMap)
		assert.Equal(t, int64(17), runtimeConfig.LongMap["debugExtensiondummylongflag"])
	})
}
