package databases

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/integrationtests"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReconciler(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	integrationtests.CreateNamespace(t, t.Context(), clt, testNamespaceName)

	t.Run("apply deployments", func(t *testing.T) {
		dk := getTestDynakube()
		integrationtests.CreateDynakube(t, t.Context(), clt, dk)

		deployment := getReconciledDeployment(t, clt, dk)
		require.True(t, meta.IsStatusConditionTrue(dk.Status.Conditions, conditionType))
		require.NotNil(t, deployment)
	})

	t.Run("delete deployments", func(t *testing.T) {
		dk := getTestDynakube()
		integrationtests.CreateKubernetesObject(t, t.Context(), clt, getMatchingDeployment(dk))

		dk.Spec.Extensions.Databases = nil
		conditions.SetDeploymentsApplied(dk, conditionType, []string{"test"})
		integrationtests.CreateDynakube(t, t.Context(), clt, dk)

		deployment := getReconciledDeployment(t, clt, dk)
		require.Nil(t, meta.FindStatusCondition(dk.Status.Conditions, conditionType))
		require.Nil(t, deployment)
	})

	t.Run("use existing replicas", func(t *testing.T) {
		dk := getTestDynakube()
		origDeployment := getMatchingDeployment(dk)
		// Use non-default (1) value
		origDeployment.Spec.Replicas = ptr.To(int32(2))
		integrationtests.CreateKubernetesObject(t, t.Context(), clt, origDeployment)

		dk.Spec.Extensions.Databases[0].Replicas = nil
		integrationtests.CreateDynakube(t, t.Context(), clt, dk)

		deployment := getReconciledDeployment(t, clt, dk)
		require.True(t, meta.IsStatusConditionTrue(dk.Status.Conditions, conditionType))
		require.NotNil(t, deployment)
		require.Equal(t, origDeployment.Spec.Replicas, deployment.Spec.Replicas)
	})

	t.Run("use default replicas", func(t *testing.T) {
		dk := getTestDynakube()

		dk.Spec.Extensions.Databases[0].Replicas = nil
		integrationtests.CreateDynakube(t, t.Context(), clt, dk)

		deployment := getReconciledDeployment(t, clt, dk)
		require.True(t, meta.IsStatusConditionTrue(dk.Status.Conditions, conditionType))
		require.NotNil(t, deployment)
		require.Equal(t, ptr.To(int32(1)), deployment.Spec.Replicas)
	})
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
}

func createDynakubeWithDatabaseID(t *testing.T, clt client.Client, id string) error {
	dk := getTestDynakube()

	dk.Name = rand.String(10)
	dk.Spec = dynakube.DynaKubeSpec{
		Extensions: &extensions.Spec{Databases: []extensions.DatabaseSpec{
			{ID: id},
		}},
	}

	return clt.Create(t.Context(), dk)
}

func createDynakubeWithDatabaseSpec(t *testing.T, clt client.Client, databases []extensions.DatabaseSpec) error {
	dk := getTestDynakube()

	dk.Name = rand.String(10)
	dk.Spec = dynakube.DynaKubeSpec{
		Extensions: &extensions.Spec{Databases: databases},
	}

	return clt.Create(t.Context(), dk)
}
