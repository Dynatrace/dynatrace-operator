// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package statefulset_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	kspmapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/version"
	operatorconsts "github.com/Dynatrace/dynatrace-operator/pkg/consts"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	kubemonauthtoken "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/authtoken"
	kubemoncustomproperties "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8smount"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8svolume"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	k8sversion "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/version"
	operatorversion "github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	imageclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	versionclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/version"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		"missing tenant secret": {
			// Only auth token secret seeded — tenant secret Get returns NotFound first.
			assertError: func(t *testing.T, err error) {
				require.True(t, k8serrors.IsNotFound(err))
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dk := newTestDynaKube()
			if test.mutate != nil {
				test.mutate(dk)
			}

			err := statefulset.NewReconciler(fake.NewClient(dk, newTestAuthTokenSecret(dk))).Reconcile(t.Context(), dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))
			require.Error(t, err)
			require.NotErrorIs(t, err, k8sstatefulset.ErrRolloutInProgress)
			test.assertError(t, err)
		})
	}
}

func TestReconcileMissingKubeSystemUID(t *testing.T) {
	dk := newTestDynaKube()
	dk.Status.KubeSystemUUID = ""
	err := statefulset.NewReconciler(fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))).Reconcile(t.Context(), dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))
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
			dk := newTestDynaKube()
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

			err := statefulset.NewReconciler(fake.NewClient(dk, tenantSecret, authSecret)).Reconcile(t.Context(), dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))
			require.ErrorIs(t, err, test.expectedErr)
		})
	}
}

// TestReconcileResolveReplicasReadFailure verifies that a non-NotFound StatefulSet read error
// from ResolveReplicas exits reconcile before any StatefulSet write.
func TestReconcileResolveReplicasReadFailure(t *testing.T) {
	dk := newTestDynaKube()
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

	// dk has a custom image so neither client is invoked; mocks with no expectations verify that
	err := statefulset.NewReconciler(fakeClient).
		Reconcile(t.Context(), dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))
	require.Error(t, err)
	require.NotErrorIs(t, err, k8sstatefulset.ErrRolloutInProgress)
	assert.False(t, writeAttempted)
}

// TestReconcileBuildsStatefulSet covers the shape of the produced StatefulSet. The fake client has
// no StatefulSet controller, so reconcile always reports rollout in progress.
func TestReconcileBuildsStatefulSet(t *testing.T) {
	t.Run("container has named https and http ports for service routing", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		container := sts.Spec.Template.Spec.Containers[0]

		assert.True(t, slices.ContainsFunc(container.Ports, func(p corev1.ContainerPort) bool {
			return p.Name == "https" && p.ContainerPort == 9999
		}), "expected https:9999 container port")

		assert.True(t, slices.ContainsFunc(container.Ports, func(p corev1.ContainerPort) bool {
			return p.Name == "http" && p.ContainerPort == 9998
		}), "expected http:9998 container port")
	})

	t.Run("container identity and service account", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.Len(t, sts.Spec.Template.Spec.Containers, 1)
		container := sts.Spec.Template.Spec.Containers[0]

		assert.Equal(t, statefulset.ContainerName, container.Name)
		assert.Equal(t, dk.KubernetesMonitoring().GetCustomImage(), container.Image)
		assert.Equal(t, dk.KubernetesMonitoring().GetServiceAccountName(), sts.Spec.Template.Spec.ServiceAccountName)
	})

	t.Run("env vars: capabilities, seed envs, deployment metadata, connection info, and custom", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.Env = []corev1.EnvVar{{Name: "CUSTOM", Value: "value"}}
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

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
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.Len(t, sts.Spec.Template.Spec.Containers, 1)
		container := sts.Spec.Template.Spec.Containers[0]

		require.Len(t, container.VolumeMounts, 9)
		assert.Equal(t, connectioninfo.TenantSecretVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, connectioninfo.TenantTokenMountPoint, container.VolumeMounts[0].MountPath)
		assert.Equal(t, connectioninfo.TenantTokenKey, container.VolumeMounts[0].SubPath)
		assert.True(t, container.VolumeMounts[0].ReadOnly)
		assert.True(t, hasTenantSecretVolume(sts, dk))
		assert.NotEmpty(t, sts.Spec.Template.Annotations[statefulset.AnnotationTenantTokenHash])
	})

	t.Run("auth token volume mount", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.Len(t, sts.Spec.Template.Spec.Containers, 1)
		container := sts.Spec.Template.Spec.Containers[0]

		require.Len(t, container.VolumeMounts, 9)
		assert.Equal(t, statefulset.AuthTokenVolumeName, container.VolumeMounts[1].Name)
		assert.Equal(t, agconsts.AuthTokenMountPoint, container.VolumeMounts[1].MountPath)
		assert.Equal(t, kubemonauthtoken.SecretKey, container.VolumeMounts[1].SubPath)
		assert.True(t, container.VolumeMounts[1].ReadOnly)
		assert.True(t, hasAuthTokenVolume(sts, dk))
		assert.NotEmpty(t, sts.Spec.Template.Annotations[statefulset.AnnotationAuthTokenHash])
	})

	t.Run("no custom properties volume when CustomProperties is nil", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		assert.False(t, hasCustomPropertiesVolume(sts, dk))
		assert.Empty(t, sts.Spec.Template.Annotations[statefulset.AnnotationCustomPropertiesHash])
	})

	t.Run("custom properties volume included and required when CustomProperties is set", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.CustomProperties = &value.Source{Value: "key=value"}
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.Len(t, sts.Spec.Template.Spec.Containers, 1)
		container := sts.Spec.Template.Spec.Containers[0]

		require.Len(t, container.VolumeMounts, 10)
		assert.Equal(t, kubemoncustomproperties.VolumeName, container.VolumeMounts[9].Name)
		assert.Equal(t, kubemoncustomproperties.MountPath, container.VolumeMounts[9].MountPath)
		assert.Equal(t, kubemoncustomproperties.DataKey, container.VolumeMounts[9].SubPath)
		assert.True(t, container.VolumeMounts[9].ReadOnly)
		assert.True(t, hasCustomPropertiesVolume(sts, dk))
	})

	t.Run("custom properties hash annotation is added when a secret exists", func(t *testing.T) {
		dk := newTestDynaKube()
		fakeClient := fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk), newTestCustomPropertiesSecret(dk))
		require.ErrorIs(t, statefulset.NewReconciler(fakeClient).Reconcile(t.Context(), dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t)), k8sstatefulset.ErrRolloutInProgress)

		sts := requireTestStatefulSet(t, t.Context(), fakeClient, dk)
		assert.NotEmpty(t, sts.Spec.Template.Annotations[statefulset.AnnotationCustomPropertiesHash])
	})

	t.Run("KSPM disabled, no token volume or annotation", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		assert.Empty(t, sts.Spec.Template.Annotations[statefulset.AnnotationKSPMTokenHash])
		assert.False(t, slices.ContainsFunc(sts.Spec.Template.Spec.Volumes, func(v corev1.Volume) bool {
			return v.Secret != nil && v.Secret.SecretName == dk.KSPM().GetTokenSecretName()
		}))
	})

	t.Run("KSPM enabled, token volume and mount are added", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KSPM = &kspmapi.Spec{}
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		kspmVolume := k8svolume.FindByName(sts.Spec.Template.Spec.Volumes, "kspm-token")
		require.NotNil(t, kspmVolume, "KSPM token volume not found")
		require.NotNil(t, kspmVolume.Secret)
		require.NotNil(t, kspmVolume.Secret.DefaultMode)
		assert.Equal(t, dk.KSPM().GetTokenSecretName(), kspmVolume.Secret.SecretName)
		assert.EqualValues(t, 0o640, *kspmVolume.Secret.DefaultMode)

		container := sts.Spec.Template.Spec.Containers[0]
		kspmMount, err := k8smount.Find(container.VolumeMounts, kspmVolume.Name)
		require.NoError(t, err)
		assert.True(t, kspmMount.ReadOnly)
		assert.Equal(t, operatorconsts.DTComponentsSecretsRootDir+"/tokens/kspm/node-configuration-collector", kspmMount.MountPath)
		assert.Equal(t, kspmapi.TokenSecretKey, kspmMount.SubPath)
	})

	t.Run("KSPM enabled, token hash annotation is set on pod", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KSPM = &kspmapi.Spec{}
		dk.Status.KSPM.TokenSecretHash = "abc123"
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		assert.Equal(t, "abc123", sts.Spec.Template.Annotations[statefulset.AnnotationKSPMTokenHash])
	})

	t.Run("update strategy with rolling partition", func(t *testing.T) {
		dk := newTestDynaKube()
		partition := int32(2)
		dk.Spec.KubernetesMonitoring.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{Partition: &partition}
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.Equal(t, appsv1.RollingUpdateStatefulSetStrategyType, sts.Spec.UpdateStrategy.Type)
		require.NotNil(t, sts.Spec.UpdateStrategy.RollingUpdate)
		require.NotNil(t, sts.Spec.UpdateStrategy.RollingUpdate.Partition)
		assert.Equal(t, partition, *sts.Spec.UpdateStrategy.RollingUpdate.Partition)
	})

	t.Run("pod scheduling overrides", func(t *testing.T) {
		dk := newTestDynaKube()
		grace := int64(45)
		dk.Spec.KubernetesMonitoring.DNSPolicy = corev1.DNSNone
		dk.Spec.KubernetesMonitoring.PriorityClassName = "high-priority"
		dk.Spec.KubernetesMonitoring.TerminationGracePeriodSeconds = &grace
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		assert.Equal(t, corev1.DNSNone, sts.Spec.Template.Spec.DNSPolicy)
		assert.Equal(t, "high-priority", sts.Spec.Template.Spec.PriorityClassName)
		assert.Equal(t, grace, *sts.Spec.Template.Spec.TerminationGracePeriodSeconds)
	})

	t.Run("storage volumes", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		assert.Empty(t, sts.Spec.VolumeClaimTemplates)
		require.Len(t, sts.Spec.Template.Spec.Volumes, 10)
		assert.Equal(t, statefulset.StorageVolumeName, sts.Spec.Template.Spec.Volumes[2].Name)
		assert.NotNil(t, sts.Spec.Template.Spec.Volumes[2].EmptyDir)
	})

	t.Run("image pull secrets: tenant registry secret included by default", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.Len(t, sts.Spec.Template.Spec.ImagePullSecrets, 1)
		assert.Equal(t, dk.TenantRegistryPullSecretName(), sts.Spec.Template.Spec.ImagePullSecrets[0].Name)
	})

	t.Run("image pull secrets: tenant registry secret omitted when public registry is enabled", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Annotations = map[string]string{exp.UsePublicRegistryKey: "true"}
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		assert.Empty(t, sts.Spec.Template.Spec.ImagePullSecrets)
	})

	t.Run("injection split-mounts annotation is always set to true", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		assert.Equal(t, "true", sts.Spec.Template.Annotations[mutator.AnnotationInjectionSplitMounts])
	})

	t.Run("image pull secrets: custom pull secret is included alongside the tenant registry secret", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.CustomPullSecret = "my-custom-pull-secret"
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		names := make([]string, 0, len(sts.Spec.Template.Spec.ImagePullSecrets))
		for _, ref := range sts.Spec.Template.Spec.ImagePullSecrets {
			names = append(names, ref.Name)
		}

		assert.Contains(t, names, dk.TenantRegistryPullSecretName())
		assert.Contains(t, names, "my-custom-pull-secret")
	})

	t.Run("KSPM disabled, no automatic TLS secret", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		assert.False(t, slices.ContainsFunc(sts.Spec.Template.Spec.Volumes, func(v corev1.Volume) bool {
			return v.Secret != nil && v.Secret.SecretName == dk.KubernetesMonitoring().GetAutoTLSSecretName()
		}))
	})

	t.Run("KSPM disabled, neither custom nor automatic TLS secret is created", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.TLSCertsRef = &kubemonapi.TLSCertsRef{
			SecretName: "custom-tls-secret",
		}
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		assert.False(t, slices.ContainsFunc(sts.Spec.Template.Spec.Volumes, func(v corev1.Volume) bool {
			return v.Secret != nil && (v.Secret.SecretName == "custom-tls-secret" || v.Secret.SecretName == dk.KubernetesMonitoring().GetAutoTLSSecretName())
		}))
	})

	t.Run("KSPM enabled, automatic TLS certificate volume and mount are added", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KSPM = &kspmapi.Spec{}
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		certVolume := k8svolume.FindByName(sts.Spec.Template.Spec.Volumes, agconsts.CertsVolumeName)
		require.NotNil(t, certVolume, agconsts.CertsVolumeName+" volume not found")
		require.NotNil(t, certVolume.Secret)
		require.NotNil(t, certVolume.Secret.DefaultMode)
		assert.Equal(t, dk.KubernetesMonitoring().GetAutoTLSSecretName(), certVolume.Secret.SecretName)
		assert.EqualValues(t, 0o640, *certVolume.Secret.DefaultMode)

		container := sts.Spec.Template.Spec.Containers[0]
		certMount, err := k8smount.Find(container.VolumeMounts, agconsts.CertsVolumeName)
		require.NoError(t, err)
		assert.True(t, certMount.ReadOnly)
		assert.Equal(t, agconsts.CertsMountPath, certMount.MountPath)
	})

	t.Run("KSPM enabled, custom TLS certificate volume and mount are added", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.TLSCertsRef = &kubemonapi.TLSCertsRef{
			SecretName: "custom-tls-secret",
		}
		dk.Spec.KSPM = &kspmapi.Spec{}
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		certVolume := k8svolume.FindByName(sts.Spec.Template.Spec.Volumes, agconsts.CertsVolumeName)
		require.NotNil(t, certVolume, agconsts.CertsVolumeName+" volume not found")
		require.NotNil(t, certVolume.Secret)
		require.NotNil(t, certVolume.Secret.DefaultMode)
		// dk.KubernetesMonitoring().GetTLSSecretName() not called to make sure custom secret has higher priority
		assert.Equal(t, "custom-tls-secret", certVolume.Secret.SecretName)
		assert.EqualValues(t, 0o640, *certVolume.Secret.DefaultMode)

		container := sts.Spec.Template.Spec.Containers[0]
		certMount, err := k8smount.Find(container.VolumeMounts, agconsts.CertsVolumeName)
		require.NoError(t, err)
		assert.True(t, certMount.ReadOnly)
		assert.Equal(t, agconsts.CertsMountPath, certMount.MountPath)
	})

	t.Run("securityContext present", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))
		require.NotNil(t, sts.Spec.Template.Spec.SecurityContext)
		require.NotNil(t, sts.Spec.Template.Spec.Containers[0].SecurityContext)
		require.NotNil(t, sts.Spec.Template.Spec.InitContainers[0].SecurityContext)

		checkSecurityContext := func(t *testing.T, sc *corev1.SecurityContext) {
			require.NotNil(t, sc.Privileged)
			assert.False(t, *sc.Privileged)

			require.NotNil(t, sc.AllowPrivilegeEscalation)
			assert.False(t, *sc.AllowPrivilegeEscalation)

			require.NotNil(t, sc.RunAsNonRoot)
			assert.True(t, *sc.RunAsNonRoot)

			require.NotNil(t, sc.RunAsUser)
			assert.Equal(t, agconsts.DockerImageUser, *sc.RunAsUser)

			require.NotNil(t, sc.RunAsGroup)
			assert.Equal(t, agconsts.DockerImageUser, *sc.RunAsGroup)

			require.NotNil(t, sc.Capabilities)
			assert.Empty(t, sc.Capabilities.Add)
			require.Len(t, sc.Capabilities.Drop, 1)
			assert.Contains(t, sc.Capabilities.Drop, corev1.Capability("ALL"))

			require.NotNil(t, sc.SeccompProfile)
			assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, sc.SeccompProfile.Type)

			require.NotNil(t, sc.ReadOnlyRootFilesystem)
			assert.True(t, *sc.ReadOnlyRootFilesystem)
		}

		checkSecurityContext(t, sts.Spec.Template.Spec.Containers[0].SecurityContext)
		checkSecurityContext(t, sts.Spec.Template.Spec.InitContainers[0].SecurityContext)
	})

	t.Run("securityContext.AppArmorProfile is not set on cluster 1.30", func(t *testing.T) {
		t.Cleanup(k8sversion.DisableCacheForTest(30))

		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.Annotations = map[string]string{
			corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + "kubemon": "runtime/default",
		}
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.NotNil(t, sts.Spec.Template.Spec.InitContainers[0].SecurityContext)
		assert.Nil(t, sts.Spec.Template.Spec.InitContainers[0].SecurityContext.AppArmorProfile)

		require.NotNil(t, sts.Spec.Template.Spec.Containers[0].SecurityContext)
		require.Nil(t, sts.Spec.Template.Spec.Containers[0].SecurityContext.AppArmorProfile)

		assert.Contains(t, sts.Spec.Template.Annotations, corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix+"kubemon")
	})

	t.Run("securityContext.AppArmorProfile is set on cluster 1.31 for given container", func(t *testing.T) {
		t.Cleanup(k8sversion.DisableCacheForTest(31))

		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.Annotations = map[string]string{
			corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + statefulset.ContainerName: "runtime/default",
		}
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.NotNil(t, sts.Spec.Template.Spec.InitContainers[0].SecurityContext)
		assert.Nil(t, sts.Spec.Template.Spec.InitContainers[0].SecurityContext.AppArmorProfile)

		require.NotNil(t, sts.Spec.Template.Spec.Containers[0].SecurityContext)
		require.NotNil(t, sts.Spec.Template.Spec.Containers[0].SecurityContext.AppArmorProfile)
		assert.Equal(t, corev1.AppArmorProfileTypeRuntimeDefault, sts.Spec.Template.Spec.Containers[0].SecurityContext.AppArmorProfile.Type)

		assert.NotContains(t, sts.Spec.Template.Annotations, corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix+statefulset.ContainerName)
	})

	t.Run("securityContext.AppArmorProfile is set on cluster 1.31 for both containers", func(t *testing.T) {
		t.Cleanup(k8sversion.DisableCacheForTest(31))

		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.Annotations = map[string]string{
			corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + statefulset.ContainerName:          "runtime/default",
			corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + agconsts.InitContainerTemplateName: "unconfined",
		}
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.NotNil(t, sts.Spec.Template.Spec.InitContainers[0].SecurityContext)
		assert.NotNil(t, sts.Spec.Template.Spec.InitContainers[0].SecurityContext.AppArmorProfile)
		assert.Equal(t, corev1.AppArmorProfileTypeUnconfined, sts.Spec.Template.Spec.InitContainers[0].SecurityContext.AppArmorProfile.Type)

		require.NotNil(t, sts.Spec.Template.Spec.Containers[0].SecurityContext)
		require.NotNil(t, sts.Spec.Template.Spec.Containers[0].SecurityContext.AppArmorProfile)
		assert.Equal(t, corev1.AppArmorProfileTypeRuntimeDefault, sts.Spec.Template.Spec.Containers[0].SecurityContext.AppArmorProfile.Type)

		assert.NotContains(t, sts.Spec.Template.Annotations, corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix+statefulset.ContainerName)
		assert.NotContains(t, sts.Spec.Template.Annotations, corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix+agconsts.InitContainerTemplateName)
	})
}

func TestReconcileBuildsStatefulSetVolumes(t *testing.T) {
	t.Run("basic container volume mounts", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.Len(t, sts.Spec.Template.Spec.Containers, 1)
		container := sts.Spec.Template.Spec.Containers[0]
		require.Len(t, container.VolumeMounts, 9)

		mounts := []corev1.VolumeMount{
			{
				Name:      connectioninfo.TenantSecretVolumeName,
				ReadOnly:  true,
				MountPath: connectioninfo.TenantTokenMountPoint,
				SubPath:   connectioninfo.TenantTokenKey,
			},
			{
				Name:      statefulset.AuthTokenVolumeName,
				ReadOnly:  true,
				MountPath: agconsts.AuthTokenMountPoint,
				SubPath:   kubemonauthtoken.SecretKey,
			},
			{
				Name:      statefulset.StorageVolumeName,
				MountPath: agconsts.GatewayTmpMountPath,
			},
			{
				ReadOnly:  false,
				Name:      agconsts.GatewaySslVolumeName,
				MountPath: agconsts.GatewaySslMountPath,
			},
			{
				ReadOnly:  false,
				Name:      agconsts.GatewayLibTempVolumeName,
				MountPath: agconsts.GatewayLibTempMountPath,
			},
			{
				ReadOnly:  false,
				Name:      agconsts.GatewayDataVolumeName,
				MountPath: agconsts.GatewayDataMountPath,
			},
			{
				ReadOnly:  false,
				Name:      agconsts.GatewayLogVolumeName,
				MountPath: agconsts.GatewayLogMountPath,
			},
			{
				ReadOnly:  false,
				Name:      agconsts.GatewayConfigVolumeName,
				MountPath: agconsts.GatewayConfigMountPath,
			},
			{
				ReadOnly:  true,
				Name:      agconsts.TrustStoreVolumeName,
				MountPath: agconsts.TrustStoreCacertsMountPath,
				SubPath:   agconsts.K8sCertificateFile,
			},
		}

		for _, mount := range mounts {
			assert.Contains(t, container.VolumeMounts, mount, "valid %s volumeMount not found", mount.Name)
		}
	})

	t.Run("initContainer volume mounts", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.Len(t, sts.Spec.Template.Spec.InitContainers, 1)
		container := sts.Spec.Template.Spec.InitContainers[0]
		require.Len(t, container.VolumeMounts, 2)

		mounts := []corev1.VolumeMount{
			{
				ReadOnly:  false,
				Name:      agconsts.TrustStoreVolumeName,
				MountPath: agconsts.GatewaySslMountPath,
			},
			{
				ReadOnly:  false,
				Name:      agconsts.InitCertLoaderWorkDirVolumeName,
				MountPath: agconsts.InitCertLoaderWorkDirMountPath,
			},
		}

		for _, mount := range mounts {
			assert.Contains(t, container.VolumeMounts, mount, "valid %s volumeMount not found", mount.Name)
		}
	})

	t.Run("basic volumes", func(t *testing.T) {
		dk := newTestDynaKube()
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

		require.Len(t, sts.Spec.Template.Spec.Volumes, 10)

		km := dk.KubernetesMonitoring()
		volumes := []corev1.Volume{
			{
				Name: connectioninfo.TenantSecretVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  km.GetTenantSecretName(),
						DefaultMode: new(int32(0o640)),
						Optional:    new(false),
					},
				},
			},
			{
				Name: statefulset.AuthTokenVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  km.GetAuthTokenSecretName(),
						DefaultMode: new(int32(0o640)),
						Optional:    new(false),
					},
				},
			},
			{
				Name:         statefulset.StorageVolumeName,
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			},
			{
				Name: agconsts.GatewayLibTempVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: agconsts.GatewayDataVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: agconsts.GatewayLogVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: agconsts.GatewayConfigVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: agconsts.TrustStoreVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: agconsts.GatewaySslVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name:         agconsts.InitCertLoaderWorkDirVolumeName,
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			},
		}

		for _, volume := range volumes {
			assert.Contains(t, sts.Spec.Template.Spec.Volumes, volume, "valid %s volume not found", volume.Name)
		}
	})
}

func TestReconcileMetadata(t *testing.T) {
	dk := newTestDynaKube()

	sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))

	expectedLabels := map[string]string{
		k8slabel.AppNameLabel:         k8slabel.KubeMonComponentLabel,
		k8slabel.AppInstanceLabel:     dk.Name,
		k8slabel.AppManagedByLabel:    operatorversion.AppName,
		k8slabel.OperatorVersionLabel: operatorversion.Version,
	}
	expectedSelector := map[string]string{
		k8slabel.AppNameLabel:      k8slabel.KubeMonComponentLabel,
		k8slabel.AppInstanceLabel:  dk.Name,
		k8slabel.AppManagedByLabel: operatorversion.AppName,
	}

	assert.Equal(t, expectedLabels, sts.Labels)
	assert.Equal(t, expectedLabels, sts.Spec.Template.Labels)
	assert.Equal(t, expectedSelector, sts.Spec.Selector.MatchLabels)
}

// TestResolveImage verifies all three image resolution paths and their error handling.
func TestResolveImage(t *testing.T) {
	t.Run("custom image is used directly without calling any client", func(t *testing.T) {
		dk := newTestDynaKube()

		// mocks with no expectations verify that neither client is invoked on the custom-image path
		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))
		assert.Equal(t, dk.KubernetesMonitoring().GetCustomImage(), sts.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, dk.KubernetesMonitoring().GetCustomImage(), dk.KubernetesMonitoring().ResolvedImage)
	})

	t.Run("public registry uses image client", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.Image = ""
		dk.Annotations = map[string]string{exp.UsePublicRegistryKey: "true"}

		expectedTag := "1.2.3"
		expectedImage := "public.registry.example.com/linux/activegate:" + expectedTag
		mockImgClient := imageclientmock.NewClient(t)
		mockImgClient.EXPECT().
			GetComponentLatestInfo(mock.Anything, image.ActiveGate, "").
			Return(&image.Info{URI: expectedImage}, nil)

		sts := reconcileAndGetSTS(t, dk, mockImgClient, versionclientmock.NewClient(t))
		assert.Equal(t, expectedImage, sts.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, expectedImage, dk.KubernetesMonitoring().ResolvedImage)
	})

	t.Run("public registry uses registry override when set", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.Image = ""
		dk.Annotations = map[string]string{exp.UsePublicRegistryKey: "true"}
		dk.Spec.PublicRegistryOverride = "my.registry.example.com"

		expectedTag := "1.2.3"
		expectedImage := "my.registry.example.com/linux/activegate:" + expectedTag
		mockImgClient := imageclientmock.NewClient(t)
		mockImgClient.EXPECT().
			GetComponentLatestInfo(mock.Anything, image.ActiveGate, "my.registry.example.com").
			Return(&image.Info{URI: expectedImage}, nil)

		sts := reconcileAndGetSTS(t, dk, mockImgClient, versionclientmock.NewClient(t))
		assert.Equal(t, expectedImage, sts.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, expectedImage, dk.KubernetesMonitoring().ResolvedImage)
	})

	t.Run("default image is built from version client and api url", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.Image = ""

		expectedVersion := "1.2.3.4"
		mockVerClient := versionclientmock.NewClient(t)
		mockVerClient.EXPECT().
			GetLatestActiveGateVersion(mock.Anything, installer.OSUnix).
			Return(expectedVersion, nil)

		expectedImage := dk.KubernetesMonitoring().GetDefaultImage(expectedVersion)
		require.NotEmpty(t, expectedImage, "test precondition: APIURL must produce a non-empty default image")

		sts := reconcileAndGetSTS(t, dk, imageclientmock.NewClient(t), mockVerClient)
		assert.Equal(t, expectedImage, sts.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, expectedImage, dk.KubernetesMonitoring().ResolvedImage)
	})

	t.Run("image client error is propagated", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.Image = ""
		dk.Annotations = map[string]string{exp.UsePublicRegistryKey: "true"}

		expectedErr := errors.New("image client error")
		mockImgClient := imageclientmock.NewClient(t)
		mockImgClient.EXPECT().
			GetComponentLatestInfo(mock.Anything, image.ActiveGate, "").
			Return(nil, expectedErr)

		err := statefulset.NewReconciler(fake.NewClient(dk, newTestTenantSecret(dk))).
			Reconcile(t.Context(), dk, mockImgClient, versionclientmock.NewClient(t))
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("version client error is propagated", func(t *testing.T) {
		dk := newTestDynaKube()
		dk.Spec.KubernetesMonitoring.Image = ""

		expectedErr := errors.New("version client error")
		mockVerClient := versionclientmock.NewClient(t)
		mockVerClient.EXPECT().
			GetLatestActiveGateVersion(mock.Anything, installer.OSUnix).
			Return("", expectedErr)

		err := statefulset.NewReconciler(fake.NewClient(dk, newTestTenantSecret(dk))).
			Reconcile(t.Context(), dk, imageclientmock.NewClient(t), mockVerClient)
		require.ErrorIs(t, err, expectedErr)
	})
}

// TestReconcileWriteFailures covers the two write/read error paths after the StatefulSet is built:
// the create itself failing, and the follow-up Get used to evaluate rollout completion.
func TestReconcileWriteFailures(t *testing.T) {
	t.Run("returns error when statefulset create fails", func(t *testing.T) {
		dk := newTestDynaKube()
		errCreate := errors.New("kube api error")
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if isStatefulSet(obj) {
					return errCreate
				}

				return c.Create(ctx, obj, opts...)
			},
		}, dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))

		err := statefulset.NewReconciler(fakeClient).
			Reconcile(t.Context(), dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))
		require.ErrorIs(t, err, errCreate)
	})

	t.Run("returns error when re-getting the statefulset fails", func(t *testing.T) {
		dk := newTestDynaKube()
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

		err := statefulset.NewReconciler(fakeClient).
			Reconcile(t.Context(), dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t))
		require.ErrorIs(t, err, errGet)
	})
}

// TestReconcileCleanupDeleteFailure covers the delete failure on the cleanup path.
func TestReconcileCleanupDeleteFailure(t *testing.T) {
	dk := newTestDynaKube()
	dk.Spec.KubernetesMonitoring = nil
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

	// disabled path skips image resolution entirely; mocks with no expectations verify that
	require.Error(t, statefulset.NewReconciler(fakeClient).
		Reconcile(t.Context(), dk, imageclientmock.NewClient(t), versionclientmock.NewClient(t)))
}

func reconcileAndGetSTS(t *testing.T, dk *dynakube.DynaKube, imgClient image.Client, verClient version.Client) *appsv1.StatefulSet {
	t.Helper()
	fakeClient := fake.NewClient(dk, newTestTenantSecret(dk), newTestAuthTokenSecret(dk))
	require.ErrorIs(t, statefulset.NewReconciler(fakeClient).Reconcile(t.Context(), dk, imgClient, verClient), k8sstatefulset.ErrRolloutInProgress)

	return requireTestStatefulSet(t, t.Context(), fakeClient, dk)
}

func newTestDynaKube() *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: testNamespace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://tenant.live.dynatrace.com/api",
			KubernetesMonitoring: &kubemonapi.Spec{
				StatefulSetProperties: kubemonapi.StatefulSetProperties{
					Image: "registry.example.com/linux/activegate:1.2.3",
				},
			},
		},
		Status: dynakube.DynaKubeStatus{
			KubeSystemUUID: "test-cluster-uuid", // set by the parent controller before any kubemon reconciler runs
		},
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

func newTestCustomPropertiesSecret(dk *dynakube.DynaKube) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.KubernetesMonitoring().GetCustomPropertiesSecretName(),
			Namespace: dk.Namespace,
		},
		Data: map[string][]byte{
			kubemoncustomproperties.DataKey: []byte("[section]\nkey=value"),
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

func hasCustomPropertiesVolume(sts *appsv1.StatefulSet, dk *dynakube.DynaKube) bool {
	return slices.ContainsFunc(sts.Spec.Template.Spec.Volumes, func(v corev1.Volume) bool {
		return v.Name == kubemoncustomproperties.VolumeName &&
			v.Secret != nil &&
			v.Secret.SecretName == dk.KubernetesMonitoring().GetCustomPropertiesSecretName() &&
			v.Secret.Optional != nil && !*v.Secret.Optional
	})
}
