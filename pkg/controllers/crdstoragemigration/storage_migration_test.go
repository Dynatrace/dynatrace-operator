package crdstoragemigration

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestRun(t *testing.T) {
	ctx := context.Background()
	testNamespace := "test-namespace"

	t.Run("returns no error when CRD not found", func(t *testing.T) {
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				return errors.New("unexpected write operation")
			},
		})
		err := Run(ctx, fakeClient, fakeClient, testNamespace)

		require.NoError(t, err)
	})

	t.Run("returns no error when CRD has no storage versions", func(t *testing.T) {
		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkCrdName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "dynatrace.com",
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: "dynakubes",
					Kind:   "DynaKube",
				},
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1beta1",
						Storage: true,
					},
				},
			},
			Status: apiextensionsv1.CustomResourceDefinitionStatus{
				StoredVersions: []string{},
			},
		}
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				return errors.New("unexpected write operation")
			},
		}, crd)

		err := Run(ctx, fakeClient, fakeClient, testNamespace)

		require.NoError(t, err)
	})

	t.Run("returns no error when CRD has single up-to-date storage version", func(t *testing.T) {
		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkCrdName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "dynatrace.com",
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: "dynakubes",
					Kind:   "DynaKube",
				},
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1beta1",
						Storage: true,
					},
				},
			},
			Status: apiextensionsv1.CustomResourceDefinitionStatus{
				StoredVersions: []string{"v1beta1"},
			},
		}
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				return errors.New("unexpected write operation")
			},
		}, crd)
		err := Run(ctx, fakeClient, fakeClient, testNamespace)

		require.NoError(t, err)
	})

	t.Run("returns error when no storage version is found", func(t *testing.T) {
		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkCrdName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "dynatrace.com",
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: "dynakubes",
					Kind:   "DynaKube",
				},
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1beta1",
						Storage: false,
					},
				},
			},
			Status: apiextensionsv1.CustomResourceDefinitionStatus{
				StoredVersions: []string{"v1beta1", "v1beta2"},
			},
		}
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				return errors.New("unexpected write operation")
			},
		}, crd)
		err := Run(ctx, fakeClient, fakeClient, testNamespace)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to determine target storage version")
	})

	t.Run("migrates DynaKube instances when multiple storage versions exist", func(t *testing.T) {
		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkCrdName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "dynatrace.com",
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: "dynakubes",
					Kind:   "DynaKube",
				},
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1beta1",
						Storage: false,
					},
					{
						Name:    "v1beta2",
						Storage: true,
					},
				},
			},
			Status: apiextensionsv1.CustomResourceDefinitionStatus{
				StoredVersions: []string{"v1beta1", "v1beta2"},
			},
		}

		// Create DynaKube instances using unstructured
		dk1 := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "dynatrace.com/v1beta2",
				"kind":       "DynaKube",
				"metadata": map[string]any{
					"name":      "dynakube-1",
					"namespace": testNamespace,
				},
				"spec": map[string]any{},
			},
		}

		dk2 := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "dynatrace.com/v1beta2",
				"kind":       "DynaKube",
				"metadata": map[string]any{
					"name":      "dynakube-2",
					"namespace": testNamespace,
				},
				"spec": map[string]any{},
			},
		}

		fakeClient := fake.NewClient(crd, dk1, dk2)
		err := Run(ctx, fakeClient, fakeClient, testNamespace)

		require.NoError(t, err)

		// Verify CRD status was updated
		var updatedCRD apiextensionsv1.CustomResourceDefinition
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dkCrdName}, &updatedCRD)
		require.NoError(t, err)
		assert.Equal(t, []string{"v1beta2"}, updatedCRD.Status.StoredVersions)
	})

	t.Run("handles empty DynaKube list", func(t *testing.T) {
		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: dkCrdName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "dynatrace.com",
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: "dynakubes",
					Kind:   "DynaKube",
				},
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1beta1",
						Storage: false,
					},
					{
						Name:    "v1beta2",
						Storage: true,
					},
				},
			},
			Status: apiextensionsv1.CustomResourceDefinitionStatus{
				StoredVersions: []string{"v1beta1", "v1beta2"},
			},
		}

		fakeClient := fake.NewClient(crd)
		err := Run(ctx, fakeClient, fakeClient, testNamespace)

		require.NoError(t, err)

		// Verify CRD status was updated even without DynaKubes
		var updatedCRD apiextensionsv1.CustomResourceDefinition
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dkCrdName}, &updatedCRD)
		require.NoError(t, err)
		assert.Equal(t, []string{"v1beta2"}, updatedCRD.Status.StoredVersions)
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
		assert.Empty(t, latestVersion)
	})

	t.Run("returns empty string for empty versions list", func(t *testing.T) {
		crd := &apiextensionsv1.CustomResourceDefinition{
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{},
			},
		}

		latestVersion := getLatestStorageVersion(crd)
		assert.Empty(t, latestVersion)
	})
}
