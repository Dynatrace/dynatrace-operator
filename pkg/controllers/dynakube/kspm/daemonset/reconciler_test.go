// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package daemonset

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	sharedimage "github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	dtimage "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/version"
	imageclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	dkName      = "test-name"
	dkNamespace = "test-namespace"

	testImageRepo         = "test-repo/dynatrace-k8s-node-config-collector"
	testImageTag          = "1.289.0"
	testFleetMgmtImageURI = "registry.example.com/dynatrace-k8s-node-config-collector:1.300.0"
	testRegistryOverride  = "my.registry.example.com"
)

func TestReconcile(t *testing.T) {
	t.Cleanup(version.DisableCacheForTest(123))
	ctx := t.Context()

	t.Run("Create and update works with minimal setup", func(t *testing.T) {
		dk := createDynakube(true)

		mockK8sClient := fake.NewClient()

		reconciler := NewReconciler(mockK8sClient, mockK8sClient)
		err := reconciler.Reconcile(ctx, imageclientmock.NewClient(t), dk)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
		oldTransitionTime := condition.LastTransitionTime
		require.NotNil(t, condition)
		require.NotEmpty(t, oldTransitionTime)
		assert.Equal(t, k8sconditions.DaemonSetSetCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)

		err = reconciler.Reconcile(t.Context(), imageclientmock.NewClient(t), dk)
		require.NoError(t, err)

		var daemonset appsv1.DaemonSet
		err = mockK8sClient.Get(ctx, types.NamespacedName{
			Name:      dk.KSPM().GetDaemonSetName(),
			Namespace: dk.Namespace,
		}, &daemonset)
		require.False(t, k8serrors.IsNotFound(err))
		assert.NotEmpty(t, daemonset)
		assert.Contains(t, daemonset.Annotations, hasher.AnnotationHash)
	})
	t.Run("Only runs when required, and cleans up condition + daemonset", func(t *testing.T) {
		dk := createDynakube(false)

		previousDaemonSet := appsv1.DaemonSet{}
		previousDaemonSet.Name = dk.KSPM().GetDaemonSetName()
		previousDaemonSet.Namespace = dk.Namespace
		mockK8sClient := fake.NewClient(&previousDaemonSet)

		k8sconditions.SetDaemonSetCreated(dk.Conditions(), conditionType, "this is a test")

		reconciler := NewReconciler(mockK8sClient, mockK8sClient)
		err := reconciler.Reconcile(ctx, imageclientmock.NewClient(t), dk)

		require.NoError(t, err)
		assert.Empty(t, *dk.Conditions())

		var daemonset appsv1.DaemonSet
		err = mockK8sClient.Get(ctx, types.NamespacedName{
			Name:      dk.KSPM().GetDaemonSetName(),
			Namespace: dk.Namespace,
		}, &daemonset)
		require.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("problem with k8s request => visible in conditions", func(t *testing.T) {
		dk := createDynakube(true)

		boomClient := createBOOMK8sClient(t)

		reconciler := NewReconciler(boomClient, boomClient)

		err := reconciler.Reconcile(t.Context(), imageclientmock.NewClient(t), dk)

		require.Error(t, err)
		require.Len(t, *dk.Conditions(), 1)
		condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
		assert.Equal(t, k8sconditions.KubeAPIErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func TestImageResolution(t *testing.T) {
	t.Cleanup(version.DisableCacheForTest(123))

	ctx := t.Context()
	anyCtx := mock.MatchedBy(func(context.Context) bool { return true })

	reconcileDaemonSet := func(t *testing.T, dk *dynakube.DynaKube, imageClient *imageclientmock.Client) (appsv1.DaemonSet, error) {
		t.Helper()

		mockK8sClient := fake.NewClient()

		err := NewReconciler(mockK8sClient, mockK8sClient).Reconcile(ctx, imageClient, dk)
		if err != nil {
			return appsv1.DaemonSet{}, err
		}

		var daemonset appsv1.DaemonSet

		err = mockK8sClient.Get(ctx, types.NamespacedName{Name: dk.KSPM().GetDaemonSetName(), Namespace: dk.Namespace}, &daemonset)
		require.NoError(t, err)

		return daemonset, nil
	}

	t.Run("custom image-ref is used as-is, fleet management is not called", func(t *testing.T) {
		dk := createDynakube(true)

		// a mock without expectations fails the test if fleet management gets called
		daemonset, err := reconcileDaemonSet(t, dk, imageclientmock.NewClient(t))
		require.NoError(t, err)

		assert.Equal(t, testImageRepo+":"+testImageTag, dk.Status.KSPM.ResolvedImage)
		assert.Equal(t, testImageRepo+":"+testImageTag, daemonset.Spec.Template.Spec.Containers[0].Image)
	})

	t.Run("no image-ref takes the image from fleet management using the default registry", func(t *testing.T) {
		dk := createDynakube(true)
		dk.Spec.Templates.KSPMNodeConfigurationCollector.ImageRef = sharedimage.Ref{}

		imageClient := imageclientmock.NewClient(t)
		imageClient.EXPECT().GetComponentLatestInfo(anyCtx, dtimage.NCC, "").
			Return(&dtimage.Info{URI: testFleetMgmtImageURI}, nil).Once()

		daemonset, err := reconcileDaemonSet(t, dk, imageClient)
		require.NoError(t, err)

		assert.Equal(t, testFleetMgmtImageURI, dk.Status.KSPM.ResolvedImage)
		assert.Equal(t, testFleetMgmtImageURI, daemonset.Spec.Template.Spec.Containers[0].Image)
	})

	t.Run("no image-ref takes the image from fleet management using the override registry", func(t *testing.T) {
		dk := createDynakube(true)
		dk.Spec.Templates.KSPMNodeConfigurationCollector.ImageRef = sharedimage.Ref{}
		dk.Spec.PublicRegistryOverride = testRegistryOverride

		imageClient := imageclientmock.NewClient(t)
		imageClient.EXPECT().GetComponentLatestInfo(anyCtx, dtimage.NCC, testRegistryOverride).
			Return(&dtimage.Info{URI: testFleetMgmtImageURI}, nil).Once()

		daemonset, err := reconcileDaemonSet(t, dk, imageClient)
		require.NoError(t, err)

		assert.Equal(t, testFleetMgmtImageURI, daemonset.Spec.Template.Spec.Containers[0].Image)
	})

	t.Run("failing image resolution is propagated and leaves the status empty", func(t *testing.T) {
		dk := createDynakube(true)
		dk.Spec.Templates.KSPMNodeConfigurationCollector.ImageRef = sharedimage.Ref{}

		expectedErr := errors.New("fleet management is unreachable")

		imageClient := imageclientmock.NewClient(t)
		imageClient.EXPECT().GetComponentLatestInfo(anyCtx, dtimage.NCC, "").Return(nil, expectedErr).Once()

		_, err := reconcileDaemonSet(t, dk, imageClient)
		require.ErrorIs(t, err, expectedErr)

		assert.Empty(t, dk.Status.KSPM.ResolvedImage)
	})
}

func TestGenerateDaemonSet(t *testing.T) {
	t.Cleanup(version.DisableCacheForTest(123))

	t.Run("generate daemonset", func(t *testing.T) {
		t.Setenv(k8senv.DTOperatorPullSecretEnvName, "")

		dk := createDynakube(true)

		reconciler := NewReconciler(nil, nil)
		daemonset, err := reconciler.generateDaemonSet(dk, "")
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Len(t, daemonset.Spec.Template.Spec.Containers, 1)
		assert.NotEmpty(t, daemonset.Spec.Template.Spec.Volumes)
		assert.Equal(t, dk.KSPM().GetDaemonSetName(), daemonset.Name)
		assert.Equal(t, dk.Namespace, daemonset.Namespace)
		assert.NotEmpty(t, daemonset.Labels)
		assert.NotEmpty(t, daemonset.Spec.Template.Labels)
		assert.NotEmpty(t, daemonset.Spec.Template.Spec.Affinity)
		assert.Subset(t, daemonset.Spec.Template.Labels, daemonset.Spec.Selector.MatchLabels)
		require.Empty(t, daemonset.Annotations)
		require.Len(t, daemonset.Spec.Template.Annotations, 2)
		assert.Contains(t, daemonset.Spec.Template.Annotations, tokenSecretHashAnnotation)
		assert.Equal(t, serviceAccountName, daemonset.Spec.Template.Spec.ServiceAccountName)
		assert.Empty(t, daemonset.Spec.Template.Spec.DNSPolicy)
		assert.Empty(t, daemonset.Spec.Template.Spec.PriorityClassName)
		assert.Empty(t, daemonset.Spec.Template.Spec.Tolerations)
		// KSPM does not pull from the tenant registry, so the operator-generated pull secret must not be mounted
		assert.Empty(t, daemonset.Spec.Template.Spec.ImagePullSecrets)
		require.NotNil(t, daemonset.Spec.UpdateStrategy.RollingUpdate)
		assert.Equal(t, *getDefaultMaxUnavailable(), *daemonset.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable)
		assert.True(t, daemonset.Spec.Template.Spec.HostPID)
		require.NotNil(t, daemonset.Spec.Template.Spec.AutomountServiceAccountToken)
		assert.False(t, *daemonset.Spec.Template.Spec.AutomountServiceAccountToken)
	})

	t.Run("respect custom labels", func(t *testing.T) {
		customLabels := map[string]string{
			"custom": "label",
		}

		dk := createDynakube(true)
		dk.KSPM().Labels = customLabels

		reconciler := NewReconciler(nil, nil)
		daemonset, err := reconciler.generateDaemonSet(dk, "")
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Subset(t, daemonset.Spec.Template.Labels, customLabels)
	})

	t.Run("respect custom annotations", func(t *testing.T) {
		customAnnotations := map[string]string{
			"custom": "annotation",
		}

		dk := createDynakube(true)
		dk.KSPM().Annotations = customAnnotations

		reconciler := NewReconciler(nil, nil)
		daemonset, err := reconciler.generateDaemonSet(dk, "")
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Subset(t, daemonset.Annotations, customAnnotations)
		assert.Subset(t, daemonset.Spec.Template.Annotations, customAnnotations)
	})

	t.Run("respect priority class", func(t *testing.T) {
		customClass := "custom-class"

		dk := createDynakube(true)
		dk.KSPM().PriorityClassName = customClass

		reconciler := NewReconciler(nil, nil)
		daemonset, err := reconciler.generateDaemonSet(dk, "")
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Equal(t, customClass, daemonset.Spec.Template.Spec.PriorityClassName)
	})

	t.Run("respect custom pull-secret", func(t *testing.T) {
		customPullSecret := "custom-pull-secret"

		dk := createDynakube(true)
		dk.Spec.CustomPullSecret = customPullSecret

		reconciler := NewReconciler(nil, nil)
		daemonset, err := reconciler.generateDaemonSet(dk, "")
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Contains(t, daemonset.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: customPullSecret})
		// KSPM must not receive the operator-generated tenant registry pull secret
		assert.NotContains(t, daemonset.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: dk.TenantRegistryPullSecretName()})
	})

	t.Run("respect custom tolerations", func(t *testing.T) {
		customTolerations := []corev1.Toleration{
			{
				Key:      "toleration-key",
				Operator: "toleration-operator",
				Value:    "toleration-value",
			},
		}

		dk := createDynakube(true)
		dk.KSPM().Tolerations = customTolerations
		reconciler := NewReconciler(nil, nil)
		daemonset, err := reconciler.generateDaemonSet(dk, "")
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Equal(t, daemonset.Spec.Template.Spec.Tolerations, customTolerations)
	})
	t.Run("respect custom nodeSelector", func(t *testing.T) {
		customNodeSelector := map[string]string{
			"some.nodeSelector.key": "true",
		}

		dk := createDynakube(true)
		dk.KSPM().NodeSelector = customNodeSelector
		reconciler := NewReconciler(nil, nil)
		daemonset, err := reconciler.generateDaemonSet(dk, "")
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Equal(t, daemonset.Spec.Template.Spec.NodeSelector, customNodeSelector)
	})

	t.Run("respect custom nodeAffinity", func(t *testing.T) {
		customNodeAffinity := &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{{MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "example", Values: []string{"value1"}}}}},
			},
		}

		dk := createDynakube(true)
		dk.KSPM().NodeAffinity = customNodeAffinity
		reconciler := NewReconciler(nil, nil)
		daemonset, err := reconciler.generateDaemonSet(dk, "")
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Equal(t, daemonset.Spec.Template.Spec.Affinity.NodeAffinity, customNodeAffinity)
	})
}

func TestAppArmorAnnotationHandling(t *testing.T) {
	const appArmorAnnotationKey = corev1.DeprecatedAppArmorBetaContainerAnnotationKeyPrefix + containerName

	getDaemonSet := func(t *testing.T) *appsv1.DaemonSet {
		t.Helper()

		dk := createDynakube(true)
		dk.Spec.Templates.KSPMNodeConfigurationCollector.Annotations = map[string]string{appArmorAnnotationKey: corev1.DeprecatedAppArmorBetaProfileRuntimeDefault}

		ds, err := NewReconciler(nil, nil).generateDaemonSet(dk, "")
		require.NoError(t, err)

		return ds
	}

	t.Run("apparmor annotation present in 1.30", func(t *testing.T) {
		t.Cleanup(version.DisableCacheForTest(30))

		sts := getDaemonSet(t)
		require.Len(t, sts.Spec.Template.Spec.Containers, 1)
		require.NotNil(t, sts.Spec.Template.Spec.Containers[0].SecurityContext)
		assert.Nil(t, sts.Spec.Template.Spec.Containers[0].SecurityContext.AppArmorProfile)
		assert.Contains(t, sts.Spec.Template.Annotations, appArmorAnnotationKey)
	})

	t.Run("apparmor annotation absent in 1.31", func(t *testing.T) {
		t.Cleanup(version.DisableCacheForTest(31))

		sts := getDaemonSet(t)
		require.Len(t, sts.Spec.Template.Spec.Containers, 1)
		require.NotNil(t, sts.Spec.Template.Spec.Containers[0].SecurityContext)
		assert.NotNil(t, sts.Spec.Template.Spec.Containers[0].SecurityContext.AppArmorProfile)
		assert.NotContains(t, sts.Spec.Template.Annotations, appArmorAnnotationKey)
	})
}

func TestTlsSecretHashAnnotationHandling(t *testing.T) {
	t.Run("no AG TLS secret", func(t *testing.T) {
		dk := createDynakube(true)
		dk.Annotations = map[string]string{
			exp.AGAutomaticTLSCertificateKey: "false",
		}

		kubeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*corev1.Secret); ok {
					return errors.New("secret should not be read")
				}

				return c.Get(ctx, key, obj, opts...)
			},
		})

		reconciler := NewReconciler(kubeClient, nil)
		hash, err := reconciler.getTLSSecretHash(t.Context(), dk)
		require.NoError(t, err)
		assert.Empty(t, hash)
	})

	t.Run("generic AG TLS secret used", func(t *testing.T) {
		dk := createDynakube(true)

		kubeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*corev1.Secret); ok {
					if key.Name != dk.ActiveGate().GetTLSSecretName() {
						return errors.New("wrong secret is read")
					}
				}

				return c.Get(ctx, key, obj, opts...)
			},
		}, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.ActiveGate().GetTLSSecretName(),
				Namespace: dk.Namespace,
			},
			Data: map[string][]byte{
				"tls.key": []byte("foo"),
			},
		})
		reconciler := NewReconciler(kubeClient, nil)
		hash, err := reconciler.getTLSSecretHash(t.Context(), dk)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("kubemon AG preferred", func(t *testing.T) {
		t.Setenv(k8senv.ExperimentalEnableKubemonOperand, "true")

		dk := createDynakube(true)
		dk.Spec.KubernetesMonitoring = &kubemon.Spec{}

		kubeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*corev1.Secret); ok {
					if key.Name != dk.KubernetesMonitoring().GetTLSSecretName() {
						return errors.New("wrong secret is read")
					}
				}

				return c.Get(ctx, key, obj, opts...)
			},
		}, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.KubernetesMonitoring().GetTLSSecretName(),
				Namespace: dk.Namespace,
			},
			Data: map[string][]byte{
				"tls.key": []byte("foo"),
			},
		})
		reconciler := NewReconciler(kubeClient, nil)
		hash, err := reconciler.getTLSSecretHash(t.Context(), dk)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})
}

func createDynakube(isEnabled bool) *dynakube.DynaKube {
	var kspmSpec *kspm.Spec
	if isEnabled {
		kspmSpec = &kspm.Spec{}
	}

	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dkNamespace,
			Name:      dkName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "test-url",
			KSPM:   kspmSpec,
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.KubeMonCapability.DisplayName,
				},
			},
			Templates: dynakube.TemplatesSpec{
				KSPMNodeConfigurationCollector: kspm.NodeConfigurationCollectorSpec{
					ImageRef: sharedimage.Ref{Repository: testImageRepo, Tag: testImageTag},
				},
			},
		},
		Status: dynakube.DynaKubeStatus{
			ActiveGate: activegate.Status{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: "test-tenant",
				},
			},
			KSPM: kspm.Status{
				TokenSecretHash: "some-hash",
			},
		},
	}
}

func createBOOMK8sClient(t *testing.T) client.Client {
	t.Helper()

	boomClient := fake.NewClientWithInterceptors(interceptor.Funcs{
		Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
			return errors.New("BOOM")
		},
		Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			return errors.New("BOOM")
		},
		Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			return errors.New("BOOM")
		},
		Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			return errors.New("BOOM")
		},
	})

	return boomClient
}
