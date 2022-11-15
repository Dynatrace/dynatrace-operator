package activegate

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	testName        = "test-name"
	testNamespace   = "test-namespace"
	testProxyName   = "test-proxy"
	testServiceName = testName + "-activegate"
)

var (
	testKubeSystemNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
			UID:  "01234-5678-9012-3456",
		},
	}
)

func TestReconciler_Reconcile(t *testing.T) {
	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

	t.Run(`Create works with minimal setup`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			}}
		fakeClient := fake.NewClient()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile()
		require.NoError(t, err)
	})
	t.Run(`Create AG proxy secret`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{Value: testProxyName},
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var proxySecret corev1.Secret
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: "dynatrace-activegate-internal-proxy", Namespace: testNamespace}, &proxySecret)
		assert.NoError(t, err)
	})
	t.Run(`Create AG capability (creation and deletion)`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.StatsdIngestCapability.DisplayName},
				},
			},
		}
		fakeClient := fake.NewClient(testKubeSystemNamespace)
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		// assert that service is created if statsd is enabled
		var service corev1.Service
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)

		// assert that service is created if other capabilities are enabled
		instance.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.RoutingCapability.DisplayName}

		err = r.Reconcile()
		require.NoError(t, err)
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)

		// remove AG from spec
		instance.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{}
		err = r.Reconcile()
		require.NoError(t, err)
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, errors.IsNotFound(err))
	})

}
