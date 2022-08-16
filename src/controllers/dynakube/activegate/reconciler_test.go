package activegate

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
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

var testKubeSystemNamespace = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "kube-system",
		UID:  "01234-5678-9012-3456",
	},
}

func TestReconciler_Reconcile(t *testing.T) {
	t.Run(`Reconcile works with minimal setup`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			}}
		fakeClient := fake.NewClient()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance)
		upd, err := r.Reconcile()
		assert.NoError(t, err)
		assert.True(t, upd)
	})
	t.Run(`Reconcile AG proxy secret`, func(t *testing.T) {
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
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance)
		upd, err := r.Reconcile()
		assert.NoError(t, err)
		assert.True(t, upd)

		var proxySecret corev1.Secret
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: "dynatrace-activegate-internal-proxy", Namespace: testNamespace}, &proxySecret)
		assert.NoError(t, err)
	})
	t.Run(`Reconcile AG capability (creation and deletion)`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.RoutingCapability.DisplayName},
				},
			},
		}
		fakeClient := fake.NewClient(testKubeSystemNamespace)
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance)
		upd, err := r.Reconcile()
		assert.NoError(t, err)
		assert.True(t, upd)

		var service corev1.Service
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.NoError(t, err)

		// remove AG from spec
		instance.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{}
		upd, err = r.Reconcile()
		assert.NoError(t, err)
		assert.True(t, upd)
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, errors.IsNotFound(err))
	})
}
