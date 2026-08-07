// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_isInvalidServiceName(t *testing.T) {
	t.Run("intentionally empty name", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ServiceAccountName: new(""),
			},
		}
		assertDenied(t, []string{errorInvalidServiceName}, ec)
	})

	t.Run("not set", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "tenant.apps.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint: testValidOAuthEndpoint,
				},
			},
		}
		assertAllowed(t, ec)
	})
}

func prepareTestServiceAccount(t *testing.T, name string, namespace string) *corev1.ServiceAccount {
	t.Helper()

	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
