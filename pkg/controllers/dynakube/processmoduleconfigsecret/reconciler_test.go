package processmoduleconfigsecret

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	clientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testName                   = "test-name"
	testNamespace              = "test-namespace"
	testTokenValue             = "test-token"
	oneAgentTenantSecretSuffix = "oneagent-tenant-secret"
)

func TestReconcile(t *testing.T) {
	t.Run("Create and update works with minimal setup", func(t *testing.T) {
		dynakube := createDynakube(dynatracev1beta2.OneAgentSpec{
			CloudNativeFullStack: &dynatracev1beta2.CloudNativeFullStackSpec{}})

		mockK8sClient := fake.NewClient(dynakube)
		_ = mockK8sClient.Create(context.Background(),
			&corev1.Secret{
				Data: map[string][]byte{connectioninfo.TenantTokenKey: []byte(testTokenValue)},
				ObjectMeta: metav1.ObjectMeta{
					Name:      strings.Join([]string{testName, oneAgentTenantSecretSuffix}, "-"),
					Namespace: testNamespace,
				},
			},
		)

		mockTime := timeprovider.New().Freeze()

		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, createMockDtClient(t, 0), dynakube, mockTime)
		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		checkSecretForValue(t, mockK8sClient, "\"revision\":0")

		condition := meta.FindStatusCondition(*dynakube.Conditions(), pmcConditionType)
		oldTransitionTime := condition.LastTransitionTime
		require.NotNil(t, condition)
		require.NotEmpty(t, oldTransitionTime)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)

		// update should be blocked by timeout
		reconciler.dtClient = createMockDtClient(t, 1)
		err = reconciler.Reconcile(context.Background())

		require.NoError(t, err)
		checkSecretForValue(t, mockK8sClient, "\"revision\":0")

		condition = meta.FindStatusCondition(*dynakube.Conditions(), pmcConditionType)
		require.NotNil(t, condition)
		require.Equal(t, condition.LastTransitionTime, oldTransitionTime)

		// go forward in time => should update again
		mockTime.Set(time.Now().Add(time.Hour))

		err = reconciler.Reconcile(context.Background())
		require.NoError(t, err)
		checkSecretForValue(t, mockK8sClient, "\"revision\":1")

		condition = meta.FindStatusCondition(*dynakube.Conditions(), pmcConditionType)
		require.NotNil(t, condition)
		require.Greater(t, condition.LastTransitionTime.Time, oldTransitionTime.Time)
		assert.Equal(t, conditions.SecretUpdatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("Only runs when required, and cleans up condition", func(t *testing.T) {
		dynakube := createDynakube(dynatracev1beta2.OneAgentSpec{
			ClassicFullStack: &dynatracev1beta2.HostInjectSpec{}})
		conditions.SetSecretCreated(dynakube.Conditions(), pmcConditionType, "this is a test")

		reconciler := NewReconciler(nil, nil, nil, dynakube, timeprovider.New())
		err := reconciler.Reconcile(context.Background())

		require.NoError(t, err)
		assert.Empty(t, *dynakube.Conditions())
	})

	t.Run("problem with k8s request => visible in conditions", func(t *testing.T) {
		dynakube := createDynakube(dynatracev1beta2.OneAgentSpec{
			CloudNativeFullStack: &dynatracev1beta2.CloudNativeFullStackSpec{}})

		boomClient := createBOOMK8sClient()

		mockTime := timeprovider.New().Freeze()

		reconciler := NewReconciler(boomClient,
			boomClient, createMockDtClient(t, 0), dynakube, mockTime)

		err := reconciler.Reconcile(context.Background())

		require.Error(t, err)
		require.Len(t, *dynakube.Conditions(), 1)
		condition := meta.FindStatusCondition(*dynakube.Conditions(), pmcConditionType)
		assert.Equal(t, conditions.KubeApiErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})

	t.Run("problem with dynatrace request => visible in conditions", func(t *testing.T) {
		dynakube := createDynakube(dynatracev1beta2.OneAgentSpec{
			CloudNativeFullStack: &dynatracev1beta2.CloudNativeFullStackSpec{}})

		mockK8sClient := fake.NewClient(dynakube)
		_ = mockK8sClient.Create(context.Background(),
			&corev1.Secret{
				Data: map[string][]byte{connectioninfo.TenantTokenKey: []byte(testTokenValue)},
				ObjectMeta: metav1.ObjectMeta{
					Name:      strings.Join([]string{testName, oneAgentTenantSecretSuffix}, "-"),
					Namespace: testNamespace,
				},
			},
		)

		mockTime := timeprovider.New().Freeze()

		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, createBOOMDtClient(t), dynakube, mockTime)

		err := reconciler.Reconcile(context.Background())

		require.Error(t, err)
		require.Len(t, *dynakube.Conditions(), 1)
		condition := meta.FindStatusCondition(*dynakube.Conditions(), pmcConditionType)
		assert.Equal(t, conditions.DynatraceApiErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func checkSecretForValue(t *testing.T, k8sClient client.Client, shouldContain string) {
	var secret corev1.Secret
	err := k8sClient.Get(context.Background(), client.ObjectKey{Name: extendWithSuffix(testName), Namespace: testNamespace}, &secret)
	require.NoError(t, err)

	processModuleConfig, ok := secret.Data[SecretKeyProcessModuleConfig]
	require.True(t, ok)
	require.True(t, strings.Contains(string(processModuleConfig), shouldContain))
}

func createDynakube(oneAgentSpec dynatracev1beta2.OneAgentSpec) *dynatracev1beta2.DynaKube {
	return &dynatracev1beta2.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynatracev1beta2.DynaKubeSpec{
			OneAgent: oneAgentSpec,
		},
	}
}

func createMockDtClient(t *testing.T, revision uint) *clientmock.Client {
	mockClient := clientmock.NewClient(t)
	mockClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).Return(&dtclient.ProcessModuleConfig{
		Revision:   revision,
		Properties: nil,
	}, nil)

	return mockClient
}

func createBOOMDtClient(t *testing.T) *clientmock.Client {
	mockClient := clientmock.NewClient(t)
	mockClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).Return(nil, errors.New("BOOM"))

	return mockClient
}

func createBOOMK8sClient() client.Client {
	boomClient := fake.NewClientWithInterceptors(interceptor.Funcs{
		Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			return errors.New("BOOM")
		},
		Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			return errors.New("BOOM")
		},
		Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			return errors.New("BOOM")
		},
		Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			return errors.New("BOOM")
		},
	})

	return boomClient
}

func TestGetSecretData(t *testing.T) {
	t.Run("unmarshal secret data into struct", func(t *testing.T) {
		// use Reconcile to automatically create the secret to test
		dynakube := createDynakube(dynatracev1beta2.OneAgentSpec{
			CloudNativeFullStack: &dynatracev1beta2.CloudNativeFullStackSpec{}})
		mockK8sClient := fake.NewClient(dynakube)
		_ = mockK8sClient.Create(context.Background(),
			&corev1.Secret{
				Data: map[string][]byte{connectioninfo.TenantTokenKey: []byte(testTokenValue)},
				ObjectMeta: metav1.ObjectMeta{
					Name:      strings.Join([]string{testName, oneAgentTenantSecretSuffix}, "-"),
					Namespace: testNamespace,
				},
			},
		)

		mockTime := timeprovider.New().Freeze()
		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, createMockDtClient(t, 0), dynakube, mockTime)
		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		got, err := GetSecretData(context.Background(), mockK8sClient, testName, testNamespace)
		require.NoError(t, err)
		assert.Contains(t, got.Properties, dtclient.ProcessModuleProperty{
			Section: "general",
			Key:     "tenantToken",
			Value:   "test-token",
		})
	})
	t.Run("error when secret not found", func(t *testing.T) {
		got, err := GetSecretData(context.Background(), fake.NewClient(), testName, testNamespace)
		require.Error(t, err)
		assert.Nil(t, got)
	})
	t.Run("error when unmarshaling secret data", func(t *testing.T) {
		fakeClient := fake.NewClient()
		_ = fakeClient.Create(context.Background(),
			&corev1.Secret{
				Data:       map[string][]byte{SecretKeyProcessModuleConfig: []byte("WRONG VALUE!")},
				ObjectMeta: metav1.ObjectMeta{Name: extendWithSuffix(testName), Namespace: testNamespace},
			},
		)

		got, err := GetSecretData(context.Background(), fakeClient, testName, testNamespace)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}
