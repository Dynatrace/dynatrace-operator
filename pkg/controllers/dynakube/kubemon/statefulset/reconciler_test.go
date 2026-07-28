// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

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
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	kubemonauthtoken "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			// Only auth token secret seeded — tenant secret Get returns NotFound first.
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

			err := statefulset.NewReconciler(fake.NewClient(dk, newTestAuthTokenSecret(dk))).Reconcile(t.Context(), dk)
			require.Error(t, err)
			require.NotErrorIs(t, err, k8sstatefulset.ErrRolloutInProgress)
			test.assertError(t, err)
		})
	}
}

func TestReconcileMissingKubeSystemUID(t *testing.T) {
	dk := newTestDynaKube(true)
	dk.Status.KubeSystemUUID = ""
	err := statefulset.NewReconciler(fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))).Reconcile(t.Context(), dk)
	require.ErrorIs(t, err, statefulset.ErrMissingKubeSystemUID)
}

func TestReconcileMissingTokenValue(t *testing.T) {
	tests := []struct {
		name        string
		tenantData  map[string][]byte
		authData    map[string][]byte
		expectedErr error
	}{
		{
			"tenant token key missing",
			map[string][]byte{},
			map[string][]byte{kubemonauthtoken.SecretKey: []byte("test-auth-token")},
			statefulset.ErrMissingTenantToken,
		},
		{
			"tenant token value empty",
			map[string][]byte{connectioninfo.TenantTokenKey: {}},
			map[string][]byte{kubemonauthtoken.SecretKey: []byte("test-auth-token")},
			statefulset.ErrMissingTenantToken,
		},
		{
			"auth token key missing",
			map[string][]byte{connectioninfo.TenantTokenKey: []byte("test-tenant-token")},
			map[string][]byte{},
			statefulset.ErrMissingAuthToken,
		},
		{
			"auth token value empty",
			map[string][]byte{connectioninfo.TenantTokenKey: []byte("test-tenant-token")},
			map[string][]byte{kubemonauthtoken.SecretKey: {}},
			statefulset.ErrMissingAuthToken,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dk := newTestDynaKube(true)
			tenantSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dk.KubernetesMonitoring().GetTenantSecretName(),
					Namespace: dk.Namespace,
				},
				Data: test.tenantData,
			}
			authSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dk.KubernetesMonitoring().GetAuthTokenSecretName(),
					Namespace: dk.Namespace,
				},
				Data: test.authData,
			}

			err := statefulset.NewReconciler(fake.NewClient(dk, tenantSecret, authSecret)).Reconcile(t.Context(), dk)
			require.ErrorIs(t, err, test.expectedErr)
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

// TestReconcileBuildsStatefulSet covers the shape of the produced StatefulSet. The fake client has
// no StatefulSet controller, so reconcile always reports rollout in progress.
func TestReconcileBuildsStatefulSet(t *testing.T) {
	t.Run("container identity and service account", func(t *testing.T) {
		dk := newTestDynaKube(true)
		sts := reconcileAndGetSTS(t, dk)

		require.Len(t, sts.Spec.Template.Spec.Containers, 1)
		container := sts.Spec.Template.Spec.Containers[0]

		assert.Equal(t, statefulset.ContainerName, container.Name)
		assert.Equal(t, dk.KubernetesMonitoring().GetCustomImage(), container.Image)
		assert.Equal(t, dk.KubernetesMonitoring().GetServiceAccountName(), sts.Spec.Template.Spec.ServiceAccountName)
	})

	t.Run("env vars: capabilities, seed envs, deployment metadata, connection info, and custom", func(t *testing.T) {
		dk := newTestDynaKube(true)
		dk.Spec.KubernetesMonitoring.Env = []corev1.EnvVar{{Name: "CUSTOM", Value: "value"}}
		sts := reconcileAndGetSTS(t, dk)

		require.Len(t, sts.Spec.Template.Spec.Containers, 1)
		container := sts.Spec.Template.Spec.Containers[0]

		require.Len(t, container.Env, 7)

		capabilitiesEnv := k8senv.Find(container.Env, agconsts.EnvDTCapabilities)
		require.NotNil(t, capabilitiesEnv)
		assert.Equal(t, activegate.KubeMonCapability.ArgumentName, capabilitiesEnv.Value)

		namespaceEnv := k8senv.Find(container.Env, agconsts.EnvDTIDSeedNamespace)
		require.NotNil(t, namespaceEnv)
		assert.Equal(t, dk.Namespace, namespaceEnv.Value)

		clusterIDEnv := k8senv.Find(container.Env, agconsts.EnvDTIDSeedClusterID)
		require.NotNil(t, clusterIDEnv)
		assert.Equal(t, dk.Status.KubeSystemUUID, clusterIDEnv.Value)

		metadataEnv := k8senv.Find(container.Env, deploymentmetadata.EnvDTDeploymentMetadata)
		require.NotNil(t, metadataEnv)
		require.NotNil(t, metadataEnv.ValueFrom)
		require.NotNil(t, metadataEnv.ValueFrom.ConfigMapKeyRef)
		assert.Equal(t, deploymentmetadata.KubemonMetadataKey, metadataEnv.ValueFrom.ConfigMapKeyRef.Key)

		assert.NotNil(t, k8senv.Find(container.Env, connectioninfo.EnvDTTenant))
		assert.NotNil(t, k8senv.Find(container.Env, connectioninfo.EnvDTServer))
		assert.NotNil(t, k8senv.Find(container.Env, "CUSTOM"))
	})

	t.Run("tenant token volume mount", func(t *testing.T) {
		dk := newTestDynaKube(true)
		sts := reconcileAndGetSTS(t, dk)

		require.Len(t, sts.Spec.Template.Spec.Containers, 1)
		container := sts.Spec.Template.Spec.Containers[0]

		require.Len(t, container.VolumeMounts, 3)
		assert.Equal(t, connectioninfo.TenantSecretVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, connectioninfo.TenantTokenMountPoint, container.VolumeMounts[0].MountPath)
		assert.Equal(t, connectioninfo.TenantTokenKey, container.VolumeMounts[0].SubPath)
		assert.True(t, container.VolumeMounts[0].ReadOnly)
		assert.True(t, hasTenantSecretVolume(sts, dk))
		assert.NotEmpty(t, sts.Spec.Template.Annotations[statefulset.AnnotationTenantTokenHash])
	})

	t.Run("auth token volume mount", func(t *testing.T) {
		dk := newTestDynaKube(true)
		sts := reconcileAndGetSTS(t, dk)

		require.Len(t, sts.Spec.Template.Spec.Containers, 1)
		container := sts.Spec.Template.Spec.Containers[0]

		require.Len(t, container.VolumeMounts, 3)
		assert.Equal(t, statefulset.AuthTokenVolumeName, container.VolumeMounts[1].Name)
		assert.Equal(t, agconsts.AuthTokenMountPoint, container.VolumeMounts[1].MountPath)
		assert.Equal(t, kubemonauthtoken.SecretKey, container.VolumeMounts[1].SubPath)
		assert.True(t, container.VolumeMounts[1].ReadOnly)
		assert.True(t, hasAuthTokenVolume(sts, dk))
		assert.NotEmpty(t, sts.Spec.Template.Annotations[statefulset.AnnotationAuthTokenHash])
	})

	t.Run("update strategy with rolling partition", func(t *testing.T) {
		dk := newTestDynaKube(true)
		partition := int32(2)
		dk.Spec.KubernetesMonitoring.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{Partition: &partition}
		sts := reconcileAndGetSTS(t, dk)

		require.Equal(t, appsv1.RollingUpdateStatefulSetStrategyType, sts.Spec.UpdateStrategy.Type)
		require.NotNil(t, sts.Spec.UpdateStrategy.RollingUpdate)
		require.NotNil(t, sts.Spec.UpdateStrategy.RollingUpdate.Partition)
		assert.Equal(t, partition, *sts.Spec.UpdateStrategy.RollingUpdate.Partition)
	})

	t.Run("pod scheduling overrides", func(t *testing.T) {
		dk := newTestDynaKube(true)
		grace := int64(45)
		dk.Spec.KubernetesMonitoring.DNSPolicy = corev1.DNSNone
		dk.Spec.KubernetesMonitoring.PriorityClassName = "high-priority"
		dk.Spec.KubernetesMonitoring.TerminationGracePeriodSeconds = &grace
		sts := reconcileAndGetSTS(t, dk)

		assert.Equal(t, corev1.DNSNone, sts.Spec.Template.Spec.DNSPolicy)
		assert.Equal(t, "high-priority", sts.Spec.Template.Spec.PriorityClassName)
		assert.Equal(t, grace, *sts.Spec.Template.Spec.TerminationGracePeriodSeconds)
	})

	t.Run("storage volumes", func(t *testing.T) {
		dk := newTestDynaKube(true)
		sts := reconcileAndGetSTS(t, dk)

		assert.Empty(t, sts.Spec.VolumeClaimTemplates)
		require.Len(t, sts.Spec.Template.Spec.Volumes, 3)
		assert.Equal(t, statefulset.StorageVolumeName, sts.Spec.Template.Spec.Volumes[2].Name)
		assert.NotNil(t, sts.Spec.Template.Spec.Volumes[2].EmptyDir)
	})
}

// TestReconcileWriteFailures covers the two write/read error paths after the StatefulSet is built:
// the create itself failing, and the follow-up Get used to evaluate rollout completion.
func TestReconcileWriteFailures(t *testing.T) {
	t.Run("returns error when statefulset create fails", func(t *testing.T) {
		dk := newTestDynaKube(true)
		errCreate := errors.New("kube api error")
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if isStatefulSet(obj) {
					return errCreate
				}

				return c.Create(ctx, obj, opts...)
			},
		}, dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))

		err := statefulset.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.ErrorIs(t, err, errCreate)
	})

	t.Run("returns error when re-getting the statefulset fails", func(t *testing.T) {
		dk := newTestDynaKube(true)
		created := false
		errGet := errors.New("kube api error")
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if isStatefulSet(obj) {
					created = true
				}

				return c.Create(ctx, obj, opts...)
			},
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if created && isStatefulSet(obj) {
					return errGet
				}

				return c.Get(ctx, key, obj, opts...)
			},
		}, dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))

		err := statefulset.NewReconciler(fakeClient).Reconcile(t.Context(), dk)
		require.ErrorIs(t, err, errGet)
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

func reconcileAndGetSTS(t *testing.T, dk *dynakube.DynaKube) *appsv1.StatefulSet {
	t.Helper()
	fakeClient := fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))
	require.ErrorIs(t, statefulset.NewReconciler(fakeClient).Reconcile(t.Context(), dk), k8sstatefulset.ErrRolloutInProgress)

	return requireTestStatefulSet(t, t.Context(), fakeClient, dk)
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

	dk.Status.KubeSystemUUID = "test-cluster-uuid"

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
	require.NoError(t, clt.Get(ctx, client.ObjectKey{Name: dk.KubernetesMonitoring().GetStatefulSetName(), Namespace: dk.Namespace}, sts))

	return sts
}

func isStatefulSet(obj client.Object) bool {
	_, ok := obj.(*appsv1.StatefulSet)

	return ok
}

func newTestAuthTokenSecret(dk *dynakube.DynaKube) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.KubernetesMonitoring().GetAuthTokenSecretName(),
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{
			kubemonauthtoken.SecretKey: []byte("test-auth-token"),
		},
	}
}

func hasTenantSecretVolume(sts *appsv1.StatefulSet, dk *dynakube.DynaKube) bool {
	return slices.ContainsFunc(sts.Spec.Template.Spec.Volumes, func(v corev1.Volume) bool {
		return v.Name == connectioninfo.TenantSecretVolumeName &&
			v.Secret != nil &&
			v.Secret.SecretName == dk.KubernetesMonitoring().GetTenantSecretName()
	})
}

func hasAuthTokenVolume(sts *appsv1.StatefulSet, dk *dynakube.DynaKube) bool {
	return slices.ContainsFunc(sts.Spec.Template.Spec.Volumes, func(v corev1.Volume) bool {
		return v.Name == statefulset.AuthTokenVolumeName &&
			v.Secret != nil &&
			v.Secret.SecretName == dk.KubernetesMonitoring().GetAuthTokenSecretName()
	})
}
