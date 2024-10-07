package customproperties

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
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
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}

		r := NewReconciler(nil, dk, "", &value.Source{})
		err := r.Reconcile(context.Background())
		require.NoError(t, err)
	})

	t.Run(`Create creates custom properties secret for no-proxy`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy: &value.Source{
					Value: "test",
				},
			}}
		dk.Annotations = map[string]string{
			dynakube.AnnotationFeatureNoProxy: testValue,
		}

		fakeClient := fake.NewClient(dk)
		r := NewReconciler(fakeClient, dk, testOwner, nil)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var customPropertiesSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: r.buildCustomPropertiesName(testName), Namespace: testNamespace}, &customPropertiesSecret)

		require.NoError(t, err)
		assert.NotNil(t, customPropertiesSecret)
		assert.NotEmpty(t, customPropertiesSecret.Data)
		assert.Contains(t, customPropertiesSecret.Data, DataKey)

		expectedValue := "\n" + clientInternalSection + "\n" + noProxyFieldName + "=" + testValue

		assert.Equal(t, []byte(expectedValue), customPropertiesSecret.Data[DataKey])

		assert.Len(t, dk.Status.Conditions, 1)
		assert.Equal(t, customPropertiesConditionType, dk.Status.Conditions[0].Type)
	})

	t.Run(`Create creates custom properties secret for no-proxy with custom properties`, func(t *testing.T) {
		valueSource := value.Source{Value: testValue}
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy: &value.Source{
					Value: "test",
				},
			}}
		dk.Annotations = map[string]string{
			dynakube.AnnotationFeatureNoProxy: testValue,
		}

		fakeClient := fake.NewClient(dk)
		r := NewReconciler(fakeClient, dk, testOwner, &valueSource)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var customPropertiesSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: r.buildCustomPropertiesName(testName), Namespace: testNamespace}, &customPropertiesSecret)

		require.NoError(t, err)
		assert.NotNil(t, customPropertiesSecret)
		assert.NotEmpty(t, customPropertiesSecret.Data)
		assert.Contains(t, customPropertiesSecret.Data, DataKey)

		expectedValue := testValue + "\n" + clientInternalSection + "\n" + noProxyFieldName + "=" + testValue

		assert.Equal(t, []byte(expectedValue), customPropertiesSecret.Data[DataKey])
	})

	t.Run(`Always copy custom properties to secret`, func(t *testing.T) {
		valueSource := value.Source{ValueFrom: testKey}
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
		}

		fakeClient := fake.NewClient(dk, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testKey,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{
				DataKey: []byte(testValue),
			},
		})
		r := NewReconciler(fakeClient, dk, testOwner, &valueSource)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var customPropertiesSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: r.buildCustomPropertiesName(testName), Namespace: testNamespace}, &customPropertiesSecret)

		require.NoError(t, err)
		assert.NotNil(t, customPropertiesSecret)
		assert.NotEmpty(t, customPropertiesSecret.Data)
		assert.Contains(t, customPropertiesSecret.Data, DataKey)
		assert.Equal(t, []byte(testValue), customPropertiesSecret.Data[DataKey])
	})

	t.Run(`Create creates custom properties secret`, func(t *testing.T) {
		valueSource := value.Source{Value: testValue}
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		fakeClient := fake.NewClient(dk)
		r := NewReconciler(fakeClient, dk, testOwner, &valueSource)
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
		valueSource := value.Source{Value: testValue}
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}
		fakeClient := fake.NewClient(dk)
		r := NewReconciler(fakeClient, dk, testOwner, &valueSource)
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
