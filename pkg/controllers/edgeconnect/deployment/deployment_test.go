package deployment

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sresource"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/version"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
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
	t.Cleanup(version.DisableCacheForTest(123))

	const (
		testObjectMetaLabelKey        = "test-om-label-key"
		testObjectMetaLabelValue      = "test-om-label-value"
		testObjectMetaAnnotationKey   = "test-om-annotation-key"
		testObjectMetaAnnotationValue = "test-om-annotation-value"
		testLabelKey                  = "test-label-key"
		testLabelValue                = "test-label-value"
		testAnnotationKey             = "test-annotation-key"
		testAnnotationValue           = "test-annotation-value"
	)

	testECWithLabels := func(specLabels map[string]string) *edgeconnect.EdgeConnect {
		return &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Labels:    map[string]string{testObjectMetaLabelKey: testObjectMetaLabelValue},
			},
			Spec: edgeconnect.EdgeConnectSpec{Labels: specLabels},
		}
	}

	testECWithAnnotations := func(specAnnotations map[string]string) *edgeconnect.EdgeConnect {
		return &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:        testName,
				Namespace:   testNamespace,
				Annotations: map[string]string{testObjectMetaAnnotationKey: testObjectMetaAnnotationValue},
			},
			Spec: edgeconnect.EdgeConnectSpec{Annotations: specAnnotations},
		}
	}

	testECWithAppArmor := func() *edgeconnect.EdgeConnect {
		return &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec: edgeconnect.EdgeConnectSpec{
				Annotations: map[string]string{
					corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + consts.EdgeConnectContainerName: corev1.DeprecatedAppArmorBetaProfileRuntimeDefault,
				},
			},
		}
	}

	t.Run("create new edgeconnect deployment", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "abc12345.dynatrace.com",
			},
		}

		deployment := New(ec)

		assert.NotNil(t, deployment)
	})

	t.Run("check empty custom labels", func(t *testing.T) {
		ec := testECWithLabels(nil)

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

	t.Run("check custom label set correctly", func(t *testing.T) {
		ec := testECWithLabels(map[string]string{testLabelKey: testLabelValue})

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

	t.Run("check empty annotations", func(t *testing.T) {
		ec := testECWithAnnotations(nil)

		deployment := New(ec)

		assert.Len(t, deployment.Spec.Template.Annotations, 1)
		assert.Contains(t, deployment.Spec.Template.Annotations, webhook.AnnotationDynatraceInject)
		assert.Equal(t, "false", deployment.Spec.Template.Annotations[webhook.AnnotationDynatraceInject])

		assert.NotContains(t, deployment.Annotations, testObjectMetaAnnotationKey)
	})

	t.Run("check custom annotations set correctly", func(t *testing.T) {
		ec := testECWithAnnotations(map[string]string{testAnnotationKey: testAnnotationValue})

		deployment := New(ec)

		assert.Len(t, deployment.Spec.Template.Annotations, 2)
		assert.Contains(t, deployment.Spec.Template.Annotations, testAnnotationKey)
		assert.Equal(t, testAnnotationValue, deployment.Spec.Template.Annotations[testAnnotationKey])
		assert.Contains(t, deployment.Spec.Template.Annotations, webhook.AnnotationDynatraceInject)
		assert.Equal(t, "false", deployment.Spec.Template.Annotations[webhook.AnnotationDynatraceInject])

		assert.NotContains(t, deployment.Annotations, testAnnotationKey)
		assert.NotContains(t, deployment.Annotations, testObjectMetaAnnotationKey)
	})

	t.Run("apparmor is untouched in 1.30", func(t *testing.T) {
		version.DisableCacheForTest(30)

		deployment := New(testECWithAppArmor())

		assert.Contains(t, deployment.Spec.Template.Annotations, corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix+consts.EdgeConnectContainerName)
		require.Len(t, deployment.Spec.Template.Spec.Containers, 1)
		assert.Nil(t, deployment.Spec.Template.Spec.Containers[0].SecurityContext.AppArmorProfile)
	})

	t.Run("apparmor is migrated in 1.31", func(t *testing.T) {
		version.DisableCacheForTest(31)

		deployment := New(testECWithAppArmor())

		assert.NotContains(t, deployment.Spec.Template.Annotations, corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix+consts.EdgeConnectContainerName)
		require.Len(t, deployment.Spec.Template.Spec.Containers, 1)
		assert.NotNil(t, deployment.Spec.Template.Spec.Containers[0].SecurityContext.AppArmorProfile)
	})
}

func Test_buildAppLabels(t *testing.T) {
	t.Run("check version label set correctly", func(t *testing.T) {
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
		}

		labels := buildAppLabels(ec)

		assert.Empty(t, labels.Version)
	})
}

func Test_prepareResourceRequirements(t *testing.T) {
	testEC := func(resources corev1.ResourceRequirements) *edgeconnect.EdgeConnect {
		return &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret-name",
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
				},
				Resources: resources,
			},
		}
	}

	t.Run("check limits requirements are set correctly", func(t *testing.T) {
		ec := testEC(corev1.ResourceRequirements{
			Limits: k8sresource.NewResourceList("500m", "256Mi"),
		})

		resourceRequirements := prepareResourceRequirements(ec)

		assert.Equal(t, ec.Spec.Resources.Limits, resourceRequirements.Limits)
		// check that we use default requests when not provided
		assert.Equal(t, k8sresource.NewResourceList("100m", "128Mi"), resourceRequirements.Requests)
	})

	t.Run("check requests in requirements are set correctly", func(t *testing.T) {
		ec := testEC(corev1.ResourceRequirements{
			Requests: k8sresource.NewResourceList("500m", "256Mi"),
		})

		resourceRequirements := prepareResourceRequirements(ec)

		assert.Equal(t, ec.Spec.Resources.Requests, resourceRequirements.Requests)
		// check that we use default limits when not provided
		assert.Equal(t, k8sresource.NewResourceList("100m", "128Mi"), resourceRequirements.Limits)
	})
}
