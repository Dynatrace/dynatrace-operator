package k8sdeployment

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateOrUpdateDeployment(t *testing.T) {
	const namespaceName = "dynatrace"

	const deploymentName = "my-deployment"

	ctx := context.Background()

	t.Run("create when not exists", func(t *testing.T) {
		fakeClient := fake.NewClient()
		annotations := map[string]string{hasher.AnnotationHash: "hash"}
		depl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, annotations, nil)

		created, err := Query(fakeClient, fakeClient, deploymentLog).CreateOrUpdate(ctx, &depl)

		require.NoError(t, err)
		assert.True(t, created)
	})

	t.Run("update when exists and changed", func(t *testing.T) {
		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldDepl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, oldAnnotations, nil)
		newAnnotations := map[string]string{hasher.AnnotationHash: "new"}
		newDepl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, newAnnotations, nil)
		fakeClient := fake.NewClient(&oldDepl)

		updated, err := Query(fakeClient, fakeClient, deploymentLog).CreateOrUpdate(ctx, &newDepl)

		require.NoError(t, err)
		assert.True(t, updated)
	})
	t.Run("not update when exists and no changed", func(t *testing.T) {
		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldDepl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, oldAnnotations, nil)

		fakeClient := fake.NewClient(&oldDepl)

		updated, err := Query(fakeClient, fakeClient, deploymentLog).CreateOrUpdate(ctx, &oldDepl)
		require.NoError(t, err)
		assert.False(t, updated)
	})
	t.Run("recreate when exists and changed for immutable field", func(t *testing.T) {
		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldMatchLabels := map[string]string{"match": "old"}
		oldDepl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, oldAnnotations, oldMatchLabels)

		newAnnotations := map[string]string{hasher.AnnotationHash: "new"}
		newMatchLabels := map[string]string{"match": "new"}
		newDepl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, newAnnotations, newMatchLabels)
		fakeClient := fake.NewClient(&oldDepl)

		updated, err := Query(fakeClient, fakeClient, deploymentLog).CreateOrUpdate(ctx, &newDepl)

		require.NoError(t, err)
		assert.True(t, updated)

		var actualDepl appsv1.Deployment
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: deploymentName, Namespace: namespaceName}, &actualDepl)
		require.NoError(t, err)
		assert.Equal(t, newMatchLabels, actualDepl.Spec.Selector.MatchLabels)
	})

	t.Run("update will not remove owner reference", func(t *testing.T) {
		fakeClient := fake.NewClient()
		matchLabels := map[string]string{"match": "new"}
		dummyOwner := createTestDeploymentWithMatchLabels("owner", namespaceName, nil, nil)

		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldDepl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, oldAnnotations, matchLabels)

		created, err := Query(fakeClient, fakeClient, deploymentLog).WithOwner(&dummyOwner).CreateOrUpdate(ctx, &oldDepl)
		require.NoError(t, err)
		assert.True(t, created)

		actual, err := Query(fakeClient, fakeClient, deploymentLog).Get(ctx, client.ObjectKeyFromObject(&oldDepl))
		require.NoError(t, err)
		assert.NotEmpty(t, actual.OwnerReferences)

		newAnnotations := map[string]string{hasher.AnnotationHash: "new"}
		newDepl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, newAnnotations, matchLabels)

		updated, err := Query(fakeClient, fakeClient, deploymentLog).WithOwner(&dummyOwner).CreateOrUpdate(ctx, &newDepl)
		require.NoError(t, err)
		assert.True(t, updated)

		actual, err = Query(fakeClient, fakeClient, deploymentLog).Get(ctx, client.ObjectKeyFromObject(&newDepl))
		require.NoError(t, err)
		assert.NotEmpty(t, actual.OwnerReferences)
		assert.Equal(t, matchLabels, actual.Spec.Selector.MatchLabels)
	})
}
