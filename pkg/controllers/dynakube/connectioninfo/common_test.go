package connectioninfo

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestIsTenantSecretPresent(t *testing.T) {
	ctx := t.Context()
	log := logd.Get().WithName("test-connectioninfo")
	secretNamespacedName := types.NamespacedName{Name: "secret-not-found", Namespace: "test-namespace"}

	t.Run("secret found", func(t *testing.T) {
		testNamespace := "test-namespace"
		testName := "test-name"

		fakeClient := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
		})

		existingSecretNamespacedName := types.NamespacedName{Name: testName, Namespace: testNamespace}

		secrets := k8ssecret.Query(fakeClient, fakeClient, log)

		isPresent, err := IsTenantSecretPresent(ctx, secrets, existingSecretNamespacedName, log)
		assert.True(t, isPresent)
		assert.NoError(t, err)
	})

	t.Run("secret not found", func(t *testing.T) {
		fakeClient := fake.NewClient()

		secrets := k8ssecret.Query(fakeClient, fakeClient, log)

		isPresent, err := IsTenantSecretPresent(ctx, secrets, secretNamespacedName, log)
		assert.False(t, isPresent)
		assert.NoError(t, err)
	})

	t.Run("k8s api error", func(t *testing.T) {
		testError := errors.New("test-error")

		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return testError
			},
		})

		secrets := k8ssecret.Query(fakeClient, fakeClient, log)

		isPresent, err := IsTenantSecretPresent(ctx, secrets, secretNamespacedName, log)
		assert.False(t, isPresent)
		assert.ErrorIs(t, err, testError)
	})
}
