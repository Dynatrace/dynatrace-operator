package kubemon

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestAddToDashboard(t *testing.T) {
	t.Run(`AddToDashboard returns id on successful call`, func(t *testing.T) {
		instance := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
			},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: dynatracev1alpha1.KubernetesMonitoringSpec{
					KubernetesAPIEndpoint: testEndpoint,
				}}}
		fakeClient := fake.NewFakeClientWithScheme(
			scheme.Scheme,
			instance,
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      MonitoringServiceAccount,
					Namespace: testNamespace,
				},
				Secrets: []corev1.ObjectReference{{Name: testName}},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"token": []byte(testValue),
				},
			})
		mockClient := &dtclient.MockDynatraceClient{}
		logger := logf.Log

		mockClient.
			On("AddToDashboard", mock.AnythingOfType("string"), testEndpoint, testValue).
			Return(testId, nil)

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, mockClient, logger, nil, instance)
		result, err := r.addToDashboard()

		assert.NoError(t, err)
		assert.Equal(t, testId, result)
	})
	t.Run(`AddToDashboard can use customized service account name`, func(t *testing.T) {
		instance := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: dynatracev1alpha1.KubernetesMonitoringSpec{
					KubernetesAPIEndpoint: testEndpoint,
				}}}
		fakeClient := fake.NewFakeClientWithScheme(
			scheme.Scheme,
			instance,
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Secrets: []corev1.ObjectReference{{Name: testName}},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"token": []byte(testValue),
				},
			})
		mockClient := &dtclient.MockDynatraceClient{}
		logger := logf.Log

		mockClient.
			On("AddToDashboard", mock.AnythingOfType("string"), testEndpoint, testValue).
			Return(testId, nil)

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, mockClient, logger, nil, instance)
		result, err := r.addToDashboard()

		assert.Error(t, err)
		assert.Equal(t, "", result)

		instance.Spec.KubernetesMonitoringSpec.ServiceAccountName = testName
		err = fakeClient.Update(context.TODO(), instance)

		assert.NoError(t, err)

		result, err = r.addToDashboard()

		assert.NoError(t, err)
		assert.Equal(t, testId, result)
	})
}

func TestFindBearerToken_FindsArbitraryTokenName(t *testing.T) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Secrets: []corev1.ObjectReference{{Name: testName}}}
	instance := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		}}
	fakeClient := fake.NewFakeClientWithScheme(
		scheme.Scheme,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				testKey: []byte(testValue),
			},
		})
	r := NewReconciler(fakeClient, nil, nil, nil, nil, nil, instance)
	token, err := r.findBearerTokenSecret(serviceAccount)

	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, testValue, string(token.Data[testKey]))
}
