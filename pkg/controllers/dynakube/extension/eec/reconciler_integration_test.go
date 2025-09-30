package eec

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestStatefulSet(t *testing.T) {
	t.Log("WELCOME")

	clt := integrationtests.SetupTestEnvironment(t)

	dk := getTestDynakube()

	integrationtests.CreateNamespace(t, t.Context(), clt, testNamespaceName)
	integrationtests.CreateDynakube(t, t.Context(), clt, dk)
	mockTLSSecret(t, clt, dk)

	reconciler := NewReconciler(clt, clt, dk)
	err := reconciler.Reconcile(t.Context())
	require.NoError(t, err)

	dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = true
	err = reconciler.Reconcile(t.Context())
	require.NoError(t, err)

	// stop test environment
	integrationtests.DestroyTestEnvironment(t)
}

func TestExtensionsDatabases(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	integrationtests.CreateNamespace(t, t.Context(), clt, testNamespaceName)

	t.Run("Valid database ID", func(t *testing.T) {
		require.NoError(t, testExtensionsDatabasesID(t, clt, "db"))
		require.NoError(t, testExtensionsDatabasesID(t, clt, "db-1"))
		require.NoError(t, testExtensionsDatabasesID(t, clt, "db-1-1"))
	})
	t.Run("Invalid database ID string pattern", func(t *testing.T) {
		require.Error(t, testExtensionsDatabasesID(t, clt, "-wrong"))
		require.Error(t, testExtensionsDatabasesID(t, clt, "WRONG-1"))
		require.Error(t, testExtensionsDatabasesID(t, clt, "db-"))
		require.Error(t, testExtensionsDatabasesID(t, clt, "db--1"))
	})
	t.Run("Invalid database ID string length", func(t *testing.T) {
		require.Error(t, testExtensionsDatabasesID(t, clt, "super-long-value"))
	})

	// stop test environment
	integrationtests.DestroyTestEnvironment(t)
}

func testExtensionsDatabasesID(t *testing.T, clt client.Client, id string) error {
	dk := getTestDynakube()

	dk.Name = rand.String(10)
	dk.Spec = dynakube.DynaKubeSpec{
		Extensions: &extensions.Spec{DatabasesSpec: []extensions.DatabaseSpec{
			{ID: id},
		}},
		Templates: dynakube.TemplatesSpec{
			ExtensionExecutionController: extensions.ExecutionControllerSpec{
				ImageRef: image.Ref{
					Repository: testEecImageRepository,
					Tag:        testEecImageTag,
				},
			},
		},
	}

	return clt.Create(t.Context(), dk)
}
