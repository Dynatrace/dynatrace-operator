// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway_test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/gateway"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestKubemonDisabled(t *testing.T) {
	t.Run("no existing service, reconcile completes without error", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{},
		}
		err := gateway.NewReconciler(fake.NewClient(dk)).Reconcile(t.Context(), dk)
		require.NoError(t, err)
	})

	t.Run("existing service is removed from the cluster", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{},
		}
		existingSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      gateway.ServiceName(dk.Name),
				Namespace: dk.Namespace,
			},
		}
		fakeClient := fake.NewClient(dk, existingSvc)

		err := gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.NoError(t, err)

		svc := &corev1.Service{}
		getErr := fakeClient.Get(t.Context(), client.ObjectKey{Name: gateway.ServiceName(dk.Name), Namespace: dk.Namespace}, svc)
		assert.True(t, k8serrors.IsNotFound(getErr), "service should have been removed from the cluster")
	})
}

func TestKubemonEnabled(t *testing.T) {
	t.Run("no service is created when KSPM is disabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
			},
		}
		fakeClient := fake.NewClient(dk)

		require.NoError(t, gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk))

		svc := &corev1.Service{}
		svcMissing := k8serrors.IsNotFound(fakeClient.Get(t.Context(), client.ObjectKey{Name: gateway.ServiceName(dk.Name), Namespace: dk.Namespace}, svc))
		assert.True(t, svcMissing, "service should not have been created")
	})

	t.Run("existing service is removed when KSPM is disabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
			},
		}
		existingSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      gateway.ServiceName(dk.Name),
				Namespace: dk.Namespace,
			},
		}
		fakeClient := fake.NewClient(dk, existingSvc)

		require.NoError(t, gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk))

		svc := &corev1.Service{}
		svcMissing := k8serrors.IsNotFound(fakeClient.Get(t.Context(), client.ObjectKey{Name: gateway.ServiceName(dk.Name), Namespace: dk.Namespace}, svc))
		assert.True(t, svcMissing, "service should have been removed from the cluster")
	})

	t.Run("service has expected configuration", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
				KSPM:                 &kspm.Spec{},
			},
		}
		fakeClient := fake.NewClient(dk)

		err := gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.NoError(t, err)

		svc := &corev1.Service{}
		require.NoError(t, fakeClient.Get(t.Context(), client.ObjectKey{Name: gateway.ServiceName(dk.Name), Namespace: dk.Namespace}, svc))
		assert.Equal(t, corev1.ServiceTypeClusterIP, svc.Spec.Type)

		httpsPort := requirePort(t, svc, agconsts.HTTPSServicePortName)
		assert.Equal(t, int32(agconsts.HTTPSServicePort), httpsPort.Port)
		assert.Equal(t, corev1.ProtocolTCP, httpsPort.Protocol)
		assert.Equal(t, intstr.FromString(agconsts.HTTPSServicePortName), httpsPort.TargetPort)

		httpPort := requirePort(t, svc, agconsts.HTTPServicePortName)
		assert.Equal(t, int32(agconsts.HTTPServicePort), httpPort.Port)
		assert.Equal(t, corev1.ProtocolTCP, httpPort.Protocol)
		assert.Equal(t, intstr.FromString(agconsts.HTTPServicePortName), httpPort.TargetPort)

		assert.Equal(t, map[string]string{
			"app.kubernetes.io/name":                  "kubemon",
			"app.kubernetes.io/instance":              "test-dk",
			"app.kubernetes.io/managed-by":            "dynatrace-operator",
			"internal.dynatrace.com/operator-version": "snapshot",
		}, svc.Labels)
		assert.Equal(t, map[string]string{
			"app.kubernetes.io/name":       "kubemon",
			"app.kubernetes.io/instance":   "test-dk",
			"app.kubernetes.io/managed-by": "dynatrace-operator",
		}, svc.Spec.Selector)
	})

	t.Run("service creation failure is propagated", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
				KSPM:                 &kspm.Spec{},
			},
		}
		createErr := errors.New("kube api error")
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*corev1.Service); ok {
					return createErr
				}

				return c.Create(ctx, obj, opts...)
			},
		}, dk)

		err := gateway.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.ErrorIs(t, err, createErr)
	})
}

func requirePort(t *testing.T, svc *corev1.Service, name string) corev1.ServicePort {
	t.Helper()

	for _, p := range svc.Spec.Ports {
		if p.Name == name {
			return p
		}
	}

	t.Fatalf("port %q not found in service spec", name)

	return corev1.ServicePort{}
}
