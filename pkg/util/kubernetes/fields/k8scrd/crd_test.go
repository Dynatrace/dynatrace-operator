package k8scrd

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckVersion(t *testing.T) {
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

func getTestCRD(name string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
