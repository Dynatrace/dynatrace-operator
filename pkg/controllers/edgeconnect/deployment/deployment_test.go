package deployment

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/resources"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName      = "test-name-edgeconnect"
	testNamespace = "test-namespace"
)

func TestNew(t *testing.T) {
	t.Run("Create new edgeconnect deployment", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
			},
			Status: edgeconnect.EdgeConnectStatus{
				UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
		}

		deployment := New(ec)

		assert.NotNil(t, deployment)
	})
}

func Test_buildAppLabels(t *testing.T) {
	ec := &edgeconnect.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: edgeconnect.EdgeConnectSpec{
			ApiServer: "abc12345.dynatrace.com",
			OAuth: edgeconnect.OAuthSpec{
				ClientSecret: "secret-name",
				Endpoint:     "https://test.com/sso/oauth2/token",
				Resource:     "urn:dtenvironment:test12345",
			},
		},
		Status: edgeconnect.EdgeConnectStatus{
			Version: status.VersionStatus{
				Version: "",
			},
			UpdatedTimestamp: metav1.NewTime(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
		},
	}

	t.Run("Check version label set correctly", func(t *testing.T) {
		labels := buildAppLabels(ec)
		assert.Equal(t, "", labels.Version)
	})
}

func Test_prepareResourceRequirements(t *testing.T) {
	ec := &edgeconnect.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: edgeconnect.EdgeConnectSpec{
			ApiServer: "abc12345.dynatrace.com",
			OAuth: edgeconnect.OAuthSpec{
				ClientSecret: "secret-name",
				Endpoint:     "https://test.com/sso/oauth2/token",
				Resource:     "urn:dtenvironment:test12345",
			},
		},
		Status: edgeconnect.EdgeConnectStatus{
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
		ec.Spec.Resources = customResources
		resourceRequirements := prepareResourceRequirements(ec)
		assert.Equal(t, customResources.Limits, resourceRequirements.Limits)
		// check that we use default requests when not provided
		assert.Equal(t, resources.NewResourceList("100m", "128Mi"), resourceRequirements.Requests)
	})

	t.Run("Check requests in requirements are set correctly", func(t *testing.T) {
		customResources := corev1.ResourceRequirements{
			Requests: resources.NewResourceList("500m", "256Mi"),
		}
		ec.Spec.Resources = customResources
		resourceRequirements := prepareResourceRequirements(ec)
		assert.Equal(t, customResources.Requests, resourceRequirements.Requests)
		// check that we use default requests when not provided
		assert.Equal(t, resources.NewResourceList("100m", "128Mi"), resourceRequirements.Limits)
	})
}
