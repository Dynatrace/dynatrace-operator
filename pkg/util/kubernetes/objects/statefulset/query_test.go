package statefulset

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

var statefulSetLog = logd.Get().WithName("test-statefulset")

func TestCreateOrUpdateStatefulSet(t *testing.T) {
	const namespaceName = "dynatrace"

	const statefulSetName = "my-daemonset"

	ctx := context.Background()

	t.Run("create when not exists", func(t *testing.T) {
		fakeClient := fake.NewClient()
		annotations := map[string]string{hasher.AnnotationHash: "hash"}
		daemonSet := createTestStatefulSetWithMatchLabels(statefulSetName, namespaceName, annotations, nil)

		created, err := Query(fakeClient, fakeClient, statefulSetLog).CreateOrUpdate(ctx, &daemonSet)
		require.NoError(t, err)
		require.True(t, created)

		ds, err := Query(fakeClient, fakeClient, statefulSetLog).Get(ctx, client.ObjectKeyFromObject(&daemonSet))
		require.NoError(t, err)
		assert.NotEmpty(t, ds)
	})
	t.Run("update when exists and changed", func(t *testing.T) {
		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldStatefulSet := createTestStatefulSetWithMatchLabels(statefulSetName, namespaceName, oldAnnotations, nil)
		newAnnotations := map[string]string{hasher.AnnotationHash: "new"}
		newStatefulSet := createTestStatefulSetWithMatchLabels(statefulSetName, namespaceName, newAnnotations, nil)
		fakeClient := fake.NewClient(&oldStatefulSet)

		updated, err := Query(fakeClient, fakeClient, statefulSetLog).CreateOrUpdate(ctx, &newStatefulSet)
		require.NoError(t, err)
		require.True(t, updated)

		ds, err := Query(fakeClient, fakeClient, statefulSetLog).Get(ctx, client.ObjectKeyFromObject(&newStatefulSet))
		require.NoError(t, err)
		assert.Equal(t, newAnnotations, ds.Annotations)
	})
	t.Run("not update when exists and no changed", func(t *testing.T) {
		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldStatefulSet := createTestStatefulSetWithMatchLabels(statefulSetName, namespaceName, oldAnnotations, nil)

		fakeClient := fake.NewClient(&oldStatefulSet)

		updated, err := Query(fakeClient, fakeClient, statefulSetLog).CreateOrUpdate(ctx, &oldStatefulSet)
		require.NoError(t, err)
		require.False(t, updated)

		ds, err := Query(fakeClient, fakeClient, statefulSetLog).Get(ctx, client.ObjectKeyFromObject(&oldStatefulSet))
		require.NoError(t, err)
		assert.Equal(t, oldStatefulSet, *ds)
	})
	t.Run("recreate when exists and changed for immutable field", func(t *testing.T) {
		oldAnnotations := map[string]string{hasher.AnnotationHash: "old"}
		oldMatchLabels := map[string]string{"match": "old"}
		oldStatefulSet := createTestStatefulSetWithMatchLabels(statefulSetName, namespaceName, oldAnnotations, oldMatchLabels)

		newAnnotations := map[string]string{hasher.AnnotationHash: "new"}
		newMatchLabels := map[string]string{"match": "new"}
		newStatefulSet := createTestStatefulSetWithMatchLabels(statefulSetName, namespaceName, newAnnotations, newMatchLabels)
		fakeClient := fake.NewClient(&oldStatefulSet)

		recreate, err := Query(fakeClient, fakeClient, statefulSetLog).CreateOrUpdate(ctx, &newStatefulSet)
		require.NoError(t, err)
		require.True(t, recreate)

		ds, err := Query(fakeClient, fakeClient, statefulSetLog).Get(ctx, client.ObjectKeyFromObject(&newStatefulSet))
		require.NoError(t, err)
		assert.Equal(t, newStatefulSet, *ds)
		assert.Equal(t, newMatchLabels, ds.Spec.Selector.MatchLabels)
	})
}

func createTestStatefulSetWithMatchLabels(name, namespace string, annotations, matchLabels map[string]string) appsv1.StatefulSet {
	return appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
	}
}
