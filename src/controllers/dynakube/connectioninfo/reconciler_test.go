package connectioninfo

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
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
	testTenantUuid      = "test-uuid"
	testTenantEndpoints = "test-endpoints"
)

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
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
	})
	t.Run(`update activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.ActivegateTenantSecret(),
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					TenantTokenName: []byte("outdated"),
				},
			},
		).Build()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
	})
	t.Run(`up to date activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.ActivegateTenantSecret(),
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					TenantTokenName: []byte(testTenantToken),
				},
			},
		).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)
	})
}

func TestReconcile_ActivegateConfigMap(t *testing.T) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		}}

	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetActiveGateConnectionInfo").Return(getTestActiveGateConnectionInfo(), nil)
	dtc.On("GetOneAgentConnectionInfo").Return(getTestOneAgentConnectionInfo(), nil)

	t.Run(`create activegate ConfigMap`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actual corev1.ConfigMap
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.ActiveGateConnectionInfoConfigMapName(), Namespace: testNamespace}, &actual)
		require.NoError(t, err)
		assert.Equal(t, testTenantEndpoints, actual.Data[CommunicationEndpointsName])
	})
	t.Run(`update activegate ConfigMap`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.ActiveGateConnectionInfoConfigMapName(),
					Namespace: testNamespace,
				},
				Data: map[string]string{
					CommunicationEndpointsName: "outdated",
				},
			},
		).Build()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actual corev1.ConfigMap
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.ActiveGateConnectionInfoConfigMapName(), Namespace: testNamespace}, &actual)
		require.NoError(t, err)
		assert.Equal(t, testTenantEndpoints, actual.Data[CommunicationEndpointsName])
	})
	t.Run(`up to date activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.ActiveGateConnectionInfoConfigMapName(),
					Namespace: testNamespace,
				},
				Data: map[string]string{
					CommunicationEndpointsName: testTenantEndpoints,
				},
			},
		).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)
	})
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

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
	})
	t.Run(`update oneagent secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.OneagentTenantSecret(),
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					TenantTokenName: []byte("outdated"),
				},
			},
		).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
	})
	t.Run(`up to date oneagent secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.OneagentTenantSecret(),
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					TenantTokenName: []byte(testTenantToken),
				},
			},
		).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)
	})
}

func TestReconcile_OneagentConfigMap(t *testing.T) {
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

	t.Run(`create oneagent ConfigMap`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actual corev1.ConfigMap
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.OneAgentConnectionInfoConfigMapName(), Namespace: testNamespace}, &actual)
		require.NoError(t, err)
		assert.Equal(t, testTenantEndpoints, actual.Data[CommunicationEndpointsName])
	})
	t.Run(`update oneagent ConfigMap`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.OneAgentConnectionInfoConfigMapName(),
					Namespace: testNamespace,
				},
				Data: map[string]string{
					CommunicationEndpointsName: "outdated",
				},
			},
		).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var actual corev1.ConfigMap
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.OneAgentConnectionInfoConfigMapName(), Namespace: testNamespace}, &actual)
		require.NoError(t, err)
		assert.Equal(t, testTenantEndpoints, actual.Data[CommunicationEndpointsName])
	})
	t.Run(`up to date oneagent ConfigMap`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.OneAgentConnectionInfoConfigMapName(),
					Namespace: testNamespace,
				},
				Data: map[string]string{
					CommunicationEndpointsName: testTenantEndpoints,
				},
			},
		).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		err := r.Reconcile()
		require.NoError(t, err)
	})
}

func getTestOneAgentConnectionInfo() dtclient.OneAgentConnectionInfo {
	return dtclient.OneAgentConnectionInfo{
		ConnectionInfo: dtclient.ConnectionInfo{
			TenantUUID:  testTenantUuid,
			TenantToken: testTenantToken,
			Endpoints:   testTenantEndpoints,
		},
	}
}

func getTestActiveGateConnectionInfo() *dtclient.ActiveGateConnectionInfo {
	return &dtclient.ActiveGateConnectionInfo{
		ConnectionInfo: dtclient.ConnectionInfo{
			TenantUUID:  testTenantUuid,
			TenantToken: testTenantToken,
			Endpoints:   testTenantEndpoints,
		},
	}
}
