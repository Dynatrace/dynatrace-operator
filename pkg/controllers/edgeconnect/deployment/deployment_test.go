package deployment

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sresource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				APIServer: "abc12345.dynatrace.com",
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
			APIServer: "abc12345.dynatrace.com",
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
		assert.Empty(t, labels.Version)
	})
}

func TestLabels(t *testing.T) {
	testObjectMetaLabelKey := "test-om-label-key"
	testObjectMetaValue := "test-om-label-value"

	testLabelKey := "test-label-key"
	testLabelValue := "test-label-value"

	t.Run("Check empty custom labels", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels: map[string]string{
					testObjectMetaLabelKey: testObjectMetaValue,
				},
			},
			Spec: edgeconnect.EdgeConnectSpec{},
		}

		deployment := New(ec)

		require.Len(t, deployment.Spec.Template.Labels, 5)
		assert.Contains(t, deployment.Spec.Template.Labels, k8slabel.AppNameLabel)
		assert.Contains(t, deployment.Spec.Template.Labels, k8slabel.AppCreatedByLabel)
		assert.Contains(t, deployment.Spec.Template.Labels, k8slabel.AppManagedByLabel)
		assert.Contains(t, deployment.Spec.Template.Labels, k8slabel.AppVersionLabel)
		assert.Contains(t, deployment.Spec.Template.Labels, k8slabel.AppComponentLabel)

		require.Len(t, deployment.Labels, 5)
		assert.Contains(t, deployment.Labels, k8slabel.AppNameLabel)
		assert.Contains(t, deployment.Labels, k8slabel.AppCreatedByLabel)
		assert.Contains(t, deployment.Labels, k8slabel.AppManagedByLabel)
		assert.Contains(t, deployment.Labels, k8slabel.AppVersionLabel)
		assert.Contains(t, deployment.Labels, k8slabel.AppComponentLabel)
	})

	t.Run("Check custom label set correctly", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels: map[string]string{
					testObjectMetaLabelKey: testObjectMetaValue,
				},
			},
			Spec: edgeconnect.EdgeConnectSpec{
				Labels: map[string]string{
					testLabelKey: testLabelValue,
				},
			},
		}

		deployment := New(ec)

		assert.Len(t, deployment.Spec.Template.Labels, 6)
		assert.Contains(t, deployment.Spec.Template.Labels, k8slabel.AppNameLabel)
		assert.Contains(t, deployment.Spec.Template.Labels, k8slabel.AppCreatedByLabel)
		assert.Contains(t, deployment.Spec.Template.Labels, k8slabel.AppManagedByLabel)
		assert.Contains(t, deployment.Spec.Template.Labels, k8slabel.AppVersionLabel)
		assert.Contains(t, deployment.Spec.Template.Labels, k8slabel.AppComponentLabel)

		assert.Contains(t, deployment.Spec.Template.Labels, testLabelKey)
		assert.Equal(t, testLabelValue, deployment.Spec.Template.Labels[testLabelKey])

		require.Len(t, deployment.Labels, 5)
		assert.Contains(t, deployment.Labels, k8slabel.AppNameLabel)
		assert.Contains(t, deployment.Labels, k8slabel.AppCreatedByLabel)
		assert.Contains(t, deployment.Labels, k8slabel.AppManagedByLabel)
		assert.Contains(t, deployment.Labels, k8slabel.AppVersionLabel)
		assert.Contains(t, deployment.Labels, k8slabel.AppComponentLabel)
	})
}

func TestAnnotations(t *testing.T) {
	testObjectMetaAnnotationKey := "test-om-annotation-key"
	testObjectMetaAnnotationValue := "test-om-annotation-value"

	testAnnotationKey := "test-annotation-key"
	testAnnotationValue := "test-annotation-value"

	t.Run("Check empty annotations", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					testObjectMetaAnnotationKey: testObjectMetaAnnotationValue,
				},
			},
			Spec: edgeconnect.EdgeConnectSpec{},
		}

		deployment := New(ec)

		assert.Nil(t, deployment.Spec.Template.Annotations)

		assert.NotContains(t, deployment.Annotations, testObjectMetaAnnotationKey)
	})

	t.Run("Check custom annotations set correctly", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					testObjectMetaAnnotationKey: testObjectMetaAnnotationValue,
				},
			},
			Spec: edgeconnect.EdgeConnectSpec{
				Annotations: map[string]string{
					testAnnotationKey: testAnnotationValue,
				},
			},
		}

		deployment := New(ec)
		deployment.Spec = CreateSpec(ec)

		assert.Len(t, deployment.Spec.Template.Annotations, 1)
		assert.Contains(t, deployment.Spec.Template.Annotations, testAnnotationKey)
		assert.Equal(t, testAnnotationValue, deployment.Spec.Template.Annotations[testAnnotationKey])

		assert.NotContains(t, deployment.Annotations, testAnnotationKey)
		assert.NotContains(t, deployment.Annotations, testObjectMetaAnnotationKey)
	})
}

func Test_prepareResourceRequirements(t *testing.T) {
	ec := &edgeconnect.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: edgeconnect.EdgeConnectSpec{
			APIServer: "abc12345.dynatrace.com",
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
			Limits: k8sresource.NewResourceList("500m", "256Mi"),
		}
		ec.Spec.Resources = customResources
		resourceRequirements := prepareResourceRequirements(ec)
		assert.Equal(t, customResources.Limits, resourceRequirements.Limits)
		// check that we use default requests when not provided
		assert.Equal(t, k8sresource.NewResourceList("100m", "128Mi"), resourceRequirements.Requests)
	})

	t.Run("Check requests in requirements are set correctly", func(t *testing.T) {
		customResources := corev1.ResourceRequirements{
			Requests: k8sresource.NewResourceList("500m", "256Mi"),
		}
		ec.Spec.Resources = customResources
		resourceRequirements := prepareResourceRequirements(ec)
		assert.Equal(t, customResources.Requests, resourceRequirements.Requests)
		// check that we use default requests when not provided
		assert.Equal(t, k8sresource.NewResourceList("100m", "128Mi"), resourceRequirements.Limits)
	})
}
