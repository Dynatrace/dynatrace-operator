package statefulset_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// Unit tests for the statefulset reconciler. Use a fake client with interceptors to inject
// write/read failures; they own all branch and error logic. The full happy path (completed rollout)
// and the token-rotation lifecycle are covered by the integration test.

const testNamespace = "dynatrace"

// TestReconcilePreconditionErrors verifies early pre-write failures for missing required
// prerequisites (image and tenant-token secret).
func TestReconcilePreconditionErrors(t *testing.T) {
	tests := map[string]struct {
		mutate      func(*dynakube.DynaKube)
		assertError func(*testing.T, error)
	}{
		"missing image": {
			mutate: func(dk *dynakube.DynaKube) { dk.Spec.KubernetesMonitoring.Image = "" },
			assertError: func(t *testing.T, err error) {
				require.ErrorIs(t, err, statefulset.ErrImageRequired)
			},
		},
		"missing tenant secret": {
			// secret not seeded — k8s Get error, not ErrImageRequired
			assertError: func(t *testing.T, err error) {
				require.True(t, k8serrors.IsNotFound(err))
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dk := newTestDynaKube(true)
			if test.mutate != nil {
				test.mutate(dk)
			}

			err := statefulset.NewReconciler(fake.NewClient(dk)).Reconcile(t.Context(), dk)
			require.Error(t, err)
			require.NotErrorIs(t, err, k8sstatefulset.ErrRolloutInProgress)
			test.assertError(t, err)
		})
	}
}

// TestReconcileResolveReplicasReadFailure verifies that a non-NotFound StatefulSet read error
// from ResolveReplicas exits reconcile before any StatefulSet write.
func TestReconcileResolveReplicasReadFailure(t *testing.T) {
	dk := newTestDynaKube(true)
	writeAttempted := false
	fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if isStatefulSet(obj) {
				return errors.New("kube api error")
			}

			return c.Get(ctx, key, obj, opts...)
		},
		Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			if isStatefulSet(obj) {
				writeAttempted = true
			}

			return c.Create(ctx, obj, opts...)
		},
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			if isStatefulSet(obj) {
				writeAttempted = true
			}

			return c.Update(ctx, obj, opts...)
		},
	}, dk, newTestTenantSecret(dk))

	err := statefulset.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
	require.Error(t, err)
	require.NotErrorIs(t, err, k8sstatefulset.ErrRolloutInProgress)
	require.NotErrorIs(t, err, statefulset.ErrImageRequired)
	assert.False(t, writeAttempted)
}

// TestReconcileBuildsStatefulSet covers the shape of the produced StatefulSet — container, envs,
// volume mounts, and the tenant-token restart annotation. The fake client has no StatefulSet
// controller, so the reconcile always reports rollout in progress.
func TestReconcileBuildsStatefulSet(t *testing.T) {
	dk := newTestDynaKube(true)
	dk.Spec.KubernetesMonitoring.Env = []corev1.EnvVar{{Name: "CUSTOM", Value: "value"}}
	fakeClient := fake.NewClient(dk, newTestTenantSecret(dk))

	err := statefulset.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
	require.ErrorIs(t, err, k8sstatefulset.ErrRolloutInProgress)

	sts := requireTestStatefulSet(t, t.Context(), fakeClient, dk)

	require.Len(t, sts.Spec.Template.Spec.Containers, 1)
	container := sts.Spec.Template.Spec.Containers[0]
	assert.Equal(t, statefulset.ContainerName, container.Name)
	assert.Equal(t, dk.KubernetesMonitoring().GetCustomImage(), container.Image)
	assert.Equal(t, dk.KubernetesMonitoring().GetServiceAccountName(), sts.Spec.Template.Spec.ServiceAccountName)

	// buildEnvs prepends the three runtime vars, then appends user-supplied vars. No DT_GROUP when group is unset.
	require.Len(t, container.Env, 4)
	assert.Equal(t, agconsts.EnvDTCapabilities, container.Env[0].Name)
	assert.Equal(t, activegate.KubeMonCapability.ArgumentName, container.Env[0].Value)
	assert.Equal(t, connectioninfo.EnvDTTenant, container.Env[1].Name)
	assert.Equal(t, connectioninfo.EnvDTServer, container.Env[2].Name)
	assert.Equal(t, "CUSTOM", container.Env[3].Name)

	// buildVolumeMounts + buildVolumes wire the tenant token secret into the container. No custom-props mount when unset.
	require.Len(t, container.VolumeMounts, 1)
	assert.Equal(t, connectioninfo.TenantSecretVolumeName, container.VolumeMounts[0].Name)
	assert.Equal(t, connectioninfo.TenantTokenMountPoint, container.VolumeMounts[0].MountPath)
	assert.Equal(t, connectioninfo.TenantTokenKey, container.VolumeMounts[0].SubPath)
	assert.True(t, container.VolumeMounts[0].ReadOnly)
	assert.True(t, hasTenantSecretVolume(sts, dk))

	// getTenantTokenHash records the restart-trigger annotation derived from the secret.
	assert.NotEmpty(t, sts.Spec.Template.Annotations[statefulset.AnnotationTenantTokenHash])
}

// TestReconcileWriteFailures covers the two write/read error paths after the StatefulSet is built:
// the create itself failing, and the follow-up Get used to evaluate rollout completion.
func TestReconcileWriteFailures(t *testing.T) {
	t.Run("returns error when statefulset create fails", func(t *testing.T) {
		dk := newTestDynaKube(true)
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if isStatefulSet(obj) {
					return errors.New("kube api error")
				}

				return c.Create(ctx, obj, opts...)
			},
		}, dk, newTestTenantSecret(dk))

		err := statefulset.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.Error(t, err)
		require.NotErrorIs(t, err, k8sstatefulset.ErrRolloutInProgress)
	})

	t.Run("returns error when re-getting the statefulset fails", func(t *testing.T) {
		dk := newTestDynaKube(true)
		created := false
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if isStatefulSet(obj) {
					created = true
				}

				return c.Create(ctx, obj, opts...)
			},
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if created && isStatefulSet(obj) {
					return errors.New("kube api error")
				}

				return c.Get(ctx, key, obj, opts...)
			},
		}, dk, newTestTenantSecret(dk))

		err := statefulset.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.Error(t, err)
		require.NotErrorIs(t, err, k8sstatefulset.ErrRolloutInProgress)
	})
}

// TestReconcileCleanupDeleteFailure covers the delete failure on the cleanup path.
func TestReconcileCleanupDeleteFailure(t *testing.T) {
	dk := newTestDynaKube(false)
	existing := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{
		Name:      dk.KubernetesMonitoring().GetStatefulSetName(),
		Namespace: dk.Namespace,
	}}
	fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
		Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			if isStatefulSet(obj) {
				return errors.New("kube api error")
			}

			return c.Delete(ctx, obj, opts...)
		},
	}, dk, existing)

	require.Error(t, statefulset.NewReconciler(fakeClient).Reconcile(t.Context(), dk))
}

func newTestDynaKube(enabled bool) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://tenant.live.dynatrace.com/api",
		},
	}

	if enabled {
		dk.Spec.KubernetesMonitoring = &kubemonapi.Spec{}
		dk.Spec.KubernetesMonitoring.Image = "registry.example.com/linux/activegate:1.2.3"
	}

	return dk
}

func newTestTenantSecret(dk *dynakube.DynaKube) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.KubernetesMonitoring().GetTenantSecretName(),
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{
			connectioninfo.TenantTokenKey: []byte("test-tenant-token"),
		},
	}
}

func requireTestStatefulSet(t *testing.T, ctx context.Context, clt client.Client, dk *dynakube.DynaKube) *appsv1.StatefulSet {
	t.Helper()

	sts := &appsv1.StatefulSet{}
	require.NoError(t, clt.Get(ctx, types.NamespacedName{Name: dk.KubernetesMonitoring().GetStatefulSetName(), Namespace: dk.Namespace}, sts))

	return sts
}

func isStatefulSet(obj client.Object) bool {
	_, ok := obj.(*appsv1.StatefulSet)

	return ok
}

func hasTenantSecretVolume(sts *appsv1.StatefulSet, dk *dynakube.DynaKube) bool {
	return slices.ContainsFunc(sts.Spec.Template.Spec.Volumes, func(v corev1.Volume) bool {
		return v.Name == connectioninfo.TenantSecretVolumeName &&
			v.Secret != nil &&
			v.Secret.SecretName == dk.KubernetesMonitoring().GetTenantSecretName()
	})
}
