package kubeobjects

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateOrUpdateDaemonSet(t *testing.T) {
	const namespaceName = "dynatrace"
	const daemonsetName = "my-daemonset"

	t.Run("create when not exists", func(t *testing.T) {
		fakeClient := fake.NewClient()
		annotations := map[string]string{AnnotationHash: "hash"}
		daemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, annotations, nil)

		created, err := CreateOrUpdateDaemonSet(fakeClient, log, &daemonSet)

		require.NoError(t, err)
		assert.True(t, created)
	})
	t.Run("update when exists and changed", func(t *testing.T) {
		oldAnnotations := map[string]string{AnnotationHash: "old"}
		oldDaemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, oldAnnotations, nil)
		newAnnotations := map[string]string{AnnotationHash: "new"}
		newDaemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, newAnnotations, nil)
		fakeClient := fake.NewClient(&oldDaemonSet)

		updated, err := CreateOrUpdateDaemonSet(fakeClient, log, &newDaemonSet)

		require.NoError(t, err)
		assert.True(t, updated)
	})
	t.Run("not update when exists and no changed", func(t *testing.T) {
		oldAnnotations := map[string]string{AnnotationHash: "old"}
		oldDaemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, oldAnnotations, nil)

		fakeClient := fake.NewClient(&oldDaemonSet)

		updated, err := CreateOrUpdateDaemonSet(fakeClient, log, &oldDaemonSet)
		require.NoError(t, err)
		assert.False(t, updated)
	})
	t.Run("recreate when exists and changed for immutable field", func(t *testing.T) {
		oldAnnotations := map[string]string{AnnotationHash: "old"}
		oldMatchLabels := map[string]string{"match": "old"}
		oldDaemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, oldAnnotations, oldMatchLabels)

		newAnnotations := map[string]string{AnnotationHash: "new"}
		newMatchLabels := map[string]string{"match": "new"}
		newDaemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, newAnnotations, newMatchLabels)
		fakeClient := fake.NewClient(&oldDaemonSet)

		updated, err := CreateOrUpdateDaemonSet(fakeClient, log, &newDaemonSet)

		require.NoError(t, err)
		assert.True(t, updated)
		var actualDaemonSet appsv1.DaemonSet
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: daemonsetName, Namespace: namespaceName}, &actualDaemonSet)
		require.NoError(t, err)
		assert.Equal(t, newMatchLabels, actualDaemonSet.Spec.Selector.MatchLabels)
	})
}

func createTestDaemonSetWithMatchLabels(name, namespace string, annotations, matchLabels map[string]string) appsv1.DaemonSet {
	return appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
	}
}
