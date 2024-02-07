package dtpullsecret

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
	testKey       = "test-key"
	testValue     = "test-value"
)

func TestGetImagePullSecret(t *testing.T) {
	fakeClient := fake.NewClient()
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		}}
	secret, err := GetImagePullSecret(fakeClient, instance)

	assert.Nil(t, secret)
	require.Error(t, err)
	assert.IsType(t, &k8serrors.StatusError{}, errors.Cause(err))

	pullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      fmt.Sprintf("%s-pull-secret", testName),
		},
		Data: map[string][]byte{testKey: []byte(testValue)}}
	err = fakeClient.Create(context.TODO(), pullSecret)

	require.NoError(t, err)

	secret, err = GetImagePullSecret(fakeClient, instance)

	assert.NotNil(t, secret)
	require.NoError(t, err)
	assert.Equal(t, pullSecret.Name, secret.Name)
	assert.Equal(t, pullSecret.Namespace, secret.Namespace)
	assert.Contains(t, pullSecret.Data, testKey)
	assert.Equal(t, pullSecret.Data[testKey], []byte(testValue))
}
