package kubesystem

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetUID(t *testing.T) {
	const testUID = types.UID("test-uid")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: Namespace,
					UID:  testUID,
				},
			},
		).Build()
	uid, err := GetUID(context.Background(), fakeClient)

	require.NoError(t, err)
	assert.NotEmpty(t, uid)
	assert.Equal(t, testUID, uid)
}

func TestIsDeployedViaOLM(t *testing.T) {
	testPodName := "test-pod"
	testNamespaceName := "test-namespace"

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespaceName,
			Annotations: map[string]string{
				"olm.operatorNamespace": "operators",
			},
		},
	}

	deployed := IsDeployedViaOlm(pod)
	assert.True(t, deployed)
}
