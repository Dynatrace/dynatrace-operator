package dtversion

import (
	"fmt"
	"testing"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	testErrorMessage = "test error message"
	testName         = "test-name"
	testImage        = "test-image"
	testNamespace    = "test-namespace"
	testVersion      = "1.0.0"
)

func init() {
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))

	utilruntime.Must(dynatracev1alpha1.AddToScheme(scheme.Scheme))
	// +kubebuilder:scaffold:scheme
}

func TestRetryOnStatusError(t *testing.T) {
	t.Run(`RetryOnStatusError returns reconcile-result with five seconds on status error`, func(t *testing.T) {
		r := NewReconciler(nil, logf.Log, nil, nil)
		result, err := r.retryOnStatusError(&errors.StatusError{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 5*time.Second, result.RequeueAfter)
	})
	t.Run(`RetryOnStatusError returns reconcile-result with five minutes on error`, func(t *testing.T) {
		r := NewReconciler(nil, logf.Log, nil, nil)
		err := fmt.Errorf(testErrorMessage)
		result, err := r.retryOnStatusError(err)

		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 5*time.Minute, result.RequeueAfter)
		assert.EqualError(t, err, testErrorMessage)
	})
}

func TestVersionLabelReconciler_Reconcile(t *testing.T) {
	labels := map[string]string{
		"dynatrace":  "activegate",
		"activegate": testName,
	}
	instance := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		}}
	fakeClient := fake.NewFakeClientWithScheme(
		scheme.Scheme,
		instance,
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + "1",
				Namespace: testNamespace,
				Labels:    labels,
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{Image: testImage},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + "2",
				Namespace: testNamespace,
				Labels:    labels,
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{Image: testImage},
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + dtpullsecret.PullSecretSuffix,
				Namespace: testNamespace,
			}})
	r := NewReconciler(fakeClient, logf.Log, instance, labels)
	mockImageInformation := &MockImageInformation{}

	r.imageInformationConstructor = func(s string, config *DockerConfig) ImageInformation {
		return mockImageInformation
	}
	r.dockerConfigConstructor = func(secret *corev1.Secret) (*DockerConfig, error) {
		return &DockerConfig{}, nil
	}

	mockImageInformation.
		On("GetVersionLabel").
		Return(testVersion, nil)

	assert.NotNil(t, r)

	result, err := r.Reconcile()

	assert.NoError(t, err)
	assert.NotNil(t, result)

	pods, err := NewPodFinder(r, r.instance, r.matchLabels).FindPods()

	assert.NoError(t, err)
	assert.NotEmpty(t, pods)

	for _, pod := range pods {
		assert.NotEmpty(t, pod.Labels)
		assert.Contains(t, pod.Labels, VersionKey)
		assert.Equal(t, testVersion, pod.Labels[VersionKey])
	}
}

func TestGetVersionLabel_ReturnsDockerConfigError(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName + dtpullsecret.PullSecretSuffix,
				Namespace: testNamespace,
			}})
	r := NewReconciler(fakeClient, logf.Log, &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		}}, nil)
	mockImageInformation := &MockImageInformation{}

	r.imageInformationConstructor = func(s string, config *DockerConfig) ImageInformation {
		return mockImageInformation
	}
	r.dockerConfigConstructor = func(secret *corev1.Secret) (*DockerConfig, error) {
		return &DockerConfig{}, fmt.Errorf(testErrorMessage)
	}

	mockImageInformation.
		On("GetVersionLabel").
		Return(testVersion, fmt.Errorf(testErrorMessage+" image information"))

	label, err := r.getVersionLabelForPod(&corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{Image: testImage},
			},
		},
	})

	assert.Error(t, err)
	assert.EqualError(t, err, testErrorMessage)
	assert.Empty(t, label)
}