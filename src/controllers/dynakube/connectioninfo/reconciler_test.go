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

	tenantInfoResponse := &dtclient.ActiveGateConnectionInfo{
		ConnectionInfo: dtclient.ConnectionInfo{
			TenantUUID:  testTenantUuid,
			TenantToken: testTenantToken,
			Endpoints:   testTenantEndpoints,
		},
	}
	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetActiveGateConnectionInfo").Return(tenantInfoResponse, nil)

	t.Run(`create activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		_, err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
		assert.Equal(t, []byte(testTenantUuid), actualSecret.Data[TenantUuidName])
		assert.Equal(t, []byte(testTenantEndpoints), actualSecret.Data[CommunicationEndpointsName])
	})
	t.Run(`update activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.ActivegateTenantSecret(),
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					TenantTokenName:            []byte("outdated"),
					TenantUuidName:             []byte("outdated"),
					CommunicationEndpointsName: []byte("outdated"),
				},
			},
		).Build()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		_, err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
		assert.Equal(t, []byte(testTenantUuid), actualSecret.Data[TenantUuidName])
		assert.Equal(t, []byte(testTenantEndpoints), actualSecret.Data[CommunicationEndpointsName])
	})
	t.Run(`up to date activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.ActivegateTenantSecret(),
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					TenantTokenName:            []byte(testTenantToken),
					TenantUuidName:             []byte(testTenantUuid),
					CommunicationEndpointsName: []byte(testTenantEndpoints),
				},
			},
		).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		upd, err := r.Reconcile()
		require.NoError(t, err)
		assert.False(t, upd)
	})
}

func TestReconcile_OneagentSecret(t *testing.T) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureActiveGateRawImage:     "false",
				dynatracev1beta1.AnnotationFeatureOneAgentImmutableImage: "true",
			},
		}}

	connectionInfo := dtclient.OneAgentConnectionInfo{
		ConnectionInfo: dtclient.ConnectionInfo{
			TenantUUID:  testTenantUuid,
			TenantToken: testTenantToken,
			Endpoints:   testTenantEndpoints,
		},
	}
	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetOneAgentConnectionInfo").Return(connectionInfo, nil)

	t.Run(`create oneagent secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		_, err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
		assert.Equal(t, []byte(testTenantUuid), actualSecret.Data[TenantUuidName])
		assert.Equal(t, []byte(testTenantEndpoints), actualSecret.Data[CommunicationEndpointsName])
	})
	t.Run(`update oneagent secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.OneagentTenantSecret(),
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					TenantTokenName:            []byte("outdated"),
					TenantUuidName:             []byte("outdated"),
					CommunicationEndpointsName: []byte("outdated"),
				},
			},
		).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		_, err := r.Reconcile()
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.OneagentTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[TenantTokenName])
		assert.Equal(t, []byte(testTenantUuid), actualSecret.Data[TenantUuidName])
		assert.Equal(t, []byte(testTenantEndpoints), actualSecret.Data[CommunicationEndpointsName])
	})
	t.Run(`up to date oneagent secret`, func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakube.OneagentTenantSecret(),
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					TenantTokenName:            []byte(testTenantToken),
					TenantUuidName:             []byte(testTenantUuid),
					CommunicationEndpointsName: []byte(testTenantEndpoints),
				},
			},
		).Build()

		r := NewReconciler(context.TODO(), fakeClient, fakeClient, dynakube, dtc)
		upd, err := r.Reconcile()
		require.NoError(t, err)
		assert.False(t, upd)
	})
}
