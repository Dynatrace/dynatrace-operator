package oaconnectioninfo

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName            = "test-name"
	testNamespace       = "test-namespace"
	testTenantToken     = "test-token"
	testTenantUUID      = "test-uuid"
	testTenantEndpoints = "test-endpoints"
	testOutdated        = "outdated"
)

var testCommunicationHosts = []dynatracev1beta1.CommunicationHostStatus{
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

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	dynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		}}

	dtc := mocks.NewClient(t)
	dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil).Maybe()

	t.Run(`store OneAgent connection info to DynaKube status`, func(t *testing.T) {
		fakeClient := fake.NewClient(&dynakube)
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, &dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)
		assert.Equal(t, testCommunicationHosts, dynakube.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts)
	})
	t.Run(`update OneAgent connection info`, func(t *testing.T) {
		fakeClient := fake.NewClient(&dynakube)
		dynakube.Status.OneAgent.ConnectionInfoStatus = dynatracev1beta1.OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID:  testOutdated,
				Endpoints:   testOutdated,
				LastRequest: metav1.NewTime(time.Now()),
			},
		}
		resetCachedTimestamps(&dynakube.Status)

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, &dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)
	})
	t.Run(`do not update OneAgent connection info within timeout`, func(t *testing.T) {
		fakeClient := fake.NewClient(&dynakube, buildOneAgentTenantSecret(dynakube, testOutdated))
		dynakube.Status.OneAgent.ConnectionInfoStatus = dynatracev1beta1.OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID:  testOutdated,
				Endpoints:   testOutdated,
				LastRequest: metav1.NewTime(time.Now()),
			},
		}

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, &dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testOutdated, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testOutdated, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)
	})
	t.Run(`update OneAgent connection info if tenant secret is missing, ignore timestamp`, func(t *testing.T) {
		fakeClient := fake.NewClient(&dynakube)
		dynakube.Status.OneAgent.ConnectionInfoStatus = dynatracev1beta1.OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID:  testOutdated,
				Endpoints:   testOutdated,
				LastRequest: metav1.NewTime(time.Now()),
			},
		}

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, &dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)
	})
}

func TestReconcile_NoOneAgentCommunicationHosts(t *testing.T) {
	ctx := context.Background()
	dynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		}}

	dtc := mocks.NewClient(t)
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
}

func getTestOneAgentConnectionInfo() dtclient.OneAgentConnectionInfo {
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

func resetCachedTimestamps(dynakubeStatus *dynatracev1beta1.DynaKubeStatus) {
	dynakubeStatus.DynatraceApi.LastTokenScopeRequest = metav1.Time{}
	dynakubeStatus.OneAgent.ConnectionInfoStatus.LastRequest = metav1.Time{}
}

func TestReconcile_TenantSecret(t *testing.T) {
	ctx := context.Background()
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		}}

	dtc := mocks.NewClient(t)
	dtc.On("GetOneAgentConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestOneAgentConnectionInfo(), nil)

	t.Run(`create oneagent secret`, func(t *testing.T) {
		fakeClient := fake.NewClient(dynakube)

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])
	})
	t.Run(`update oneagent secret`, func(t *testing.T) {
		fakeClient := fake.NewClient(dynakube, buildOneAgentTenantSecret(*dynakube, testOutdated))

		// responses from the Dynatrace API are cached for 15 minutes, so we need to reset the cache here and assume
		// we traveled 15 minutes into the future
		resetCachedTimestamps(&dynakube.Status)

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])
	})
	t.Run(`update oneagent secret, check if caches are used`, func(t *testing.T) {
		fakeClient := fake.NewClient(dynakube, buildOneAgentTenantSecret(*dynakube, testOutdated))

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testOutdated), actualSecret.Data[connectioninfo.TenantTokenKey])
	})
	t.Run(`up to date oneagent secret`, func(t *testing.T) {
		fakeClient := fake.NewClient(dynakube, buildOneAgentTenantSecret(*dynakube, testTenantToken))

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)
	})
}

func buildOneAgentTenantSecret(dynakube dynatracev1beta1.DynaKube, token string) *corev1.Secret {
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
