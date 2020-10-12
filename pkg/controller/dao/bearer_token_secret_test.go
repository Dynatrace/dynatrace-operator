package dao

import (
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestFindBearerTokenSecret(t *testing.T) {
	tokenValue := []byte("super-secret-bearer-token")
	kubeClient := fake.NewFakeClientWithScheme(scheme.Scheme,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      _const.BearerTokenSecretName,
				Namespace: _const.DynatraceNamespace,
			},
			Data: map[string][]byte{
				"token": tokenValue,
			},
		})

	secret, err := FindBearerTokenSecret(kubeClient, _const.BearerTokenSecretName)
	assert.NoError(t, err)
	assert.NotNil(t, secret)
	assert.Equal(t, 1, len(secret.Data))

	token, hasToken := secret.Data["token"]
	assert.True(t, hasToken)
	assert.Equal(t, tokenValue, token)
}
