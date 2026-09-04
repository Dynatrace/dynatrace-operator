// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8scrd

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestIsLatestVersion(t *testing.T) {
	testDynaKubeCRD := getTestCRD(DynaKubeName)
	testEdgeConnectCRD := getTestCRD(EdgeConnectName)

	t.Run("DynaKube version matches", func(t *testing.T) {
		testDynaKubeCRD.Labels = map[string]string{
			k8slabel.AppVersionLabel: "1.2.3",
		}
		t.Setenv(k8senv.AppVersion, "1.2.3")

		result, err := IsLatestVersion(t.Context(), fake.NewClientWithIndex(testDynaKubeCRD), DynaKubeName)
		assert.True(t, result)
		assert.NoError(t, err)
	})
	t.Run("DynaKube version doesn't match", func(t *testing.T) {
		testDynaKubeCRD.Labels = map[string]string{
			k8slabel.AppVersionLabel: "1.2.3",
		}
		t.Setenv(k8senv.AppVersion, "0.0.0-snapshot")

		result, err := IsLatestVersion(t.Context(), fake.NewClientWithIndex(testDynaKubeCRD), DynaKubeName)
		assert.False(t, result)
		assert.NoError(t, err)
	})
	t.Run("EdgeConnect version matches", func(t *testing.T) {
		testEdgeConnectCRD.Labels = map[string]string{
			k8slabel.AppVersionLabel: "1.2.3",
		}
		t.Setenv(k8senv.AppVersion, "1.2.3")

		result, err := IsLatestVersion(t.Context(), fake.NewClientWithIndex(testEdgeConnectCRD), EdgeConnectName)
		assert.True(t, result)
		assert.NoError(t, err)
	})
	t.Run("EdgeConnect version doesn't match", func(t *testing.T) {
		testEdgeConnectCRD.Labels = map[string]string{
			k8slabel.AppVersionLabel: "1.2.3",
		}
		t.Setenv(k8senv.AppVersion, "0.0.0-snapshot")

		result, err := IsLatestVersion(t.Context(), fake.NewClientWithIndex(testEdgeConnectCRD), EdgeConnectName)
		assert.False(t, result)
		assert.NoError(t, err)
	})
	t.Run("Fail if label is missing", func(t *testing.T) {
		testDynaKubeCRD.Labels = map[string]string{}
		t.Setenv(k8senv.AppVersion, "0.0.0-snapshot")

		result, err := IsLatestVersion(t.Context(), fake.NewClientWithIndex(testDynaKubeCRD), DynaKubeName)
		assert.False(t, result)
		assert.Error(t, err)
	})
	t.Run("Fail if env var is missing", func(t *testing.T) {
		testDynaKubeCRD.Labels = map[string]string{
			k8slabel.AppVersionLabel: "1.2.3",
		}

		result, err := IsLatestVersion(t.Context(), fake.NewClientWithIndex(testDynaKubeCRD), DynaKubeName)
		assert.False(t, result)
		assert.Error(t, err)
	})
}

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

		latestVersion := GetLatestStorageVersion(crd)
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

		latestVersion := GetLatestStorageVersion(crd)
		assert.Empty(t, latestVersion)
	})

	t.Run("returns empty string for empty versions list", func(t *testing.T) {
		crd := &apiextensionsv1.CustomResourceDefinition{
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{},
			},
		}

		latestVersion := GetLatestStorageVersion(crd)
		assert.Empty(t, latestVersion)
	})
}

func TestIsInstalled(t *testing.T) {
	testGVK := schema.GroupVersionKind{Group: "networking.istio.io", Version: "v1beta1", Kind: "VirtualService"}

	createErrorClient := func(kindMissing bool) client.Client {
		return fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				if kindMissing {
					return new(meta.NoResourceMatchError)
				}

				return errors.New("BOOM")
			},
		})
	}

	t.Run("CRD is installed => returns true", func(t *testing.T) {
		fakeClient := fake.NewClient()

		installed := IsInstalled(t.Context(), fakeClient, testGVK)
		assert.True(t, installed)
	})

	t.Run("CRD is not installed => returns false", func(t *testing.T) {
		fakeClient := createErrorClient(true)

		installed := IsInstalled(t.Context(), fakeClient, testGVK)
		assert.False(t, installed)
	})

	t.Run("unknown client err => returns true (no discovery fail == CRD is present)", func(t *testing.T) {
		fakeClient := createErrorClient(false)

		installed := IsInstalled(t.Context(), fakeClient, testGVK)
		assert.True(t, installed)
	})
}

func getTestCRD(name string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
