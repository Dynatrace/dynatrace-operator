package deployment

import (
	"testing"
	"time"

	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName      = "test-name-edgeconnectv1alpha1"
	testNamespace = "test-namespace"
)

func TestNew(t *testing.T) {
	t.Run("Create new edgeconnect deployment", func(t *testing.T) {
		instance := &edgeconnectv1alpha1.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnectv1alpha1.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
			},
			Status: edgeconnectv1alpha1.EdgeConnectStatus{
				UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
		}

		deployment := New(instance)

		assert.NotNil(t, deployment)
	})
}

func Test_prepareContainerEnvVars(t *testing.T) {
	t.Run("Create env vars for simple edgeconnect deployment", func(t *testing.T) {
		instance := &edgeconnectv1alpha1.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnectv1alpha1.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnectv1alpha1.OAuthSpec{
					ClientSecret: "secret-name",
					Endpoint:     "https://sso-dev.dynatracelabs.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
				},
			},
			Status: edgeconnectv1alpha1.EdgeConnectStatus{
				UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
		}

		envVars := prepareContainerEnvVars(instance)

		assert.Equal(t, envVars, []corev1.EnvVar{
			{Name: "EDGE_CONNECT_NAME", Value: testName},
			{Name: "EDGE_CONNECT_API_ENDPOINT_HOST", Value: "abc12345.dynatrace.com"},
			{Name: "EDGE_CONNECT_OAUTH__ENDPOINT", Value: "https://sso-dev.dynatracelabs.com/sso/oauth2/token"},
			{Name: "EDGE_CONNECT_OAUTH__RESOURCE", Value: "urn:dtenvironment:test12345"},
		})
	})
}
