// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package targetallocator_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/prometheusmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/prometheusmonitoring/targetallocator"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Integration tests for the target allocator reconciler against a real API server. Drive one PrometheusMonitoring through
// ordered, state-sharing phases and call only the public Reconcile method. Assertions stay high-level (existence,
// absence, resourceVersion change) — exact resource shape and defaulting/label-merge branch logic are covered by the
// unit test and its golden files.

const (
	integrationNamespace   = "dynatrace"
	integrationPMName      = "lifecycle"
	integrationDynaKubeRef = "dk"
	integrationImage       = "registry.example.com/target-allocator:1.2.3"
)

type lifecycleDeps struct {
	clt        client.Client
	reconciler *targetallocator.Reconciler
	pm         *prometheusmonitoring.PrometheusMonitoring
	dk         *dynakube.DynaKube
}

// TestReconcileLifecycle walks the phases in order: missing image -> provision -> stabilize -> update.
func TestReconcileLifecycle(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	integrationtests.CreateNamespace(t, clt, integrationNamespace)

	pm := &prometheusmonitoring.PrometheusMonitoring{
		ObjectMeta: metav1.ObjectMeta{Name: integrationPMName, Namespace: integrationNamespace},
		Spec:       prometheusmonitoring.PrometheusMonitoringSpec{DynaKubeRef: integrationDynaKubeRef},
	}
	integrationtests.CreateKubernetesObject(t, clt, pm)

	deps := &lifecycleDeps{
		clt:        clt,
		reconciler: &targetallocator.Reconciler{Client: clt},
		pm:         pm,
		dk:         &dynakube.DynaKube{},
	}

	t.Run("missing-image", func(t *testing.T) { runMissingImagePhase(t, deps) })
	t.Run("provision", func(t *testing.T) { runProvisionPhase(t, deps) })
	t.Run("stabilize", func(t *testing.T) { runStabilizePhase(t, deps) })
	t.Run("update", func(t *testing.T) { runUpdatePhase(t, deps) })
}

// runMissingImagePhase reconciles when no image is configured, simulating a fleet API failure. The ConfigMap has no
// image dependency and is created; the Deployment and Service are never attempted because the reconcile loop breaks on
// the first error.
func runMissingImagePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	imageClient := imagemock.NewClient(t)
	imageClient.EXPECT().GetComponentLatestInfo(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("fleet API unavailable"))

	require.Error(t, deps.reconciler.Reconcile(t.Context(), deps.pm, deps.dk, imageClient))

	getConfigMap(t, deps)
	assertDeploymentAbsent(t, deps)
	assertServiceAbsent(t, deps)
}

// runProvisionPhase sets an image and reconciles. All three resources must now exist, owned by the PrometheusMonitoring.
func runProvisionPhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	deps.pm.Spec.TargetAllocator.Image = integrationImage
	// Only rollingUpdate is set, no type: the apiserver defaults the type to RollingUpdate on its own.
	maxUnavailable := intstr.FromInt(0)
	deps.pm.Spec.TargetAllocator.UpdateStrategy = appsv1.DeploymentStrategy{
		RollingUpdate: &appsv1.RollingUpdateDeployment{MaxUnavailable: &maxUnavailable},
	}
	require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.pm, deps.dk, nil))

	cm := getConfigMap(t, deps)
	deploy := getDeployment(t, deps)
	svc := getService(t, deps)

	assert.True(t, metav1.IsControlledBy(cm, deps.pm))
	assert.True(t, metav1.IsControlledBy(deploy, deps.pm))
	assert.True(t, metav1.IsControlledBy(svc, deps.pm))

	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, integrationImage, deploy.Spec.Template.Spec.Containers[0].Image)

	assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, deploy.Spec.Strategy.Type)
	require.NotNil(t, deploy.Spec.Strategy.RollingUpdate)
	assert.Equal(t, &maxUnavailable, deploy.Spec.Strategy.RollingUpdate.MaxUnavailable)
}

// runStabilizePhase reconciles repeatedly with unchanged input. None of the three resources may be rewritten.
// resourceVersion staying constant isn't enough to prove that on its own: the API server re-defaults an incoming object
// before comparing it to storage, so a reconcile that sends a stale, non-defaulted object still triggers a real Update
// call that gets silently no-op'ed server-side, leaving resourceVersion unchanged. The updateCallCounter catches that
// case by counting the actual client-side Update calls instead.
func runStabilizePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	cmRV := getConfigMap(t, deps).ResourceVersion
	deployRV := getDeployment(t, deps).ResourceVersion
	svcRV := getService(t, deps).ResourceVersion

	counting := &updateCallCounter{Client: deps.clt}
	reconciler := &targetallocator.Reconciler{Client: counting}

	for range 3 {
		require.NoError(t, reconciler.Reconcile(t.Context(), deps.pm, deps.dk, nil))

		assert.Equal(t, cmRV, getConfigMap(t, deps).ResourceVersion)
		assert.Equal(t, deployRV, getDeployment(t, deps).ResourceVersion)
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

	// A ConfigMap content change carries into the Deployment through the pod template's config/checksum annotation;
	// the Service has no such dependency.
	t.Run("configmap change ripples into the deployment, service untouched", func(t *testing.T) {
		cmRV := getConfigMap(t, deps).ResourceVersion
		deployRV := getDeployment(t, deps).ResourceVersion
		svcRV := getService(t, deps).ResourceVersion

		deps.pm.Spec.TargetAllocator.ScrapeInterval = metav1.Duration{Duration: 5 * time.Minute}
		require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.pm, deps.dk, nil))

		assert.NotEqual(t, cmRV, getConfigMap(t, deps).ResourceVersion)
		assert.NotEqual(t, deployRV, getDeployment(t, deps).ResourceVersion)
		assert.Equal(t, svcRV, getService(t, deps).ResourceVersion)
	})

	// A Deployment-only field must not touch the ConfigMap or the Service.
	t.Run("replicas change only touches the deployment", func(t *testing.T) {
		cmRV := getConfigMap(t, deps).ResourceVersion
		deployRV := getDeployment(t, deps).ResourceVersion
		svcRV := getService(t, deps).ResourceVersion

		deps.pm.Spec.TargetAllocator.Replicas = new(int32(2))
		require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.pm, deps.dk, nil))

		assert.Equal(t, cmRV, getConfigMap(t, deps).ResourceVersion)
		assert.NotEqual(t, deployRV, getDeployment(t, deps).ResourceVersion)
		assert.Equal(t, svcRV, getService(t, deps).ResourceVersion)
	})

	// Switching to a non-rolling strategy must clear the stale rollingUpdate block and leave the
	// ConfigMap and Service alone.
	t.Run("switching to Recreate clears rollingUpdate, service and configmap untouched", func(t *testing.T) {
		cmRV := getConfigMap(t, deps).ResourceVersion
		svcRV := getService(t, deps).ResourceVersion

		deps.pm.Spec.TargetAllocator.UpdateStrategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
		require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.dtp, deps.dk, nil))

		deploy := getDeployment(t, deps)
		assert.Equal(t, appsv1.RecreateDeploymentStrategyType, deploy.Spec.Strategy.Type)
		assert.Nil(t, deploy.Spec.Strategy.RollingUpdate)

		assert.Equal(t, cmRV, getConfigMap(t, deps).ResourceVersion)
		assert.Equal(t, svcRV, getService(t, deps).ResourceVersion)
	})
}

func targetAllocatorKey(pm *prometheusmonitoring.PrometheusMonitoring) client.ObjectKey {
	return client.ObjectKey{Name: pm.TargetAllocator().GetDeploymentName(), Namespace: pm.Namespace}
}

func getConfigMap(t *testing.T, deps *lifecycleDeps) *corev1.ConfigMap {
	t.Helper()

	cm := &corev1.ConfigMap{}
	require.NoError(t, deps.clt.Get(t.Context(), targetAllocatorKey(deps.pm), cm))

	return cm
}

func getDeployment(t *testing.T, deps *lifecycleDeps) *appsv1.Deployment {
	t.Helper()

	deploy := &appsv1.Deployment{}
	require.NoError(t, deps.clt.Get(t.Context(), targetAllocatorKey(deps.pm), deploy))

	return deploy
}

func getService(t *testing.T, deps *lifecycleDeps) *corev1.Service {
	t.Helper()

	svc := &corev1.Service{}
	require.NoError(t, deps.clt.Get(t.Context(), targetAllocatorKey(deps.pm), svc))

	return svc
}

func assertDeploymentAbsent(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	err := deps.clt.Get(t.Context(), targetAllocatorKey(deps.pm), &appsv1.Deployment{})
	assert.True(t, k8serrors.IsNotFound(err))
}

func assertServiceAbsent(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	err := deps.clt.Get(t.Context(), targetAllocatorKey(deps.pm), &corev1.Service{})
	assert.True(t, k8serrors.IsNotFound(err))
}
