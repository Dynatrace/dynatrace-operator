package dtpods

import (
	"testing"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
	testKey       = "test-key"
	testValue     = "test-value"
	testVersion   = "1.0.0"
	testImage     = "test-image"
	testImageId   = "test-image-id"
)

func init() {
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))

	utilruntime.Must(dynatracev1alpha1.AddToScheme(scheme.Scheme))
	// +kubebuilder:scaffold:scheme
}

func TestReconciler_Reconcile(t *testing.T) {
	t.Run(`Reconcile works with minimal setup`, func(t *testing.T) {
		instance := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, instance)
		r := NewReconciler(fakeClient, logf.Log, instance, nil, "")
		result, err := r.Reconcile()

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Requeue)
		assert.True(t, instance.Status.UpdatedTimestamp.Time.Before(metav1.Now().Add(1*time.Second)))
		assert.True(t, instance.Status.UpdatedTimestamp.Time.After(metav1.Now().Add(-1*time.Second)))
	})
	t.Run(`Reconcile deletes outdated pods`, func(t *testing.T) {
		releaseValidator := &dtversion.MockReleaseValidator{}
		matchLabels := map[string]string{testKey: testValue}
		podLabels := map[string]string{testKey: testValue, dtversion.VersionKey: testVersion}
		instance := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, instance,
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName + "1",
					Namespace: testNamespace,
					Labels:    podLabels,
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{{Image: testImage, ImageID: testImageId}},
				}},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName + "2",
					Namespace: testNamespace,
					Labels:    podLabels,
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{{Image: testImage, ImageID: testImageId}},
				}})
		r := NewReconciler(fakeClient, logf.Log, instance, matchLabels, "")

		r.releaseValidatorConstructor = func(_ string, _ map[string]string, _ *dtversion.DockerConfig) dtversion.ReleaseValidator {
			return releaseValidator
		}

		releaseValidator.
			On("IsLatest").
			Return(false, nil)

		result, err := r.Reconcile()

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Requeue)
		assert.True(t, instance.Status.UpdatedTimestamp.Time.Before(metav1.Now().Add(1*time.Second)))
		assert.True(t, instance.Status.UpdatedTimestamp.Time.After(metav1.Now().Add(-1*time.Second)))

		pods, err := dtversion.NewPodFinder(fakeClient, instance, matchLabels).FindPods()
		assert.NoError(t, err)
		assert.Empty(t, pods)
	})
	t.Run(`Reconcile does not delete up to date pods`, func(t *testing.T) {
		releaseValidator := &dtversion.MockReleaseValidator{}
		matchLabels := map[string]string{testKey: testValue}
		podLabels := map[string]string{testKey: testValue, dtversion.VersionKey: testVersion}
		instance := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, instance,
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName + "1",
					Namespace: testNamespace,
					Labels:    podLabels,
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{{Image: testImage, ImageID: testImageId}},
				}},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName + "2",
					Namespace: testNamespace,
					Labels:    podLabels,
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{{Image: testImage, ImageID: testImageId}},
				}})
		r := NewReconciler(fakeClient, logf.Log, instance, matchLabels, "")

		r.releaseValidatorConstructor = func(_ string, _ map[string]string, _ *dtversion.DockerConfig) dtversion.ReleaseValidator {
			return releaseValidator
		}

		releaseValidator.
			On("IsLatest").
			Return(true, nil)

		result, err := r.Reconcile()

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Requeue)
		assert.True(t, instance.Status.UpdatedTimestamp.Time.Before(metav1.Now().Add(1*time.Second)))
		assert.True(t, instance.Status.UpdatedTimestamp.Time.After(metav1.Now().Add(-1*time.Second)))

		pods, err := dtversion.NewPodFinder(fakeClient, instance, matchLabels).FindPods()
		assert.NoError(t, err)
		assert.NotEmpty(t, pods)
		assert.Equal(t, 2, len(pods))
	})
}
