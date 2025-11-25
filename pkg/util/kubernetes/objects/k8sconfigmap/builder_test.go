package k8sconfigmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigMapBuilder(t *testing.T) {
	dataKey := "cfg"
	data := map[string]string{
		dataKey: "",
	}

	t.Run("create config map", func(t *testing.T) {
		configMap, err := Build(createDeployment(),
			testConfigMapName,
			nil,
			setNamespace(testNamespace),
		)
		require.NoError(t, err)
		require.Len(t, configMap.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, configMap.OwnerReferences[0].Name)
		assert.Equal(t, testConfigMapName, configMap.Name)
		assert.Empty(t, configMap.Data)
	})
	t.Run("create config map with data", func(t *testing.T) {
		configMap, err := Build(createDeployment(),
			testConfigMapName,
			data,
			setNamespace(testNamespace),
		)
		require.NoError(t, err)
		require.Len(t, configMap.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, configMap.OwnerReferences[0].Name)
		assert.Equal(t, testConfigMapName, configMap.Name)
		assert.Contains(t, configMap.Data, dataKey)
	})
}
