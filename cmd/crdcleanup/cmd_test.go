package crdcleanup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestGetLatestStorageVersion(t *testing.T) {
	t.Run("returns storage version when marked as storage", func(t *testing.T) {
		crd := &apiextensionsv1.CustomResourceDefinition{
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1alpha1",
						Storage: false,
					},
					{
						Name:    "v1beta1",
						Storage: true,
					},
					{
						Name:    "v1beta2",
						Storage: false,
					},
				},
			},
		}

		latestVersion := getLatestStorageVersion(crd)
		assert.Equal(t, "v1beta1", latestVersion)
	})

	t.Run("returns empty string when no storage version is marked", func(t *testing.T) {
		crd := &apiextensionsv1.CustomResourceDefinition{
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1alpha1",
						Storage: false,
					},
					{
						Name:    "v1beta1",
						Storage: false,
					},
				},
			},
		}

		latestVersion := getLatestStorageVersion(crd)
		assert.Equal(t, "", latestVersion)
	})

	t.Run("returns empty string for empty versions list", func(t *testing.T) {
		crd := &apiextensionsv1.CustomResourceDefinition{
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{},
			},
		}

		latestVersion := getLatestStorageVersion(crd)
		assert.Equal(t, "", latestVersion)
	})
}

func TestNew(t *testing.T) {
	t.Run("creates command with correct use", func(t *testing.T) {
		cmd := New()
		require.NotNil(t, cmd)
		assert.Equal(t, use, cmd.Use)
	})

	t.Run("has namespace flag", func(t *testing.T) {
		cmd := New()
		require.NotNil(t, cmd)

		flag := cmd.PersistentFlags().Lookup(namespaceFlagName)
		require.NotNil(t, flag)
		assert.Equal(t, namespaceFlagShorthand, flag.Shorthand)
	})
}
