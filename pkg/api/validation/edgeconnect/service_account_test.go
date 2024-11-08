package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceAccountName(t *testing.T) {
	t.Run("intentionally empty name", func(t *testing.T) {
		empty := ""
		ec := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ServiceAccountName: &empty,
			},
		}
		assertDenied(t, []string{errorInvalidServiceName}, ec)
	})

	t.Run("not set", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "tenant.apps.dynatrace.com",
			},
		}
		assertAllowed(t, ec)
	})
}

func prepareTestServiceAccount(name string, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
