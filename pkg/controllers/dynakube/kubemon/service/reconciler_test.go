// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package service_test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/service"
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
		err := service.NewReconciler(fake.NewClient(dk)).Reconcile(t.Context(), dk)
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
				Name:      service.BuildServiceName(dk.Name),
				Namespace: dk.Namespace,
			},
		}
		fakeClient := fake.NewClient(dk, existingSvc)

		err := service.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.NoError(t, err)

		svc := &corev1.Service{}
		getErr := fakeClient.Get(t.Context(), client.ObjectKey{Name: service.BuildServiceName(dk.Name), Namespace: dk.Namespace}, svc)
		assert.True(t, k8serrors.IsNotFound(getErr), "service should have been removed from the cluster")
	})
}

func TestKubemonEnabled(t *testing.T) {
	t.Run("ClusterIP service is created with HTTPS and HTTP ports", func(t *testing.T) {
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

		err := service.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.NoError(t, err)

		svc := &corev1.Service{}
		require.NoError(t, fakeClient.Get(t.Context(), client.ObjectKey{Name: service.BuildServiceName(dk.Name), Namespace: dk.Namespace}, svc))
		assert.Equal(t, corev1.ServiceTypeClusterIP, svc.Spec.Type)

		httpsPort := requirePort(t, svc, agconsts.HTTPSServicePortName)
		assert.Equal(t, int32(agconsts.HTTPSServicePort), httpsPort.Port)
		assert.Equal(t, corev1.ProtocolTCP, httpsPort.Protocol)
		assert.Equal(t, intstr.FromString(agconsts.HTTPSServicePortName), httpsPort.TargetPort)

		httpPort := requirePort(t, svc, agconsts.HTTPServicePortName)
		assert.Equal(t, int32(agconsts.HTTPServicePort), httpPort.Port)
		assert.Equal(t, corev1.ProtocolTCP, httpPort.Protocol)
		assert.Equal(t, intstr.FromString(agconsts.HTTPServicePortName), httpPort.TargetPort)
	})

	t.Run("ClusterIPs assigned by Kubernetes are reflected in DynaKube status", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
			},
		}
		wantIPs := []string{"10.0.0.1", "fd00::1"}
		fakeClient := fake.NewClient(dk)
		reconciler := service.NewReconciler(fakeClient)

		require.NoError(t, reconciler.Reconcile(t.Context(), dk))

		svc := &corev1.Service{}
		require.NoError(t, fakeClient.Get(t.Context(), client.ObjectKey{Name: service.BuildServiceName(dk.Name), Namespace: dk.Namespace}, svc))
		svc.Spec.ClusterIPs = wantIPs
		require.NoError(t, fakeClient.Update(t.Context(), svc))

		require.NoError(t, reconciler.Reconcile(t.Context(), dk))
		assert.Equal(t, wantIPs, dk.Status.KubernetesMonitoring.ServiceIPs)
	})

	t.Run("service creation failure is propagated", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
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

		err := service.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.ErrorIs(t, err, createErr)
	})

	t.Run("cluster API error for Service objects is propagated", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dk",
				Namespace: "dynatrace",
			},
			Spec: dynakube.DynaKubeSpec{
				KubernetesMonitoring: &kubemonapi.Spec{},
			},
		}
		getErr := errors.New("kube api error")
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*corev1.Service); ok {
					return getErr
				}

				return c.Get(ctx, key, obj, opts...)
			},
		}, dk)

		err := service.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.ErrorIs(t, err, getErr)
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
