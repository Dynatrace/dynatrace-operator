package kubeobjects

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testDaemonSetName = "test-name"
	testNamespace     = "test-namespace"
)

func Test_CreateOrUpdateDaemonSet_Create(t *testing.T) {
	log := logger.NewDTLogger()
	dsBefore := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDaemonSetName,
		},
	}
	fakeClient := fake.NewClient()

	result, err := CreateOrUpdateDaemonSet(fakeClient, log, &dsBefore)
	require.NoError(t, err)
	assert.True(t, result)

	dsAfter := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      testDaemonSetName,
	}, dsAfter)
	require.NoError(t, err)
	assert.Equal(t, dsBefore.Name, dsBefore.Name)
	assert.Equal(t, dsBefore.Namespace, dsBefore.Namespace)
}

func Test_CreateOrUpdateDaemonSet_Update(t *testing.T) {
	log := logger.NewDTLogger()
	dsBefore := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDaemonSetName,
			Annotations: map[string]string{
				AnnotationHash: "old",
			},
		},
	}
	fakeClient := fake.NewClient(&dsBefore)

	dsUpdate := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDaemonSetName,
			Annotations: map[string]string{
				AnnotationHash: "new",
			},
		},
	}
	result, err := CreateOrUpdateDaemonSet(fakeClient, log, &dsUpdate)
	require.NoError(t, err)
	assert.True(t, result)

	dsAfter := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      testDaemonSetName,
	}, dsAfter)
	require.NoError(t, err)

	assert.NotNil(t, dsAfter.Annotations)
	annotation, ok := dsAfter.Annotations[AnnotationHash]
	assert.True(t, ok)
	assert.Equal(t, "new", annotation)
}

func Test_CreateOrUpdateDaemonSet_NoUpdateRequired(t *testing.T) {
	log := logger.NewDTLogger()
	dsBefore := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDaemonSetName,
			Annotations: map[string]string{
				AnnotationHash: "same",
			},
		},
	}
	fakeClient := fake.NewClient(&dsBefore)

	dsUpdate := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDaemonSetName,
			Annotations: map[string]string{
				AnnotationHash: "same",
			},
		},
	}
	result, err := CreateOrUpdateDaemonSet(fakeClient, log, &dsUpdate)
	require.NoError(t, err)
	assert.False(t, result)
}
