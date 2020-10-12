package dao

import (
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis"
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	_ = apis.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
	_ = os.Setenv(k8sutil.WatchNamespaceEnvVar, _const.DynatraceNamespace)
}

func TestFindServiceAccount(t *testing.T) {
	kubeClient := fake.NewFakeClientWithScheme(scheme.Scheme,
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: _const.DynatraceNamespace,
				Name:      _const.ServiceAccountName,
			},
			Secrets: []corev1.ObjectReference{
				{Name: _const.BearerTokenSecretName},
			},
		})

	serviceAccount, err := FindServiceAccount(kubeClient)
	assert.NoError(t, err)
	assert.NotNil(t, serviceAccount)
	assert.Equal(t, 1, len(serviceAccount.Secrets))

	for _, secret := range serviceAccount.Secrets {
		assert.Equal(t, _const.BearerTokenSecretName, secret.Name)
	}
}
