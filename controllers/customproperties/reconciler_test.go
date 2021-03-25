package customproperties

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
	testValue     = "test-value"
	testKey       = "test-key"
	testOwner     = "test"
)

func TestReconciler_Reconcile(t *testing.T) {
	t.Run(`Reconile works with minimal setup`, func(t *testing.T) {
		r := NewReconciler(nil, nil, nil, "", dynatracev1alpha1.DynaKubeValueSource{}, nil)
		err := r.Reconcile()
		assert.NoError(t, err)
	})
	t.Run(`Reconile creates custom properties secret`, func(t *testing.T) {
		valueSource := dynatracev1alpha1.DynaKubeValueSource{Value: testValue}
		instance := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		fakeClient := fake.NewClient(instance)
		r := NewReconciler(fakeClient, instance, nil, testOwner, valueSource, scheme.Scheme)
		err := r.Reconcile()

		assert.NoError(t, err)

		var customPropertiesSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: r.buildCustomPropertiesName(testName), Namespace: testNamespace}, &customPropertiesSecret)

		assert.NoError(t, err)
		assert.NotNil(t, customPropertiesSecret)
		assert.NotEmpty(t, customPropertiesSecret.Data)
		assert.Contains(t, customPropertiesSecret.Data, DataKey)
		assert.Equal(t, customPropertiesSecret.Data[DataKey], []byte(testValue))
	})
	t.Run(`Reconcile updates custom properties only if data changed`, func(t *testing.T) {
		valueSource := dynatracev1alpha1.DynaKubeValueSource{Value: testValue}
		instance := &dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		fakeClient := fake.NewClient(instance)
		r := NewReconciler(fakeClient, instance, nil, testOwner, valueSource, scheme.Scheme)
		err := r.Reconcile()

		assert.NoError(t, err)

		var customPropertiesSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: r.buildCustomPropertiesName(testName), Namespace: testNamespace}, &customPropertiesSecret)

		assert.NoError(t, err)
		assert.NotNil(t, customPropertiesSecret)
		assert.NotEmpty(t, customPropertiesSecret.Data)
		assert.Contains(t, customPropertiesSecret.Data, DataKey)
		assert.Equal(t, customPropertiesSecret.Data[DataKey], []byte(testValue))

		err = r.Reconcile()

		assert.NoError(t, err)

		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: r.buildCustomPropertiesName(testName), Namespace: testNamespace}, &customPropertiesSecret)

		assert.NoError(t, err)
		assert.NotNil(t, customPropertiesSecret)
		assert.NotEmpty(t, customPropertiesSecret.Data)
		assert.Contains(t, customPropertiesSecret.Data, DataKey)
		assert.Equal(t, customPropertiesSecret.Data[DataKey], []byte(testValue))

		r.customPropertiesSource.Value = testKey
		err = r.Reconcile()

		assert.NoError(t, err)

		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: r.buildCustomPropertiesName(testName), Namespace: testNamespace}, &customPropertiesSecret)

		assert.NoError(t, err)
		assert.NotNil(t, customPropertiesSecret)
		assert.NotEmpty(t, customPropertiesSecret.Data)
		assert.Contains(t, customPropertiesSecret.Data, DataKey)
		assert.Equal(t, customPropertiesSecret.Data[DataKey], []byte(testKey))
	})
}
