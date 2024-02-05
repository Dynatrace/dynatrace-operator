package deployment

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var deploymentLog = logger.Factory.GetLogger("test-deployment")
var daemonSetLog = logger.Factory.GetLogger("test-daemonset")

func createTestDeploymentWithMatchLabels(name, namespace string, annotations, matchLabels map[string]string) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
	}
}

// GetDeployment returns the Deployment object who is the owner of this pod.
func TestGetDeployment(t *testing.T) {
	const ns = "dynatrace"

	trueVar := true

	fakeClient := fake.NewClient(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mypod",
				Namespace: ns,
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "ReplicaSet", Name: "myreplicaset", Controller: &trueVar},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myreplicaset",
				Namespace: ns,
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "mydeployment", Controller: &trueVar},
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydeployment",
				Namespace: ns,
			},
		})

	deploy, err := GetDeployment(fakeClient, "mypod", "dynatrace")
	require.NoError(t, err)
	assert.Equal(t, "mydeployment", deploy.Name)
	assert.Equal(t, "dynatrace", deploy.Namespace)
}

func TestCreateOrUpdateDeployment(t *testing.T) {
	const namespaceName = "dynatrace"

	const deploymentName = "my-deployment"

	t.Run("create when not exists", func(t *testing.T) {
		fakeClient := fake.NewClient()
		annotations := map[string]string{hasher.AnnotationHash: "hash"}
		depl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, annotations, nil)
		created, err := CreateOrUpdateDeployment(fakeClient, deploymentLog, &depl)

		require.NoError(t, err)
		assert.True(t, created)
	})

	t.Run("update when exists and changed", func(t *testing.T) {
		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldDepl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, oldAnnotations, nil)
		newAnnotations := map[string]string{hasher.AnnotationHash: "new"}
		newDepl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, newAnnotations, nil)
		fakeClient := fake.NewClient(&oldDepl)

		updated, err := CreateOrUpdateDeployment(fakeClient, deploymentLog, &newDepl)

		require.NoError(t, err)
		assert.True(t, updated)
	})
	t.Run("not update when exists and no changed", func(t *testing.T) {
		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldDepl := createTestDeploymentWithMatchLabels(deploymentName, namespaceName, oldAnnotations, nil)

		fakeClient := fake.NewClient(&oldDepl)

		updated, err := CreateOrUpdateDeployment(fakeClient, deploymentLog, &oldDepl)
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

		updated, err := CreateOrUpdateDeployment(fakeClient, daemonSetLog, &newDepl)

		require.NoError(t, err)
		assert.True(t, updated)

		var actualDepl appsv1.Deployment
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: deploymentName, Namespace: namespaceName}, &actualDepl)
		require.NoError(t, err)
		assert.Equal(t, newMatchLabels, actualDepl.Spec.Selector.MatchLabels)
	})
}
