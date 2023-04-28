package connectioninfo

import (
	"context"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

func TestReconcile_ConnectionInfo(t *testing.T) {
	dynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		}}

	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetActiveGateConnectionInfo").Return(getTestActiveGateConnectionInfo(), nil)
	dtc.On("GetOneAgentConnectionInfo").Return(getTestOneAgentConnectionInfo(), nil)

	t.Run(`store OneAgent connection info to DynaKube status`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, &dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)
		assert.Equal(t, testCommunicationHosts, dynakube.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts)
	})
	t.Run(`update OneAgent connection info`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()
		dynakube.Status.OneAgent.ConnectionInfoStatus = dynatracev1beta1.OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID:  testOutdated,
				Endpoints:   testOutdated,
				LastRequest: metav1.NewTime(time.Now()),
			},
		}
		resetCachedTimestamps(&dynakube.Status)

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, &dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)
	})
	t.Run(`do not update OneAgent connection info within timeout`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()
		dynakube.Status.OneAgent.ConnectionInfoStatus = dynatracev1beta1.OneAgentConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID:  testOutdated,
				Endpoints:   testOutdated,
				LastRequest: metav1.NewTime(time.Now()),
			},
		}

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, &dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		assert.Equal(t, testOutdated, dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testOutdated, dynakube.Status.OneAgent.ConnectionInfoStatus.Endpoints)
	})

	t.Run(`store ActiveGate connection info to DynaKube status`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, &dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints)
	})
	t.Run(`update ActiveGate connection info`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()
		dynakube.Status.ActiveGate.ConnectionInfoStatus = dynatracev1beta1.ActiveGateConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID:  testOutdated,
				Endpoints:   testOutdated,
				LastRequest: metav1.NewTime(time.Now()),
			},
		}
		resetCachedTimestamps(&dynakube.Status)
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, &dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		assert.Equal(t, testTenantUUID, dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints)
	})
	t.Run(`do not update ActiveGate connection info within timeout`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()
		dynakube.Status.ActiveGate.ConnectionInfoStatus = dynatracev1beta1.ActiveGateConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID:  testOutdated,
				Endpoints:   testOutdated,
				LastRequest: metav1.NewTime(time.Now()),
			},
		}

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, &dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		assert.Equal(t, testOutdated, dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testOutdated, dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints)
	})
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

func getTestActiveGateConnectionInfo() *dtclient.ActiveGateConnectionInfo {
	return &dtclient.ActiveGateConnectionInfo{
		ConnectionInfo: dtclient.ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			Endpoints:   testTenantEndpoints,
		},
	}
}

func resetCachedTimestamps(dynakubeStatus *dynatracev1beta1.DynaKubeStatus) {
	dynakubeStatus.DynatraceApi.LastTokenScopeRequest = metav1.Time{}
	dynakubeStatus.OneAgent.ConnectionInfoStatus.LastRequest = metav1.Time{}
	dynakubeStatus.ActiveGate.ConnectionInfoStatus.LastRequest = metav1.Time{}
}

func TestReconcile_ActivegateSecret(t *testing.T) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		}}
	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetActiveGateConnectionInfo").Return(getTestActiveGateConnectionInfo(), nil)
	dtc.On("GetOneAgentConnectionInfo").Return(getTestOneAgentConnectionInfo(), nil)

	t.Run(`create activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
	})
	t.Run(`update activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(buildActiveGateSecret(*dynakube, testOutdated)).Build()
		resetCachedTimestamps(&dynakube.Status)
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
	})
	t.Run(`check activegate secret caches`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(buildActiveGateSecret(*dynakube, testOutdated)).Build()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testOutdated), actualSecret.Data[TenantTokenName])
	})
	t.Run(`up to date activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(buildActiveGateSecret(*dynakube, testTenantToken)).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)
	})
}

func buildActiveGateSecret(dynakube dynatracev1beta1.DynaKube, token string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakube.ActivegateTenantSecret(),
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			TenantTokenName: []byte(token),
		},
	}
}

func TestReconcile_OneagentSecret(t *testing.T) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureActiveGateRawImage: "false",
			},
		}}

	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetOneAgentConnectionInfo").Return(getTestOneAgentConnectionInfo(), nil)

	t.Run(`create oneagent secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
	})
	t.Run(`update oneagent secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(buildOneAgentTenantSecret(*dynakube, testOutdated)).Build()

		// responses from the Dynatrace API are cached for 15 minutes, so we need to reset the cache here and assume
		// we traveled 15 minutes into the future
		resetCachedTimestamps(&dynakube.Status)

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
	})
	t.Run(`update oneagent secret, check if caches are used`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(buildOneAgentTenantSecret(*dynakube, testOutdated)).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testOutdated), actualSecret.Data[TenantTokenName])
	})
	t.Run(`up to date oneagent secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(buildOneAgentTenantSecret(*dynakube, testTenantToken)).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dtc)
		err := r.Reconcile()
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
			TenantTokenName: []byte(token),
		},
	}
}
