package k8ssecret

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestSecretBuilder(t *testing.T) {
	labelName := "name"
	labelValue := "value"
	labels := map[string]string{
		labelName: labelValue,
	}
	dataKey := ".dockercfg"
	dockerCfg := map[string][]byte{
		dataKey: {},
	}

	t.Run("create secret", func(t *testing.T) {
		secret, err := Build(createDeployment(),
			testSecretName,
			map[string][]byte{},
			setNamespace(testNamespace))
		require.NoError(t, err)
		require.Len(t, secret.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, secret.OwnerReferences[0].Name)
		assert.Equal(t, testSecretName, secret.Name)
		assert.Empty(t, secret.Labels)
		assert.Equal(t, corev1.SecretType(""), secret.Type)
		assert.Empty(t, secret.Data)
	})
	t.Run("create secret with label", func(t *testing.T) {
		secret, err := Build(createDeployment(),
			testSecretName,
			map[string][]byte{},
			SetLabels(labels),
			setNamespace(testNamespace),
		)
		require.NoError(t, err)
		require.Len(t, secret.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, secret.OwnerReferences[0].Name)
		assert.Equal(t, testSecretName, secret.Name)
		require.Len(t, secret.Labels, 1)
		assert.Equal(t, labelValue, secret.Labels[labelName])
		assert.Equal(t, corev1.SecretType(""), secret.Type)
		assert.Empty(t, secret.Data)
	})
	t.Run("create secret with type", func(t *testing.T) {
		secret, err := Build(createDeployment(),
			testSecretName,
			dockerCfg,
			SetType(corev1.SecretTypeDockercfg),
			setNamespace(testNamespace),
		)
		require.NoError(t, err)
		require.Len(t, secret.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, secret.OwnerReferences[0].Name)
		assert.Equal(t, testSecretName, secret.Name)
		assert.Empty(t, secret.Labels)
		assert.Equal(t, corev1.SecretTypeDockercfg, secret.Type)
		assert.Contains(t, secret.Data, dataKey)
	})
	t.Run("create secret with label and type", func(t *testing.T) {
		secret, err := Build(createDeployment(),
			testSecretName,
			dockerCfg,
			SetLabels(labels),
			SetType(corev1.SecretTypeDockercfg),
			setNamespace(testNamespace),
		)
		require.NoError(t, err)
		require.Len(t, secret.OwnerReferences, 1)
		assert.Equal(t, testDeploymentName, secret.OwnerReferences[0].Name)
		assert.Equal(t, testSecretName, secret.Name)
		require.Len(t, secret.Labels, 1)
		assert.Equal(t, labelValue, secret.Labels[labelName])
		assert.Equal(t, corev1.SecretTypeDockercfg, secret.Type)
		assert.Contains(t, secret.Data, dataKey)
	})
}
