package pod

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPod(t *testing.T) {
	testPodName := "testPod"
	testNamespace := "testNamespace"
	fakeClient := fake.NewClient(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testPodName,
				Namespace: testNamespace,
			},
		},
	)
	t.Run("get existing pod", func(t *testing.T) {
		pod, err := Get(context.TODO(), fakeClient, testPodName, testNamespace)
		require.NoError(t, err)
		assert.NotNil(t, pod)
	})
	t.Run("return error if pod does not exist", func(t *testing.T) {
		pod, err := Get(context.TODO(), fakeClient, "not a pod name", testNamespace)
		require.Error(t, err)
		assert.Nil(t, pod)
	})
}
