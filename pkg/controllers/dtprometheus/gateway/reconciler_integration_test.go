// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package gateway_test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dtprometheus/gateway"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Integration tests for the gateway reconciler against a real API server. Drive one DTPrometheus through
// ordered, state-sharing phases and call only the public Reconcile method. Assertions stay high-level (existence,
// absence, resourceVersion change) — exact resource shape and defaulting/label-merge branch logic are covered by the
// unit test and its golden files.

const (
	integrationNamespace    = "dynatrace"
	integrationDTPName      = "lifecycle"
	integrationDynaKubeName = "dk"
	integrationImage        = "registry.example.com/gateway:1.2.3"
)

type lifecycleDeps struct {
	clt        client.Client
	reconciler *gateway.Reconciler
	dtp        *dtprometheus.DTPrometheus
	dk         *dynakube.DynaKube
}

// TestReconcileLifecycle walks the phases in order: missing image -> provision -> stabilize -> update.
func TestReconcileLifecycle(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	integrationtests.CreateNamespace(t, t.Context(), clt, integrationNamespace)

	dtp := &dtprometheus.DTPrometheus{
		ObjectMeta: metav1.ObjectMeta{Name: integrationDTPName, Namespace: integrationNamespace},
		Spec:       dtprometheus.DTPrometheusSpec{DynaKubeName: integrationDynaKubeName},
	}
	integrationtests.CreateKubernetesObject(t, t.Context(), clt, dtp)

	deps := &lifecycleDeps{
		clt:        clt,
		reconciler: &gateway.Reconciler{Client: clt},
		dtp:        dtp,
		dk:         &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: integrationDynaKubeName, Namespace: integrationNamespace}},
	}

	t.Run("missing-image", func(t *testing.T) { runMissingImagePhase(t, deps) })
	t.Run("provision", func(t *testing.T) { runProvisionPhase(t, deps) })
	t.Run("stabilize", func(t *testing.T) { runStabilizePhase(t, deps) })
	t.Run("update", func(t *testing.T) { runUpdatePhase(t, deps) })
}

// runMissingImagePhase reconciles before an image is configured. The ConfigMap has no image dependency and is created;
// the StatefulSet and Service are never attempted because the reconcile loop breaks on the first error.
func runMissingImagePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	require.Error(t, deps.reconciler.Reconcile(t.Context(), deps.dtp, deps.dk, nil))

	getConfigMap(t, deps)
	assertStatefulSetAbsent(t, deps)
	assertServiceAbsent(t, deps)
}

// runProvisionPhase sets an image and reconciles. All three resources must now exist, owned by the DTPrometheus.
func runProvisionPhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	deps.dtp.Spec.Gateway.Image = integrationImage
	require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.dtp, deps.dk, nil))

	cm := getConfigMap(t, deps)
	sts := getStatefulSet(t, deps)
	svc := getService(t, deps)

	assert.True(t, metav1.IsControlledBy(cm, deps.dtp))
	assert.True(t, metav1.IsControlledBy(sts, deps.dtp))
	assert.True(t, metav1.IsControlledBy(svc, deps.dtp))

	require.Len(t, sts.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, integrationImage, sts.Spec.Template.Spec.Containers[0].Image)
}

// runStabilizePhase reconciles repeatedly with unchanged input. None of the three resources may be rewritten.
// resourceVersion staying constant isn't enough to prove that on its own: the API server re-defaults an incoming object
// before comparing it to storage, so a reconcile that sends a stale, non-defaulted object still triggers a real Update
// call that gets silently no-op'ed server-side, leaving resourceVersion unchanged. The updateCallCounter catches that
// case by counting the actual client-side Update calls instead.
func runStabilizePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	cmRV := getConfigMap(t, deps).ResourceVersion
	stsRV := getStatefulSet(t, deps).ResourceVersion
	svcRV := getService(t, deps).ResourceVersion

	counting := &updateCallCounter{Client: deps.clt}
	reconciler := &gateway.Reconciler{Client: counting}

	for range 3 {
		require.NoError(t, reconciler.Reconcile(t.Context(), deps.dtp, deps.dk, nil))

		assert.Equal(t, cmRV, getConfigMap(t, deps).ResourceVersion)
		assert.Equal(t, stsRV, getStatefulSet(t, deps).ResourceVersion)
		assert.Equal(t, svcRV, getService(t, deps).ResourceVersion)
	}

	assert.Zero(t, counting.updateCalls)
}

// updateCallCounter wraps a client.Client to count Update calls issued through it.
type updateCallCounter struct {
	client.Client
	updateCalls int
}

func (c *updateCallCounter) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	c.updateCalls++

	return c.Client.Update(ctx, obj, opts...)
}

func runUpdatePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	// A ConfigMap content change carries into the StatefulSet through the pod template's config-hash annotation;
	// the Service has no such dependency.
	t.Run("configmap change ripples into the statefulset, service untouched", func(t *testing.T) {
		cmRV := getConfigMap(t, deps).ResourceVersion
		stsRV := getStatefulSet(t, deps).ResourceVersion
		svcRV := getService(t, deps).ResourceVersion

		deps.dk.Spec.APIURL = "https://changed.example.com/api"
		require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.dtp, deps.dk, nil))

		assert.NotEqual(t, cmRV, getConfigMap(t, deps).ResourceVersion)
		assert.NotEqual(t, stsRV, getStatefulSet(t, deps).ResourceVersion)
		assert.Equal(t, svcRV, getService(t, deps).ResourceVersion)
	})

	// A StatefulSet-only field must not touch the ConfigMap or the Service.
	t.Run("replicas change only touches the statefulset", func(t *testing.T) {
		cmRV := getConfigMap(t, deps).ResourceVersion
		stsRV := getStatefulSet(t, deps).ResourceVersion
		svcRV := getService(t, deps).ResourceVersion

		deps.dtp.Spec.Gateway.Replicas = new(int32(2))
		require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.dtp, deps.dk, nil))

		assert.Equal(t, cmRV, getConfigMap(t, deps).ResourceVersion)
		assert.NotEqual(t, stsRV, getStatefulSet(t, deps).ResourceVersion)
		assert.Equal(t, svcRV, getService(t, deps).ResourceVersion)
	})
}

func gatewayKey(dtp *dtprometheus.DTPrometheus) client.ObjectKey {
	return client.ObjectKey{Name: dtp.Gateway().GetStatefulSetName(), Namespace: dtp.Namespace}
}

func getConfigMap(t *testing.T, deps *lifecycleDeps) *corev1.ConfigMap {
	t.Helper()

	cm := &corev1.ConfigMap{}
	require.NoError(t, deps.clt.Get(t.Context(), gatewayKey(deps.dtp), cm))

	return cm
}

func getStatefulSet(t *testing.T, deps *lifecycleDeps) *appsv1.StatefulSet {
	t.Helper()

	sts := &appsv1.StatefulSet{}
	require.NoError(t, deps.clt.Get(t.Context(), gatewayKey(deps.dtp), sts))

	return sts
}

func getService(t *testing.T, deps *lifecycleDeps) *corev1.Service {
	t.Helper()

	svc := &corev1.Service{}
	require.NoError(t, deps.clt.Get(t.Context(), gatewayKey(deps.dtp), svc))

	return svc
}

func assertStatefulSetAbsent(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	err := deps.clt.Get(t.Context(), gatewayKey(deps.dtp), &appsv1.StatefulSet{})
	assert.True(t, k8serrors.IsNotFound(err))
}

func assertServiceAbsent(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	err := deps.clt.Get(t.Context(), gatewayKey(deps.dtp), &corev1.Service{})
	assert.True(t, k8serrors.IsNotFound(err))
}
