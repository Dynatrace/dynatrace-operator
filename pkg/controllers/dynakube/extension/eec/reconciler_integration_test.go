package eec

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
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
		err := createDynakubeWithDatabaseID(t, clt, "db")
		require.NoError(t, err, "should create dynakube with database ID 'db'")

		err = createDynakubeWithDatabaseID(t, clt, "db-1")
		require.NoError(t, err, "should create dynakube with database ID 'db-1'")

		err = createDynakubeWithDatabaseID(t, clt, "db-1-1")
		require.NoError(t, err, "should create dynakube with database ID 'db-1-1'")
	})
	t.Run("Invalid database ID string pattern", func(t *testing.T) {
		err := createDynakubeWithDatabaseID(t, clt, "-wrong")
		require.Error(t, err, "should not create dynakube with database ID starting with hyphen")

		err = createDynakubeWithDatabaseID(t, clt, "WRONG-1")
		require.Error(t, err, "should not create dynakube with database ID containing uppercase letters")

		err = createDynakubeWithDatabaseID(t, clt, "db-")
		require.Error(t, err, "should not create dynakube with database ID ending with hyphen")

		err = createDynakubeWithDatabaseID(t, clt, "db--1")
		require.Error(t, err, "should not create dynakube with database ID containing consecutive hyphens")
	})
	t.Run("Invalid database ID string length", func(t *testing.T) {
		err := createDynakubeWithDatabaseID(t, clt, "super-long-value")
		require.Error(t, err, "should not create dynakube with database ID longer than 8 characters")
	})
	t.Run("Valid database list length", func(t *testing.T) {
		err := createDynakubeWithDatabaseSpec(t, clt, []extensions.DatabaseSpec{})
		require.NoError(t, err)
		err = createDynakubeWithDatabaseSpec(t, clt, []extensions.DatabaseSpec{
			{ID: "db1"},
		})
		require.NoError(t, err, "should create dynakube with one database")
	})
	t.Run("Invalid database list length", func(t *testing.T) {
		err := createDynakubeWithDatabaseSpec(t, clt, []extensions.DatabaseSpec{
			{ID: "db1"},
			{ID: "db2"},
		})
		require.Error(t, err, "should not create dynakube with more than one database")
	})

	// stop test environment
	integrationtests.DestroyTestEnvironment(t)
}

func createDynakubeWithDatabaseID(t *testing.T, clt client.Client, id string) error {
	dk := getTestDynakube()

	dk.Name = rand.String(10)
	dk.Spec = dynakube.DynaKubeSpec{
		Extensions: &extensions.Spec{DatabasesSpec: []extensions.DatabaseSpec{
			{ID: id},
		}},
	}

	return clt.Create(t.Context(), dk)
}

func createDynakubeWithDatabaseSpec(t *testing.T, clt client.Client, databases []extensions.DatabaseSpec) error {
	dk := getTestDynakube()

	dk.Name = rand.String(10)
	dk.Spec = dynakube.DynaKubeSpec{
		Extensions: &extensions.Spec{DatabasesSpec: databases},
	}

	return clt.Create(t.Context(), dk)
}
