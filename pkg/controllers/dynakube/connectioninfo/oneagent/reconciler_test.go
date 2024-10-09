package oaconnectioninfo

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testName            = "test-name"
	testNamespace       = "test-namespace"
	testTenantToken     = "test-token"
	testTenantUUID      = "test-uuid"
	testTenantEndpoints = "test-endpoints"
	testOutdated        = "outdated"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("cleanup when oneagent is not needed", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Status.OneAgent.ConnectionInfoStatus = dynakube.OneAgentConnectionInfoStatus{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		conditions.SetSecretCreated(dk.Conditions(), oaConnectionInfoConditionType, "testing")

		dk.Spec = dynakube.DynaKubeSpec{}

		fakeClient := fake.NewClient(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: dk.OneagentTenantSecret(), Namespace: dk.Namespace}})
		dtc := dtclientmock.NewClient(t)

		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.NoError(t, err)
		assert.Empty(t, dk.Status.OneAgent.ConnectionInfoStatus)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.Nil(t, condition)

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("does not cleanup when only host oneagent is needed", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Status.OneAgent.ConnectionInfoStatus = dynakube.OneAgentConnectionInfoStatus{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		dk.Spec = dynakube.DynaKubeSpec{}
		dk.Spec.OneAgent.ClassicFullStack = &dynakube.HostInjectSpec{}

		conditions.SetSecretCreated(dk.Conditions(), oaConnectionInfoConditionType, "testing")

		fakeClient := fake.NewClient(dk)
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)

		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, dk.Status.OneAgent.ConnectionInfoStatus)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
	})

	t.Run("set correct condition on dynatrace-client error", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient()
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(dtclient.OneAgentConnectionInfo{}, errors.New("BOOM"))
		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.DynatraceApiErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})

	t.Run("set correct condition on kube-client error", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := createFailK8sClient()
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)
		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.KubeApiErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})

	t.Run("store OneAgent connection info to DynaKube status + create secret", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk)
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)
		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dk.Status.OneAgent.ConnectionInfoStatus.Endpoints)
		assert.Equal(t, getTestCommunicationHosts(), dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts)

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("update OneAgent connection info + secret", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk)
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)

		dk.Status.OneAgent.ConnectionInfoStatus = dynakube.OneAgentConnectionInfoStatus{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		conditions.SetSecretCreated(dk.Conditions(), oaConnectionInfoConditionType, "testing")

		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		rec := r.(*reconciler)
		rec.timeProvider.Set(rec.timeProvider.Now().Add(time.Minute * 20))

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dk.Status.OneAgent.ConnectionInfoStatus.Endpoints)

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.NotEqual(t, "testing", condition.Message)
	})
	t.Run("do not update OneAgent connection info within timeout", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.DynatraceApiRequestThreshold = dynakube.DefaultMinRequestThresholdMinutes
		fakeClient := fake.NewClient(dk, buildOneAgentTenantSecret(dk, testOutdated))
		dtc := dtclientmock.NewClient(t)

		dk.Status.OneAgent.ConnectionInfoStatus = dynakube.OneAgentConnectionInfoStatus{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		conditions.SetSecretCreated(dk.Conditions(), oaConnectionInfoConditionType, "testing")

		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testOutdated, dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testOutdated, dk.Status.OneAgent.ConnectionInfoStatus.Endpoints)

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testOutdated), actualSecret.Data[connectioninfo.TenantTokenKey])

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, "testing created", condition.Message)
	})
	t.Run("update OneAgent connection info if tenant secret is missing, ignore timestamp", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk)
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)

		dk.Status.OneAgent.ConnectionInfoStatus = dynakube.OneAgentConnectionInfoStatus{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		conditions.SetSecretCreated(dk.Conditions(), oaConnectionInfoConditionType, "testing")

		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dk.Status.OneAgent.ConnectionInfoStatus.Endpoints)

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.NotEqual(t, "testing", condition.Message)
	})

	t.Run("update OneAgent connection info in case conditions is in 'False' state ", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk, buildOneAgentTenantSecret(dk, testOutdated))
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)

		dk.Status.OneAgent.ConnectionInfoStatus = dynakube.OneAgentConnectionInfoStatus{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		setEmptyCommunicationHostsCondition(dk.Conditions())

		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dk.Status.OneAgent.ConnectionInfoStatus.Endpoints)

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
}

func TestReconcile_NoOneAgentCommunicationHosts(t *testing.T) {
	ctx := context.Background()
	dk := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynakube.DynaKubeSpec{
			OneAgent: dynakube.OneAgentSpec{
				CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
			},
		},
	}

	dtc := dtclientmock.NewClient(t)
	dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(dtclient.OneAgentConnectionInfo{
		ConnectionInfo: dtclient.ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			Endpoints:   "",
		},
		CommunicationHosts: nil,
	}, nil)

	fakeClient := fake.NewClient(&dk)

	r := NewReconciler(fakeClient, fakeClient, dtc, &dk)
	err := r.Reconcile(ctx)
	require.ErrorIs(t, err, NoOneAgentCommunicationHostsError)

	assert.Equal(t, testTenantUUID, dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
	assert.Empty(t, dk.Status.OneAgent.ConnectionInfoStatus.Endpoints)
	assert.Empty(t, dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts)

	condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
	require.NotNil(t, condition)
	assert.Equal(t, EmptyCommunicationHostsReason, condition.Reason)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
}

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynakube.DynaKubeSpec{
			OneAgent: dynakube.OneAgentSpec{
				CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
			},
		},
	}
}

func buildOneAgentTenantSecret(dk *dynakube.DynaKube, token string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.OneagentTenantSecret(),
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			connectioninfo.TenantTokenKey: []byte(token),
		},
	}
}

func getTestCommunicationHosts() []dynakube.CommunicationHostStatus {
	return []dynakube.CommunicationHostStatus{
		{
			Protocol: "http",
			Host:     "dummyhost",
			Port:     42,
		},
		{
			Protocol: "https",
			Host:     "foobarhost",
			Port:     84,
		},
	}
}

func getTestOneAgentConnectionInfo() dtclient.OneAgentConnectionInfo {
	testCommunicationHosts := getTestCommunicationHosts()

	return dtclient.OneAgentConnectionInfo{
		ConnectionInfo: dtclient.ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			Endpoints:   testTenantEndpoints,
		},
		CommunicationHosts: []dtclient.CommunicationHost{
			{
				Protocol: testCommunicationHosts[0].Protocol,
				Host:     testCommunicationHosts[0].Host,
				Port:     testCommunicationHosts[0].Port,
			},
			{
				Protocol: testCommunicationHosts[1].Protocol,
				Host:     testCommunicationHosts[1].Host,
				Port:     testCommunicationHosts[1].Port,
			},
		},
	}
}

func createFailK8sClient() client.Client {
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
