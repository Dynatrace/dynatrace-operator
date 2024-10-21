package configsecret

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	dkName      = "test-name"
	dkNamespace = "test-namespace"
	tokenValue  = "test-token"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("Create and update works with minimal setup", func(t *testing.T) {
		dk := createDynakube(true)

		mockK8sClient := createK8sClientWithOneAgentTenantSecret(dk, tokenValue)

		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		checkSecretForValue(t, mockK8sClient, dk)

		condition := meta.FindStatusCondition(*dk.Conditions(), lmcConditionType)
		oldTransitionTime := condition.LastTransitionTime
		require.NotNil(t, condition)
		require.NotEmpty(t, oldTransitionTime)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)

		err = reconciler.Reconcile(context.Background())

		require.NoError(t, err)
		checkSecretForValue(t, mockK8sClient, dk)
	})
	t.Run("Only runs when required, and cleans up condition + secret", func(t *testing.T) {
		dk := createDynakube(false)

		mockK8sClient := createK8sClientWithOneAgentTenantSecret(dk, tokenValue)
		conditions.SetSecretCreated(dk.Conditions(), lmcConditionType, "this is a test")

		reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.Empty(t, *dk.Conditions())

		var secretConfig corev1.Secret
		err = mockK8sClient.Get(ctx, types.NamespacedName{
			Name:      GetSecretName(dk.Name),
			Namespace: dk.Namespace,
		}, &secretConfig)
		require.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("problem with k8s request => visible in conditions", func(t *testing.T) {
		dk := createDynakube(true)

		boomClient := createBOOMK8sClient()

		reconciler := NewReconciler(boomClient,
			boomClient, dk)

		err := reconciler.Reconcile(context.Background())

		require.Error(t, err)
		require.Len(t, *dk.Conditions(), 1)
		condition := meta.FindStatusCondition(*dk.Conditions(), lmcConditionType)
		assert.Equal(t, conditions.KubeApiErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func checkSecretForValue(t *testing.T, k8sClient client.Client, dk *dynakube.DynaKube) {
	var secret corev1.Secret
	err := k8sClient.Get(context.Background(), client.ObjectKey{Name: GetSecretName((dk.Name)), Namespace: dk.Namespace}, &secret)
	require.NoError(t, err)

	deploymentConfig, ok := secret.Data[DeploymentConfigFilename]
	require.True(t, ok)

	tenantUUID, err := dk.TenantUUIDFromConnectionInfoStatus()
	require.NoError(t, err)

	expectedLines := []string{
		serverKey + "=" + fmt.Sprintf("{%s}", dk.Status.OneAgent.ConnectionInfoStatus.Endpoints),
		tenantKey + "=" + tenantUUID,
		tenantTokenKey + "=" + tokenValue,
		hostIdSourceKey + "=k8s-node-name",
	}

	split := strings.Split(strings.Trim(string(deploymentConfig), "\n"), "\n")
	require.Len(t, split, len(expectedLines))

	for _, line := range split {
		assert.Contains(t, expectedLines, line)
	}
}

func createDynakube(isLogMonitoringEnabled bool) *dynakube.DynaKube {
	var logMonitoringSpec *logmonitoring.Spec
	if isLogMonitoringEnabled {
		logMonitoringSpec = &logmonitoring.Spec{}
	}

	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dkNamespace,
			Name:      dkName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:        "test-url",
			LogMonitoring: logMonitoringSpec,
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: dynakube.OneAgentStatus{
				ConnectionInfoStatus: dynakube.OneAgentConnectionInfoStatus{
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: "test-uuid",
						Endpoints:  "https://endpoint1.com;https://endpoint2.com",
					},
				},
			},
		},
	}
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

func createK8sClientWithOneAgentTenantSecret(dk *dynakube.DynaKube, token string) client.Client {
	mockK8sClient := fake.NewClient()
	_ = mockK8sClient.Create(context.Background(),
		&corev1.Secret{
			Data: map[string][]byte{connectioninfo.TenantTokenKey: []byte(token)},
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.OneagentTenantSecret(),
				Namespace: dkNamespace,
			},
		},
	)

	return mockK8sClient
}
