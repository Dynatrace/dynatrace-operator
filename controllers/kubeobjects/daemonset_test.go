package kubeobjects

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/controllers/activegate/reconciler/statefulset"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testname      = "test-name"
	testNamespace = "test-namespace"
)

func Test_CreateOrUpdateDaemonSet_Create(t *testing.T) {
	log := logger.NewDTLogger()
	dsBefore := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testname,
		},
	}
	fakeClient := fake.NewClient()

	result, err := CreateOrUpdateDaemonSet(fakeClient, log, &dsBefore)
	require.NoError(t, err)
	assert.True(t, result)

	dsAfter := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      testname,
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
			Name:      testname,
			Annotations: map[string]string{
				statefulset.AnnotationTemplateHash: "old",
			},
		},
	}
	fakeClient := fake.NewClient(&dsBefore)

	dsUpdate := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testname,
			Annotations: map[string]string{
				statefulset.AnnotationTemplateHash: "new",
			},
		},
	}
	result, err := CreateOrUpdateDaemonSet(fakeClient, log, &dsUpdate)
	require.NoError(t, err)
	assert.True(t, result)

	dsAfter := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      testname,
	}, dsAfter)
	require.NoError(t, err)

	assert.NotNil(t, dsAfter.Annotations)
	annotation, ok := dsAfter.Annotations[statefulset.AnnotationTemplateHash]
	assert.True(t, ok)
	assert.Equal(t, "new", annotation)
}

func Test_CreateOrUpdateDaemonSet_NoUpdateRequired(t *testing.T) {
	log := logger.NewDTLogger()
	dsBefore := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testname,
			Annotations: map[string]string{
				statefulset.AnnotationTemplateHash: "same",
			},
		},
	}
	fakeClient := fake.NewClient(&dsBefore)

	dsUpdate := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testname,
			Annotations: map[string]string{
				statefulset.AnnotationTemplateHash: "same",
			},
		},
	}
	result, err := CreateOrUpdateDaemonSet(fakeClient, log, &dsUpdate)
	require.NoError(t, err)
	assert.False(t, result)
}
