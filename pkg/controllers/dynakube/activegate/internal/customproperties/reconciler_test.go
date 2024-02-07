package customproperties

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Run(`Create works with minimal setup`, func(t *testing.T) {
		r := NewReconciler(nil, nil, "", nil, &dynatracev1beta1.DynaKubeValueSource{})
		err := r.Reconcile(context.Background())
		require.NoError(t, err)
	})
	t.Run(`Create creates custom properties secret`, func(t *testing.T) {
		valueSource := dynatracev1beta1.DynaKubeValueSource{Value: testValue}
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		fakeClient := fake.NewClient(instance)
		r := NewReconciler(fakeClient, instance, testOwner, scheme.Scheme, &valueSource)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var customPropertiesSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: r.buildCustomPropertiesName(testName), Namespace: testNamespace}, &customPropertiesSecret)

		require.NoError(t, err)
		assert.NotNil(t, customPropertiesSecret)
		assert.NotEmpty(t, customPropertiesSecret.Data)
		assert.Contains(t, customPropertiesSecret.Data, DataKey)
		assert.Equal(t, customPropertiesSecret.Data[DataKey], []byte(testValue))
	})
	t.Run(`Create updates custom properties only if data changed`, func(t *testing.T) {
		valueSource := dynatracev1beta1.DynaKubeValueSource{Value: testValue}
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		fakeClient := fake.NewClient(instance)
		r := NewReconciler(fakeClient, instance, testOwner, scheme.Scheme, &valueSource)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var customPropertiesSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: r.buildCustomPropertiesName(testName), Namespace: testNamespace}, &customPropertiesSecret)

		require.NoError(t, err)
		assert.NotNil(t, customPropertiesSecret)
		assert.NotEmpty(t, customPropertiesSecret.Data)
		assert.Contains(t, customPropertiesSecret.Data, DataKey)
		assert.Equal(t, customPropertiesSecret.Data[DataKey], []byte(testValue))

		err = r.Reconcile(context.Background())

		require.NoError(t, err)

		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: r.buildCustomPropertiesName(testName), Namespace: testNamespace}, &customPropertiesSecret)

		require.NoError(t, err)
		assert.NotNil(t, customPropertiesSecret)
		assert.NotEmpty(t, customPropertiesSecret.Data)
		assert.Contains(t, customPropertiesSecret.Data, DataKey)
		assert.Equal(t, customPropertiesSecret.Data[DataKey], []byte(testValue))

		r.customPropertiesSource.Value = testKey
		err = r.Reconcile(context.Background())

		require.NoError(t, err)

		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: r.buildCustomPropertiesName(testName), Namespace: testNamespace}, &customPropertiesSecret)

		require.NoError(t, err)
		assert.NotNil(t, customPropertiesSecret)
		assert.NotEmpty(t, customPropertiesSecret.Data)
		assert.Contains(t, customPropertiesSecret.Data, DataKey)
		assert.Equal(t, customPropertiesSecret.Data[DataKey], []byte(testKey))
	})
}
