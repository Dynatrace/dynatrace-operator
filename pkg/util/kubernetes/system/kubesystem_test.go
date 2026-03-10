package system

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
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
	t.Run("returns true when OLM_OPERATOR_NAMESPACE is set", func(t *testing.T) {
		t.Setenv(k8senv.OlmOperatorNamespaceEnv, "operators")

		assert.True(t, IsDeployedViaOlm())
	})

	t.Run("returns false when OLM_OPERATOR_NAMESPACE is not set", func(t *testing.T) {
		// t.Setenv is not used here, so the env var is simply absent
		assert.False(t, IsDeployedViaOlm())
	})

	t.Run("returns false when OLM_OPERATOR_NAMESPACE is empty", func(t *testing.T) {
		t.Setenv(k8senv.OlmOperatorNamespaceEnv, "")

		assert.False(t, IsDeployedViaOlm())
	})
}
