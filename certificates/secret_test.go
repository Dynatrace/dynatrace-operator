package certificates

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
)

func TestGetSecret(t *testing.T) {
	t.Run(`get nil if secret does not exists`, func(t *testing.T) {
		clt := fake.NewClient()
		r := &CertificateReconciler{
			clt: clt,
			ctx: context.TODO(),
		}
		secret, err := r.getSecret()
		assert.NoError(t, err)
		assert.Nil(t, secret)
	})
	t.Run(`get secret`, func(t *testing.T) {
		clt := fake.NewClient(&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + "-certs",
				Namespace: testNamespace,
			},
		})
		r := &CertificateReconciler{
			clt:         clt,
			ctx:         context.TODO(),
			webhookName: testName,
			namespace:   testNamespace,
		}
		secret, err := r.getSecret()
		assert.NoError(t, err)
		assert.NotNil(t, secret)
	})
}

func TestGetCertificateData(t *testing.T) {

}
