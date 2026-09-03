// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package scraper_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dtprometheus/scraper"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Integration tests for the scraper reconciler against a real API server. Drive one DTPrometheus through
// ordered, state-sharing phases and call only the public Reconcile method. Assertions stay high-level (existence,
// absence, resourceVersion change) — exact resource shape and defaulting/label-merge branch logic are covered by the
// unit test and its golden files.

const (
	integrationNamespace    = "dynatrace"
	integrationDTPName      = "lifecycle"
	integrationDynaKubeName = "dk"
	integrationImage        = "registry.example.com/scraper:1.2.3"
)

type lifecycleDeps struct {
	clt        client.Client
	reconciler *scraper.Reconciler
	dtp        *dtprometheus.DTPrometheus
	dk         *dynakube.DynaKube
}

// TestReconcileLifecycle walks the phases in order: missing image -> provision -> stabilize -> update.
func TestReconcileLifecycle(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)
	integrationtests.CreateNamespace(t, clt, integrationNamespace)

	dtp := &dtprometheus.DTPrometheus{
		ObjectMeta: metav1.ObjectMeta{Name: integrationDTPName, Namespace: integrationNamespace},
		Spec:       dtprometheus.DTPrometheusSpec{DynaKubeName: integrationDynaKubeName},
	}
	integrationtests.CreateKubernetesObject(t, clt, dtp)

	deps := &lifecycleDeps{
		clt:        clt,
		reconciler: &scraper.Reconciler{Client: clt},
		dtp:        dtp,
		dk:         &dynakube.DynaKube{},
	}

	t.Run("missing-image", func(t *testing.T) { runMissingImagePhase(t, deps) })
	t.Run("provision", func(t *testing.T) { runProvisionPhase(t, deps) })
	t.Run("stabilize", func(t *testing.T) { runStabilizePhase(t, deps) })
	t.Run("update", func(t *testing.T) { runUpdatePhase(t, deps) })
}

// runMissingImagePhase reconciles before an image is configured. The ConfigMap has no image dependency and is created;
// the Deployment is never attempted because the reconcile loop breaks on the first error.
func runMissingImagePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	imgClient := imagemock.NewClient(t)
	imgClient.EXPECT().GetComponentLatestInfo(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("no scraper image available"))
	require.Error(t, deps.reconciler.Reconcile(t.Context(), deps.dtp, deps.dk, imgClient))

	getConfigMap(t, deps)
	assertDeploymentAbsent(t, deps)
}

// runProvisionPhase sets an image and reconciles. Both resources must now exist, owned by the DTPrometheus.
// The scraper intentionally has no Service: nothing connects to it.
func runProvisionPhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	imgClient := imagemock.NewClient(t)
	imgClient.EXPECT().GetComponentLatestInfo(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("no scraper image available")).Maybe()
	deps.dtp.Spec.Scraper.Image = integrationImage
	require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.dtp, deps.dk, imgClient))

	cm := getConfigMap(t, deps)
	deploy := getDeployment(t, deps)

	assert.True(t, metav1.IsControlledBy(cm, deps.dtp))
	assert.True(t, metav1.IsControlledBy(deploy, deps.dtp))

	require.Len(t, deploy.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, integrationImage, deploy.Spec.Template.Spec.Containers[0].Image)

	assertServiceAbsent(t, deps)
}

// runStabilizePhase reconciles repeatedly with unchanged input. Neither resource may be rewritten.
// resourceVersion staying constant isn't enough to prove that on its own: the API server re-defaults an incoming object
// before comparing it to storage, so a reconcile that sends a stale, non-defaulted object still triggers a real Update
// call that gets silently no-op'ed server-side, leaving resourceVersion unchanged. The updateCallCounter catches that
// case by counting the actual client-side Update calls instead.
func runStabilizePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	cmRV := getConfigMap(t, deps).ResourceVersion
	deployRV := getDeployment(t, deps).ResourceVersion

	counting := &updateCallCounter{Client: deps.clt}
	reconciler := &scraper.Reconciler{Client: counting}

	imgClient := imagemock.NewClient(t)
	imgClient.EXPECT().GetComponentLatestInfo(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("no scraper image available")).Maybe()

	for range 3 {
		require.NoError(t, reconciler.Reconcile(t.Context(), deps.dtp, deps.dk, imgClient))

		assert.Equal(t, cmRV, getConfigMap(t, deps).ResourceVersion)
		assert.Equal(t, deployRV, getDeployment(t, deps).ResourceVersion)
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

	// A ConfigMap content change carries into the Deployment through the pod template's config hash annotation.
	t.Run("configmap change ripples into the deployment", func(t *testing.T) {
		cmRV := getConfigMap(t, deps).ResourceVersion
		deployRV := getDeployment(t, deps).ResourceVersion

		imgClient := imagemock.NewClient(t)
		imgClient.EXPECT().GetComponentLatestInfo(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("no scraper image available")).Maybe()
		deps.dtp.Spec.Scraper.TargetsPollInterval = metav1.Duration{Duration: 5 * time.Minute}
		require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.dtp, deps.dk, imgClient))

		assert.NotEqual(t, cmRV, getConfigMap(t, deps).ResourceVersion)
		assert.NotEqual(t, deployRV, getDeployment(t, deps).ResourceVersion)
	})

	// A Deployment-only field must not touch the ConfigMap.
	t.Run("replicas change only touches the deployment", func(t *testing.T) {
		cmRV := getConfigMap(t, deps).ResourceVersion
		deployRV := getDeployment(t, deps).ResourceVersion

		imgClient := imagemock.NewClient(t)
		imgClient.EXPECT().GetComponentLatestInfo(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("no scraper image available")).Maybe()
		deps.dtp.Spec.Scraper.Replicas = new(int32(2))
		require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.dtp, deps.dk, imgClient))

		assert.Equal(t, cmRV, getConfigMap(t, deps).ResourceVersion)
		assert.NotEqual(t, deployRV, getDeployment(t, deps).ResourceVersion)
	})
}

func scraperKey(dtp *dtprometheus.DTPrometheus) client.ObjectKey {
	return client.ObjectKey{Name: dtp.Scraper().GetDeploymentName(), Namespace: dtp.Namespace}
}

func getConfigMap(t *testing.T, deps *lifecycleDeps) *corev1.ConfigMap {
	t.Helper()

	cm := &corev1.ConfigMap{}
	require.NoError(t, deps.clt.Get(t.Context(), scraperKey(deps.dtp), cm))

	return cm
}

func getDeployment(t *testing.T, deps *lifecycleDeps) *appsv1.Deployment {
	t.Helper()

	deploy := &appsv1.Deployment{}
	require.NoError(t, deps.clt.Get(t.Context(), scraperKey(deps.dtp), deploy))

	return deploy
}

func assertDeploymentAbsent(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	err := deps.clt.Get(t.Context(), scraperKey(deps.dtp), &appsv1.Deployment{})
	assert.True(t, k8serrors.IsNotFound(err))
}

func assertServiceAbsent(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	err := deps.clt.Get(t.Context(), scraperKey(deps.dtp), &corev1.Service{})
	assert.True(t, k8serrors.IsNotFound(err))
}
