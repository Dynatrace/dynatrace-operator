package oaconnectioninfo

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
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

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func TestReconcile(t *testing.T) {
	ctx := t.Context()

	assertCondition := func(t *testing.T, dk *dynakube.DynaKube, status metav1.ConditionStatus, reason string, message ...string) {
		t.Helper()
		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, status, condition.Status)
		assert.Equal(t, reason, condition.Reason)
		if message != nil {
			assert.Equal(t, message[0], condition.Message)
		}
	}

	t.Run("cleanup when oneagent is not needed", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Status.OneAgent.ConnectionInfoStatus = oneagent.ConnectionInfoStatus{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		conditions.SetSecretCreated(dk.Conditions(), oaConnectionInfoConditionType, "testing")

		dk.Spec = dynakube.DynaKubeSpec{}

		fakeClient := fake.NewClient(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: dk.OneAgent().GetTenantSecret(), Namespace: dk.Namespace}})
		dtc := dtclientmock.NewClient(t)

		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.NoError(t, err)
		assert.Empty(t, dk.Status.OneAgent.ConnectionInfoStatus)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.Nil(t, condition)

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneAgent().GetTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("does not cleanup when only host oneagent is needed", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Status.OneAgent.ConnectionInfoStatus = oneagent.ConnectionInfoStatus{
			ConnectionInfo: communication.ConnectionInfo{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		dk.Spec = dynakube.DynaKubeSpec{}
		dk.Spec.OneAgent.ClassicFullStack = &oneagent.HostInjectSpec{}

		conditions.SetSecretCreated(dk.Conditions(), oaConnectionInfoConditionType, "testing")

		fakeClient := fake.NewClient(dk)
		dtc := dtclientmock.NewClient(t)
		dtc.EXPECT().GetOneAgentConnectionInfo(anyCtx).Return(getTestOneAgentConnectionInfo(), nil).Once()

		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, dk.Status.OneAgent.ConnectionInfoStatus)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assertCondition(t, dk, metav1.ConditionTrue, conditions.SecretCreatedReason)
	})

	t.Run("set correct condition on dynatrace-client error", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient()
		dtc := dtclientmock.NewClient(t)
		dtc.EXPECT().GetOneAgentConnectionInfo(anyCtx).Return(dtclient.OneAgentConnectionInfo{}, errors.New("BOOM")).Once()
		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.Error(t, err)

		assertCondition(t, dk, metav1.ConditionFalse, conditions.DynatraceAPIErrorReason)
	})

	t.Run("set correct condition on kube-client error", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := createFailK8sClient()
		dtc := dtclientmock.NewClient(t)
		dtc.EXPECT().GetOneAgentConnectionInfo(anyCtx).Return(getTestOneAgentConnectionInfo(), nil).Once()
		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.Error(t, err)

		assertCondition(t, dk, metav1.ConditionFalse, conditions.KubeAPIErrorReason)
	})

	t.Run("store OneAgent connection info to DynaKube status + create secret", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk)
		dtc := dtclientmock.NewClient(t)
		dtc.EXPECT().GetOneAgentConnectionInfo(anyCtx).Return(getTestOneAgentConnectionInfo(), nil).Once()
		r := NewReconciler(fakeClient, fakeClient, dtc, dk)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		tenantTokenHash, err := hasher.GenerateHash(testTenantToken)

		require.NoError(t, err)
		assert.Equal(t, testTenantUUID, dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dk.Status.OneAgent.ConnectionInfoStatus.Endpoints)
		assert.Equal(t, getTestCommunicationHosts(), dk.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts)
		assert.Equal(t, tenantTokenHash, dk.Status.OneAgent.ConnectionInfoStatus.TenantTokenHash)

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneAgent().GetTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		assertCondition(t, dk, metav1.ConditionTrue, conditions.SecretCreatedReason)
	})
	t.Run("update OneAgent connection info + secret", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk)
		dtc := dtclientmock.NewClient(t)
		dtc.EXPECT().GetOneAgentConnectionInfo(anyCtx).Return(getTestOneAgentConnectionInfo(), nil).Once()

		dk.Status.OneAgent.ConnectionInfoStatus = oneagent.ConnectionInfoStatus{
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
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneAgent().GetTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		assertCondition(t, dk, metav1.ConditionTrue, conditions.SecretCreatedReason, dk.OneAgent().GetTenantSecret()+" created")
	})
	t.Run("do not update OneAgent connection info within timeout", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk, buildOneAgentTenantSecret(dk, testOutdated))
		dtc := dtclientmock.NewClient(t)

		dk.Status.OneAgent.ConnectionInfoStatus = oneagent.ConnectionInfoStatus{
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
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneAgent().GetTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testOutdated), actualSecret.Data[connectioninfo.TenantTokenKey])

		assertCondition(t, dk, metav1.ConditionTrue, conditions.SecretCreatedReason, "testing created")
	})
	t.Run("update OneAgent connection info if tenant secret is missing, ignore timestamp", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk)
		dtc := dtclientmock.NewClient(t)
		dtc.EXPECT().GetOneAgentConnectionInfo(anyCtx).Return(getTestOneAgentConnectionInfo(), nil).Once()

		dk.Status.OneAgent.ConnectionInfoStatus = oneagent.ConnectionInfoStatus{
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
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneAgent().GetTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		assertCondition(t, dk, metav1.ConditionTrue, conditions.SecretCreatedReason, dk.OneAgent().GetTenantSecret()+" created")
	})

	t.Run("update OneAgent connection info in case conditions is in 'False' state ", func(t *testing.T) {
		dk := getTestDynakube()
		fakeClient := fake.NewClient(dk, buildOneAgentTenantSecret(dk, testOutdated))
		dtc := dtclientmock.NewClient(t)
		dtc.EXPECT().GetOneAgentConnectionInfo(anyCtx).Return(getTestOneAgentConnectionInfo(), nil).Once()

		dk.Status.OneAgent.ConnectionInfoStatus = oneagent.ConnectionInfoStatus{
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
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dk.OneAgent().GetTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		assertCondition(t, dk, metav1.ConditionTrue, conditions.SecretCreatedReason)
	})
}

func TestReconcile_NoOneAgentCommunicationHosts(t *testing.T) {
	ctx := t.Context()
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{
				CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
			},
		},
	}

	dtc := dtclientmock.NewClient(t)
	dtc.EXPECT().GetOneAgentConnectionInfo(anyCtx).Return(dtclient.OneAgentConnectionInfo{
		ConnectionInfo: dtclient.ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			Endpoints:   "",
		},
		CommunicationHosts: nil,
	}, nil)

	fakeClient := fake.NewClient(dk)

	r := NewReconciler(fakeClient, fakeClient, dtc, dk)
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
			OneAgent: oneagent.Spec{
				CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
			},
		},
	}
}

func buildOneAgentTenantSecret(dk *dynakube.DynaKube, token string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.OneAgent().GetTenantSecret(),
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			connectioninfo.TenantTokenKey: []byte(token),
		},
	}
}

func getTestCommunicationHosts() []oneagent.CommunicationHostStatus {
	return []oneagent.CommunicationHostStatus{
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
		Create: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.CreateOption) error {
			return errors.New("BOOM")
		},
		Delete: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.DeleteOption) error {
			return errors.New("BOOM")
		},
		Update: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.UpdateOption) error {
			return errors.New("BOOM")
		},
		Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
			return errors.New("BOOM")
		},
	})

	return boomClient
}
