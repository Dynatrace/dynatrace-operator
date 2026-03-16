package k8ssecret

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetDataFromSecretName(t *testing.T) {
	const testSecretName = "test-secret"
	const testNamespace = "test-namespace"
	const testSecretDataKey = "key"

	getTestSecret := func() *corev1.Secret {
		return &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testSecretName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				testSecretDataKey: dataValue,
			},
		}
	}

	fakeClient := fake.NewClient()
	fakeClient.Create(t.Context(), getTestSecret())

	t.Run("get secret data", func(t *testing.T) {
		data, _ := GetDataFromSecretName(t.Context(), fakeClient, types.NamespacedName{Name: testSecretName, Namespace: testNamespace}, testSecretDataKey, logd.Logger{})
		assert.Equal(t, string(dataValue), data)
	})
}
