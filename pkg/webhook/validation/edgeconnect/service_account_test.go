package edgeconnect

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceAccountName(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		edgeConnect := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ServiceAccountName: "",
			},
		}
		assertDeniedResponse(t, []string{errorInvalidServiceName}, edgeConnect)
	})
	t.Run("exists", func(t *testing.T) {
		edgeConnect := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer:          "tenantid-test.dev.apps.dynatracelabs.com",
				ServiceAccountName: testServiceAccountName,
			},
		}
		assertAllowedResponse(t, edgeConnect, prepareTestServiceAccount(testServiceAccountName, testNamespace))
	})
	t.Run("not exists", func(t *testing.T) {
		edgeConnect := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer:          "tenantid-test.dev.apps.dynatracelabs.com",
				ServiceAccountName: testServiceAccountName,
			},
		}
		assertDeniedResponse(t, []string{errorServiceAccountNotExist}, edgeConnect)
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
