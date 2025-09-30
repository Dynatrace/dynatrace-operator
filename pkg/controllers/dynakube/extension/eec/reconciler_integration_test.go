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
		require.NoError(t, createDynakubeWithDatabaseExtension(t, clt, "db"))
		require.NoError(t, createDynakubeWithDatabaseExtension(t, clt, "db-1"))
		require.NoError(t, createDynakubeWithDatabaseExtension(t, clt, "db-1-1"))
	})
	t.Run("Invalid database ID string pattern", func(t *testing.T) {
		err := createDynakubeWithDatabaseExtension(t, clt, "-wrong")
		require.Errorf(t, err, "should not create dynakube with database ID starting with hyphen")

		err = createDynakubeWithDatabaseExtension(t, clt, "WRONG-1")
		require.Errorf(t, err, "should not create dynakube with database ID containing uppercase letters")

		err = createDynakubeWithDatabaseExtension(t, clt, "db-")
		require.Errorf(t, err, "should not create dynakube with database ID ending with hyphen")

		err = createDynakubeWithDatabaseExtension(t, clt, "db--1")
		require.Errorf(t, err, "should not create dynakube with database ID containing consecutive hyphens")
	})
	t.Run("Invalid database ID string length", func(t *testing.T) {
		err := createDynakubeWithDatabaseExtension(t, clt, "super-long-value")
		require.Errorf(t, err, "should not create dynakube with database ID longer than 8 characters")
	})

	// stop test environment
	integrationtests.DestroyTestEnvironment(t)
}

func createDynakubeWithDatabaseExtension(t *testing.T, clt client.Client, id string) error {
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
