package dtpullsecret

import (
	"context"
	"fmt"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
	testKey       = "test-key"
	testValue     = "test-value"
)

func TestGetImagePullSecret(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme)
	instance := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		}}
	secret, err := GetImagePullSecret(fakeClient, instance)

	assert.Nil(t, secret)
	assert.Error(t, err)
	assert.IsType(t, &k8serrors.StatusError{}, err)

	pullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      fmt.Sprintf("%s-pull-secret", testName),
		},
		Data: map[string][]byte{testKey: []byte(testValue)}}
	err = fakeClient.Create(context.TODO(), pullSecret)

	assert.NoError(t, err)

	secret, err = GetImagePullSecret(fakeClient, instance)

	assert.NotNil(t, secret)
	assert.NoError(t, err)
	assert.Equal(t, pullSecret.Name, secret.Name)
	assert.Equal(t, pullSecret.Namespace, secret.Namespace)
	assert.Contains(t, pullSecret.Data, testKey)
	assert.Equal(t, pullSecret.Data[testKey], []byte(testValue))
}
