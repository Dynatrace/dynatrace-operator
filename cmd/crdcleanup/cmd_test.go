package crdcleanup

import (
	"context"
	"testing"

	latest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	crdcleanupcontroller "github.com/Dynatrace/dynatrace-operator/pkg/controllers/crdcleanup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

func TestPerformCRDCleanup(t *testing.T) {
	ctx := context.Background()
	testNamespace := "dynatrace"

	t.Run("performs cleanup when CRD has multiple storage versions", func(t *testing.T) {
		crd := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: crdcleanupcontroller.DynaKubeCRDName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "dynatrace.com",
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: "dynakubes",
					Kind:   "DynaKube",
				},
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{Name: "v1beta4", Storage: false, Served: true},
					{Name: "v1beta5", Storage: true, Served: true},
				},
			},
			Status: apiextensionsv1.CustomResourceDefinitionStatus{
				StoredVersions: []string{"v1beta4", "v1beta5"},
			},
		}

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}

		dk1 := &latest.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube-1",
				Namespace: testNamespace,
			},
			Spec: latest.DynaKubeSpec{},
		}

		dk2 := &latest.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dynakube-2",
				Namespace: testNamespace,
			},
			Spec: latest.DynaKubeSpec{},
		}

		clt := fake.NewClient(crd, ns, dk1, dk2)

		err := performCRDCleanup(clt, testNamespace)
		require.NoError(t, err)

		// Verify CRD status was updated => has to be the latest storage version only (in difference to the cleanupcrdcontroller, that sets it to the latest compiled version)
		var updatedCRD apiextensionsv1.CustomResourceDefinition
		err = clt.Get(ctx, client.ObjectKey{Name: crdcleanupcontroller.DynaKubeCRDName}, &updatedCRD)
		require.NoError(t, err)
		assert.Equal(t, []string{"v1beta5"}, updatedCRD.Status.StoredVersions)
	})

	t.Run("gracefully handles missing CRD", func(t *testing.T) {
		clt := fake.NewClient()
		err := performCRDCleanup(clt, testNamespace)
		require.NoError(t, err)
	})
}
