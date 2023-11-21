package deployment

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/resources"
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

		deployment := NewRegular(instance)

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

		envVars := prepareContainerEnvVars(instance, instance.Spec.OAuth.Resource)

		assert.Equal(t, envVars, []corev1.EnvVar{
			{Name: consts.EnvEdgeConnectApiEndpointHost, Value: "abc12345.dynatrace.com"},
			{Name: consts.EnvEdgeConnectName, Value: testName},
			{Name: consts.EnvEdgeConnectOauthEndpoint, Value: "https://sso-dev.dynatracelabs.com/sso/oauth2/token"},
			{Name: consts.EnvEdgeConnectOauthResource, Value: "urn:dtenvironment:test12345"},
		})
	})
	t.Run("Create env vars for simple edgeconnect deployment with Envs in spec", func(t *testing.T) {
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
				Env: []corev1.EnvVar{{Name: "DEBUG", Value: "true"}},
			},
			Status: edgeconnectv1alpha1.EdgeConnectStatus{
				UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
		}

		envVars := prepareContainerEnvVars(instance, instance.Spec.OAuth.Resource)

		assert.Equal(t, envVars, []corev1.EnvVar{
			{Name: "DEBUG", Value: "true"},
			{Name: consts.EnvEdgeConnectApiEndpointHost, Value: "abc12345.dynatrace.com"},
			{Name: consts.EnvEdgeConnectName, Value: testName},
			{Name: consts.EnvEdgeConnectOauthEndpoint, Value: "https://sso-dev.dynatracelabs.com/sso/oauth2/token"},
			{Name: consts.EnvEdgeConnectOauthResource, Value: "urn:dtenvironment:test12345"},
		})
	})
	t.Run("Create all env vars for simple edgeconnect deployment", func(t *testing.T) {
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
				HostRestrictions: "*.test.com",
			},
			Status: edgeconnectv1alpha1.EdgeConnectStatus{
				UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
		}

		envVars := prepareContainerEnvVars(instance, instance.Spec.OAuth.Resource)

		assert.Equal(t, envVars, []corev1.EnvVar{
			{Name: consts.EnvEdgeConnectApiEndpointHost, Value: "abc12345.dynatrace.com"},
			{Name: consts.EnvEdgeConnectName, Value: testName},
			{Name: consts.EnvEdgeConnectOauthEndpoint, Value: "https://sso-dev.dynatracelabs.com/sso/oauth2/token"},
			{Name: consts.EnvEdgeConnectOauthResource, Value: "urn:dtenvironment:test12345"},
			{Name: consts.EnvEdgeConnectRestrictHostsTo, Value: "*.test.com"},
		})
	})
}

func Test_buildAppLabels(t *testing.T) {
	testEdgeConnect := &edgeconnectv1alpha1.EdgeConnect{
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
			Version: status.VersionStatus{
				Version: "",
			},
			UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
		},
	}

	t.Run("Check version label set correctly", func(t *testing.T) {
		labels := buildAppLabels(testEdgeConnect)
		assert.Equal(t, "", labels.Version)
	})
}

func Test_prepareResourceRequirements(t *testing.T) {
	testEdgeConnect := &edgeconnectv1alpha1.EdgeConnect{
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
			Version: status.VersionStatus{
				Version: "",
			},
			UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
		},
	}

	t.Run("Check limits requirements are set correctly", func(t *testing.T) {
		customResources := corev1.ResourceRequirements{
			Limits: resources.NewResourceList("500m", "256Mi"),
		}
		testEdgeConnect.Spec.Resources = customResources
		resourceRequirements := prepareResourceRequirements(testEdgeConnect)
		assert.Equal(t, customResources.Limits, resourceRequirements.Limits)
		// check that we use default requests when not provided
		assert.Equal(t, resources.NewResourceList("100m", "128Mi"), resourceRequirements.Requests)
	})

	t.Run("Check requests in requirements are set correctly", func(t *testing.T) {
		customResources := corev1.ResourceRequirements{
			Requests: resources.NewResourceList("500m", "256Mi"),
		}
		testEdgeConnect.Spec.Resources = customResources
		resourceRequirements := prepareResourceRequirements(testEdgeConnect)
		assert.Equal(t, customResources.Requests, resourceRequirements.Requests)
		// check that we use default requests when not provided
		assert.Equal(t, resources.NewResourceList("100m", "128Mi"), resourceRequirements.Limits)
	})
}
