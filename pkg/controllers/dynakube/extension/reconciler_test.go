package extension

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/servicename"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
)

func TestReconciler_Reconcile(t *testing.T) {
	t.Run("Extension secret not generated when Prometheus is disabled", func(t *testing.T) {
		dk := createDynakube()

		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		// assert extensions token is not generated
		var secretFound corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: testName + "-extensions-token", Namespace: testNamespace}, &secretFound)
		require.True(t, k8serrors.IsNotFound(err))

		// assert conditions are empty
		require.Empty(t, dk.Conditions())
	})
	t.Run("Extension secret gets deleted when Prometheus is disabled", func(t *testing.T) {
		dk := createDynakube()

		// mock SecretCreated condition
		conditions.SetSecretCreated(dk.Conditions(), consts.ExtensionsSecretConditionType, dk.ExtensionsTokenSecretName())

		// mock secret
		secretToken, _ := dttoken.New(consts.EecTokenSecretValuePrefix)
		secretData := map[string][]byte{
			consts.EecTokenSecretKey: []byte(secretToken.String()),
		}
		secretMock, _ := k8ssecret.Build(dk, testName+"-extensions-token", secretData)

		fakeClient := fake.NewClient()
		fakeClient.Create(context.Background(), secretMock)
		r := NewReconciler(fakeClient, fakeClient, dk)

		// assert extensions token is there before reconciliation
		var secretFound corev1.Secret
		err := fakeClient.Get(context.Background(), client.ObjectKey{Name: testName + "-extensions-token", Namespace: testNamespace}, &secretFound)
		require.False(t, k8serrors.IsNotFound(err))

		// assert conditions are not empty
		require.NotEmpty(t, dk.Conditions())

		// reconcile
		err = r.Reconcile(context.Background())
		require.NoError(t, err)

		// assert extensions token is deleted after reconciliation
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: testName + "-extensions-token", Namespace: testNamespace}, &secretFound)
		require.True(t, k8serrors.IsNotFound(err))

		// assert conditions are empty
		require.Empty(t, dk.Conditions())
	})
	t.Run("Extension secret is generated when Prometheus is enabled", func(t *testing.T) {
		dk := createDynakube()
		dk.Spec.Extensions.Enabled = true

		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		// assert extensions token is generated
		var secretFound corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: testName + "-extensions-token", Namespace: testNamespace}, &secretFound)
		require.NoError(t, err)
		require.NotEmpty(t, secretFound.Data[consts.EecTokenSecretKey])
		require.NotEmpty(t, secretFound.Data[consts.OtelcTokenSecretKey])

		// assert extensions token condition is added
		require.NotEmpty(t, dk.Conditions())

		condition := meta.FindStatusCondition(*dk.Conditions(), consts.ExtensionsSecretConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, dk.ExtensionsTokenSecretName()+" created", condition.Message)
	})
	t.Run(`Extension SecretCreated failure condition is set when error`, func(t *testing.T) {
		dk := createDynakube()
		dk.Spec.Extensions.Enabled = true

		misconfiguredReader, _ := client.New(&rest.Config{}, client.Options{})
		r := NewReconciler(fake.NewClient(), misconfiguredReader, dk)
		err := r.Reconcile(context.Background())
		require.Error(t, err)

		// assert extensions token condition is added
		require.NotEmpty(t, dk.Conditions())

		condition := meta.FindStatusCondition(*dk.Conditions(), consts.ExtensionsSecretConditionType)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, conditions.KubeApiErrorReason, condition.Reason)
		assert.Contains(t, condition.Message, "A problem occurred when using the Kubernetes API")
	})

	t.Run("Create service when extensions are enabled with minimal setup", func(t *testing.T) {
		dk := createDynakube()
		dk.Spec.Extensions.Enabled = true

		mockK8sClient := fake.NewClient(dk)

		r := &reconciler{client: mockK8sClient, apiReader: mockK8sClient, dk: dk, timeProvider: timeprovider.New()}
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var svc corev1.Service
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: servicename.Build(r.dk), Namespace: testNamespace}, &svc)
		require.NoError(t, err)
		assert.NotNil(t, svc)

		// assert extensions token condition is added
		require.NotEmpty(t, dk.Conditions())

		condition := meta.FindStatusCondition(*dk.Conditions(), consts.ExtensionsServiceConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.ServiceCreatedReason, condition.Reason)
		assert.Equal(t, dk.Name+consts.ExtensionsControllerSuffix+" created", condition.Message)
	})

	t.Run("Don't create service when extensions are disabled with minimal setup", func(t *testing.T) {
		dk := createDynakube()
		dk.Spec.Extensions.Enabled = false

		mockK8sClient := fake.NewClient(dk)

		r := &reconciler{client: mockK8sClient, apiReader: mockK8sClient, dk: dk, timeProvider: timeprovider.New()}
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var svc corev1.Service
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: servicename.Build(r.dk), Namespace: testNamespace}, &svc)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func createDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynakube.DynaKubeSpec{},
		Status: dynakube.DynaKubeStatus{
			ActiveGate: activegate.Status{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: "abc",
				},
			},
			KubeSystemUUID: "abc",
		},
	}
}
