package oaconnectioninfo

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
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
		dynakube := getTestDynakube()
		dynakube.Status.OneAgent.ConnectionInfoStatus = dynatracev1beta1.OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		conditions.SetSecretCreatedCondition(dynakube.Conditions(), oaConnectionInfoConditionType, "testing")

		dynakube.Spec = dynatracev1beta1.DynaKubeSpec{}

		fakeClient := fake.NewClient()
		dtc := dtclientmock.NewClient(t)

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)
		assert.Empty(t, dynakube.Status.OneAgent.ConnectionInfoStatus)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConnectionInfoConditionType)
		require.Nil(t, condition)
	})

	t.Run("set correct condition on dynatrace-client error", func(t *testing.T) {
		dynakube := getTestDynakube()
		fakeClient := fake.NewClient()
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(dtclient.OneAgentConnectionInfo{}, errors.New("BOOM"))
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.DynatraceApiErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})

	t.Run("set correct condition on kube-client error", func(t *testing.T) {
		dynakube := getTestDynakube()
		fakeClient := createFailK8sClient()
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.KubeApiErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})

	t.Run("store OneAgent connection info to DynaKube status + create secret", func(t *testing.T) {
		dynakube := getTestDynakube()
		fakeClient := fake.NewClient(dynakube)
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)
		assert.Equal(t, getTestCommunicationHosts(), dynakube.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("update OneAgent connection info + secret", func(t *testing.T) {
		dynakube := getTestDynakube()
		fakeClient := fake.NewClient(dynakube)
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)

		dynakube.Status.OneAgent.ConnectionInfoStatus = dynatracev1beta1.OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		conditions.SetSecretCreatedCondition(dynakube.Conditions(), oaConnectionInfoConditionType, "testing")

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		rec := r.(*reconciler)
		rec.timeProvider.Set(rec.timeProvider.Now().Add(time.Minute * 20))

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.NotEqual(t, "testing", condition.Message)
	})
	t.Run("do not update OneAgent connection info within timeout", func(t *testing.T) {
		dynakube := getTestDynakube()
		fakeClient := fake.NewClient(dynakube, buildOneAgentTenantSecret(dynakube, testOutdated))
		dtc := dtclientmock.NewClient(t)
		dynakube.Status.OneAgent.ConnectionInfoStatus = dynatracev1beta1.OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		conditions.SetSecretCreatedCondition(dynakube.Conditions(), oaConnectionInfoConditionType, "testing")

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testOutdated, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testOutdated, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testOutdated), actualSecret.Data[connectioninfo.TenantTokenKey])

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, "testing", condition.Message)
	})
	t.Run("update OneAgent connection info if tenant secret is missing, ignore timestamp", func(t *testing.T) {
		dynakube := getTestDynakube()
		fakeClient := fake.NewClient(dynakube)
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)

		dynakube.Status.OneAgent.ConnectionInfoStatus = dynatracev1beta1.OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		conditions.SetSecretCreatedCondition(dynakube.Conditions(), oaConnectionInfoConditionType, "testing")

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.NotEqual(t, "testing", condition.Message)
	})

	t.Run("update OneAgent connection info in case conditions is in 'False' state ", func(t *testing.T) {
		dynakube := getTestDynakube()
		fakeClient := fake.NewClient(dynakube, buildOneAgentTenantSecret(dynakube, testOutdated))
		dtc := dtclientmock.NewClient(t)
		dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)

		dynakube.Status.OneAgent.ConnectionInfoStatus = dynatracev1beta1.OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}
		setEmptyCommunicationHostsCondition(dynakube.Conditions())

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
}

func TestReconcile_NoOneAgentCommunicationHosts(t *testing.T) {
	ctx := context.Background()
	dynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
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

	fakeClient := fake.NewClient(&dynakube)

	r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, &dynakube)
	err := r.Reconcile(ctx)
	require.ErrorIs(t, err, NoOneAgentCommunicationHostsError)

	assert.Equal(t, testTenantUUID, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
	assert.Empty(t, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)
	assert.Empty(t, dynakube.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts)

	condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConnectionInfoConditionType)
	require.NotNil(t, condition)
	assert.Equal(t, EmptyCommunicationHostsReason, condition.Reason)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
}

func getTestDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
			},
		},
	}
}

func buildOneAgentTenantSecret(dynakube *dynatracev1beta1.DynaKube, token string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakube.OneagentTenantSecret(),
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			connectioninfo.TenantTokenKey: []byte(token),
		},
	}
}

func getTestCommunicationHosts() []dynatracev1beta1.CommunicationHostStatus {
	return []dynatracev1beta1.CommunicationHostStatus{
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
