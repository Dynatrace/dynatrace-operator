package k8sdaemonset

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var daemonSetLog = logd.Get().WithName("test-daemonset")

func TestCreateOrUpdateDaemonSet(t *testing.T) {
	const namespaceName = "dynatrace"

	const daemonsetName = "my-daemonset"

	ctx := context.Background()

	t.Run("create when not exists", func(t *testing.T) {
		fakeClient := fake.NewClient()
		annotations := map[string]string{hasher.AnnotationHash: "hash"}
		daemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, annotations, nil)

		created, err := Query(fakeClient, fakeClient, daemonSetLog).CreateOrUpdate(ctx, &daemonSet)
		require.NoError(t, err)
		require.True(t, created)

		ds, err := Query(fakeClient, fakeClient, daemonSetLog).Get(ctx, client.ObjectKeyFromObject(&daemonSet))
		require.NoError(t, err)
		assert.NotEmpty(t, ds)
	})
	t.Run("update when exists and changed", func(t *testing.T) {
		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldDaemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, oldAnnotations, nil)
		newAnnotations := map[string]string{hasher.AnnotationHash: "new"}
		newDaemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, newAnnotations, nil)
		fakeClient := fake.NewClient(&oldDaemonSet)

		updated, err := Query(fakeClient, fakeClient, daemonSetLog).CreateOrUpdate(ctx, &newDaemonSet)
		require.NoError(t, err)
		require.True(t, updated)

		ds, err := Query(fakeClient, fakeClient, daemonSetLog).Get(ctx, client.ObjectKeyFromObject(&newDaemonSet))
		require.NoError(t, err)
		assert.Equal(t, newAnnotations, ds.Annotations)
	})
	t.Run("not update when exists and no changed", func(t *testing.T) {
		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldDaemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, oldAnnotations, nil)

		fakeClient := fake.NewClient(&oldDaemonSet)

		updated, err := Query(fakeClient, fakeClient, daemonSetLog).CreateOrUpdate(ctx, &oldDaemonSet)
		require.NoError(t, err)
		require.False(t, updated)

		ds, err := Query(fakeClient, fakeClient, daemonSetLog).Get(ctx, client.ObjectKeyFromObject(&oldDaemonSet))
		require.NoError(t, err)
		assert.Equal(t, oldDaemonSet, *ds)
	})
	t.Run("recreate when exists and changed for immutable field", func(t *testing.T) {
		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldMatchLabels := map[string]string{"match": "old"}
		oldDaemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, oldAnnotations, oldMatchLabels)

		newAnnotations := map[string]string{hasher.AnnotationHash: "new"}
		newMatchLabels := map[string]string{"match": "new"}
		newDaemonSet := createTestDaemonSetWithMatchLabels(daemonsetName, namespaceName, newAnnotations, newMatchLabels)
		fakeClient := fake.NewClient(&oldDaemonSet)

		recreate, err := Query(fakeClient, fakeClient, daemonSetLog).CreateOrUpdate(ctx, &newDaemonSet)
		require.NoError(t, err)
		require.True(t, recreate)

		ds, err := Query(fakeClient, fakeClient, daemonSetLog).Get(ctx, client.ObjectKeyFromObject(&newDaemonSet))
		require.NoError(t, err)
		assert.Equal(t, newDaemonSet, *ds)
		assert.Equal(t, newMatchLabels, ds.Spec.Selector.MatchLabels)
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
