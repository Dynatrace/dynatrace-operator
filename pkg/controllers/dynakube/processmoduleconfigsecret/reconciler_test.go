package processmoduleconfigsecret

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	clientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName                   = "test-name"
	testNamespace              = "test-namespace"
	testTokenValue             = "test-token"
	tenantTokenKey             = "tenantToken"
	oneAgentTenantSecretSuffix = "oneagent-tenant-secret"
)

func TestReconciler_Reconcile(t *testing.T) {
	t.Run("Create and update works with minimal setup", func(t *testing.T) {
		dynakube := createDynakube(dynatracev1beta1.OneAgentSpec{
			CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{}})

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
			mockK8sClient, createMockDtClient(t, 0), dynakube, scheme.Scheme, mockTime)
		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		checkSecretForValue(t, mockK8sClient, "\"revision\":0")

		condition := meta.FindStatusCondition(*dynakube.Conditions(), conditionType)
		oldTransitionTime := condition.LastTransitionTime
		require.NotNil(t, condition)
		require.NotEmpty(t, oldTransitionTime)

		// update should be blocked by timeout
		reconciler.dtClient = createMockDtClient(t, 1)
		err = reconciler.Reconcile(context.Background())

		require.NoError(t, err)
		checkSecretForValue(t, mockK8sClient, "\"revision\":0")

		condition = meta.FindStatusCondition(*dynakube.Conditions(), conditionType)
		require.NotNil(t, condition)
		require.Equal(t, condition.LastTransitionTime, oldTransitionTime)

		// go forward in time => should update again
		futureTime := metav1.NewTime(time.Now().Add(time.Hour))
		mockTime.Set(&futureTime)

		err = reconciler.Reconcile(context.Background())
		require.NoError(t, err)
		checkSecretForValue(t, mockK8sClient, "\"revision\":1")

		condition = meta.FindStatusCondition(*dynakube.Conditions(), conditionType)
		require.NotNil(t, condition)
		require.Greater(t, condition.LastTransitionTime.Time, oldTransitionTime.Time)
	})
	t.Run("Only runs when required", func(t *testing.T) {
		dynakube := createDynakube(dynatracev1beta1.OneAgentSpec{
			ClassicFullStack: &dynatracev1beta1.HostInjectSpec{}})

		reconciler := NewReconciler(nil, nil, nil, dynakube, scheme.Scheme, timeprovider.New())
		err := reconciler.Reconcile(context.Background())

		require.NoError(t, err)
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

func createDynakube(oneAgentSpec dynatracev1beta1.OneAgentSpec) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
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

func TestGetSecretData(t *testing.T) {
	t.Run("unmarshal secret data into struct", func(t *testing.T) {
		// use Reconcile to automatically create the secret to test
		dynakube := createDynakube(dynatracev1beta1.OneAgentSpec{
			CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{}})
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
			mockK8sClient, createMockDtClient(t, 0), dynakube, scheme.Scheme, mockTime)
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
