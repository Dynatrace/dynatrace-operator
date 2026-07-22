// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package statefulset_test

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	kubemonauthtoken "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Integration tests for the statefulset reconciler against a real API server. Drives one DynaKube
// through ordered, state-sharing phases; each phase asserts with a single reconcile call against a
// direct API client. Branch and error logic is covered by the unit test.

const (
	integrationNamespace      = "dynatrace"
	integrationDynaKubeName   = "lifecycle"
	integrationAPIURL         = "https://tenant.live.dynatrace.com/api"
	integrationImage          = "registry.example.com/linux/activegate:1.2.3"
	integrationKubeSystemUUID = "test-cluster-uuid"

	integrationTenantToken        = "test-tenant-token"
	integrationAuthToken          = "test-auth-token"
	integrationRotatedTenantToken = "rotated-tenant-token"
)

type lifecycleDeps struct {
	clt                    client.Client
	reconciler             *statefulset.Reconciler
	dk                     *dynakube.DynaKube
	tenantTokenSecret      *corev1.Secret
	authTokenSecret        *corev1.Secret
	initialTenantTokenHash string
	rotatedTenantTokenHash string
}

// TestReconcileLifecycle walks the phases in order: provision → rollout complete → rotate → disable → re-enable.
func TestReconcileLifecycle(t *testing.T) {
	clt := integrationtests.SetupTestEnvironment(t)

	integrationtests.CreateNamespace(t, t.Context(), clt, integrationNamespace)

	reconciler := statefulset.NewReconciler(clt)

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      integrationDynaKubeName,
			Namespace: integrationNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: integrationAPIURL,
			KubernetesMonitoring: &kubemonapi.Spec{
				StatefulSetProperties: kubemonapi.StatefulSetProperties{
					Image: integrationImage,
				},
			},
		},
		Status: dynakube.DynaKubeStatus{
			KubeSystemUUID: integrationKubeSystemUUID,
		},
	}
	integrationtests.CreateDynakube(t, t.Context(), clt, dk)

	tenantTokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.KubernetesMonitoring().GetTenantSecretName(),
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{
			connectioninfo.TenantTokenKey: []byte(integrationTenantToken),
		},
	}
	integrationtests.CreateKubernetesObject(t, t.Context(), clt, tenantTokenSecret)

	authTokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.KubernetesMonitoring().GetAuthTokenSecretName(),
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{
			kubemonauthtoken.SecretKey: []byte(integrationAuthToken),
		},
	}
	integrationtests.CreateKubernetesObject(t, t.Context(), clt, authTokenSecret)

	// The subtests below share dk and run in order: each builds on the state left by the previous one.
	deps := &lifecycleDeps{
		clt:               clt,
		reconciler:        reconciler,
		dk:                dk,
		tenantTokenSecret: tenantTokenSecret,
		authTokenSecret:   authTokenSecret,
	}

	t.Run("provision", func(t *testing.T) { runProvisionPhase(t, deps) })
	t.Run("rollout-complete", func(t *testing.T) { runRolloutCompletePhase(t, deps) })
	t.Run("rotate", func(t *testing.T) { runRotatePhase(t, deps) })
	t.Run("stabilize", func(t *testing.T) { runStabilizePhase(t, deps) })
	t.Run("disable", func(t *testing.T) { runDisablePhase(t, deps) })
	t.Run("re-enable", func(t *testing.T) { runReEnablePhase(t, deps) })
}

func runProvisionPhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	require.ErrorIs(t, deps.reconciler.Reconcile(t.Context(), deps.dk), k8sstatefulset.ErrRolloutInProgress)

	sts := getStatefulSet(t, deps.clt, deps.dk)

	assertStatefulSetShape(t, sts, deps.dk)

	deps.initialTenantTokenHash = sts.Spec.Template.Annotations[statefulset.AnnotationTenantTokenHash]
	require.NotEmpty(t, deps.initialTenantTokenHash)
}

func runRolloutCompletePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	markRolloutComplete(t, t.Context(), deps.clt, deps.dk)

	require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.dk))
}

func runRotatePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	deps.tenantTokenSecret.Data[connectioninfo.TenantTokenKey] = []byte(integrationRotatedTenantToken)
	require.NoError(t, deps.clt.Update(t.Context(), deps.tenantTokenSecret))

	require.ErrorIs(t, deps.reconciler.Reconcile(t.Context(), deps.dk), k8sstatefulset.ErrRolloutInProgress)

	sts := getStatefulSet(t, deps.clt, deps.dk)
	deps.rotatedTenantTokenHash = sts.Spec.Template.Annotations[statefulset.AnnotationTenantTokenHash]
	assert.NotEqual(t, deps.initialTenantTokenHash, deps.rotatedTenantTokenHash)
}

func runStabilizePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	stsRV := getStatefulSet(t, deps.clt, deps.dk).ResourceVersion

	// Repeated reconciles with identical input must not rewrite the StatefulSet.
	for range 3 {
		require.ErrorIs(t, deps.reconciler.Reconcile(t.Context(), deps.dk), k8sstatefulset.ErrRolloutInProgress)
		assert.Equal(t, stsRV, getStatefulSet(t, deps.clt, deps.dk).ResourceVersion)
	}
}

func runDisablePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	name := deps.dk.KubernetesMonitoring().GetStatefulSetName()
	deps.dk.Spec.KubernetesMonitoring = nil

	require.NoError(t, deps.reconciler.Reconcile(t.Context(), deps.dk))

	err := deps.clt.Get(t.Context(), client.ObjectKey{Name: name, Namespace: integrationNamespace}, &appsv1.StatefulSet{})
	require.True(t, k8serrors.IsNotFound(err))
}

func runReEnablePhase(t *testing.T, deps *lifecycleDeps) {
	t.Helper()

	deps.dk.Spec.KubernetesMonitoring = &kubemonapi.Spec{}
	deps.dk.Spec.KubernetesMonitoring.Image = integrationImage

	require.ErrorIs(t, deps.reconciler.Reconcile(t.Context(), deps.dk), k8sstatefulset.ErrRolloutInProgress)

	sts := getStatefulSet(t, deps.clt, deps.dk)
	assertStatefulSetShape(t, sts, deps.dk)
	assert.Equal(t, deps.rotatedTenantTokenHash, sts.Spec.Template.Annotations[statefulset.AnnotationTenantTokenHash])
}

func assertStatefulSetShape(t *testing.T, sts *appsv1.StatefulSet, dk *dynakube.DynaKube) {
	t.Helper()

	assert.Equal(t, dk.KubernetesMonitoring().GetStatefulSetName(), sts.Name)
	assert.True(t, metav1.IsControlledBy(sts, dk))

	require.Len(t, sts.Spec.Template.Spec.Containers, 1)
	container := sts.Spec.Template.Spec.Containers[0]
	assert.Equal(t, statefulset.ContainerName, container.Name)
	assert.Equal(t, dk.KubernetesMonitoring().GetCustomImage(), container.Image)

	require.GreaterOrEqual(t, len(container.Env), 6)
	assert.Equal(t, connectioninfo.EnvDTTenant, container.Env[4].Name)
	assert.Equal(t, connectioninfo.EnvDTServer, container.Env[5].Name)

	require.Len(t, container.VolumeMounts, 3)
	assert.Equal(t, connectioninfo.TenantSecretVolumeName, container.VolumeMounts[0].Name)
	assert.Equal(t, statefulset.AuthTokenVolumeName, container.VolumeMounts[1].Name)
	assert.Equal(t, kubemonauthtoken.SecretKey, container.VolumeMounts[1].SubPath)
	assert.Equal(t, statefulset.StorageVolumeName, container.VolumeMounts[2].Name)
	assert.Equal(t, dk.KubernetesMonitoring().GetServiceAccountName(), sts.Spec.Template.Spec.ServiceAccountName)

	require.Len(t, sts.Spec.Template.Spec.Volumes, 3)
	assert.Equal(t, connectioninfo.TenantSecretVolumeName, sts.Spec.Template.Spec.Volumes[0].Name)
	assert.Equal(t, statefulset.AuthTokenVolumeName, sts.Spec.Template.Spec.Volumes[1].Name)
	assert.Equal(t, statefulset.StorageVolumeName, sts.Spec.Template.Spec.Volumes[2].Name)
}

func markRolloutComplete(t *testing.T, ctx context.Context, clt client.Client, dk *dynakube.DynaKube) {
	t.Helper()

	sts := getStatefulSet(t, clt, dk)

	var desired int32 = 1
	if sts.Spec.Replicas != nil {
		desired = *sts.Spec.Replicas
	}

	sts.Status.ObservedGeneration = sts.Generation
	sts.Status.Replicas = desired
	sts.Status.ReadyReplicas = desired
	require.NoError(t, clt.Status().Update(ctx, sts))
}

func statefulSetKey(dk *dynakube.DynaKube) client.ObjectKey {
	return client.ObjectKey{Name: dk.KubernetesMonitoring().GetStatefulSetName(), Namespace: dk.Namespace}
}

func getStatefulSet(t *testing.T, reader client.Reader, dk *dynakube.DynaKube) *appsv1.StatefulSet {
	t.Helper()

	sts := &appsv1.StatefulSet{}
	require.NoError(t, reader.Get(t.Context(), statefulSetKey(dk), sts))

	return sts
}
