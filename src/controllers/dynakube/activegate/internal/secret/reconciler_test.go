package secret

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace         = "test-namespace"
	testSecretName        = "test-secret"
	testSecretKey         = "test-key"
	testSecretValueFirst  = "test-secret-value"
	testSecretValueSecond = "test-secret-value-second"
)

func newTestReconcilerWithInstance(client client.Client, value string) *Reconciler {
	secret := createTestSecret(value)

	r := NewReconciler(client, client, &secret)
	return r
}

func createTestSecret(value string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			testSecretKey: []byte(value),
		},
	}
}

func TestReconcile(t *testing.T) {
	t.Run(`reconcile secret first time`, func(t *testing.T) {
		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build(), testSecretValueFirst)
		_, err := r.Reconcile()

		require.NoError(t, err)

		var secret corev1.Secret
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: testSecretName, Namespace: testNamespace}, &secret)

		require.NoError(t, err)
		assert.Equal(t, []byte(testSecretValueFirst), secret.Data[testSecretKey])
	})
	t.Run(`reconcile secret changed`, func(t *testing.T) {
		secret := createTestSecret(testSecretValueFirst)
		clt := fake.NewClientBuilder().WithObjects(&secret).Build()

		r := newTestReconcilerWithInstance(clt, testSecretValueSecond)
		_, err := r.Reconcile()

		require.NoError(t, err)

		var secretUpdated corev1.Secret
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: testSecretName, Namespace: testNamespace}, &secretUpdated)

		require.NoError(t, err)
		assert.Equal(t, []byte(testSecretValueSecond), secretUpdated.Data[testSecretKey])
	})
}
