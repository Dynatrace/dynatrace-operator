package daemonset

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("Create and update works with minimal setup", func(t *testing.T) {
		dk := createDynakube(true)

		mockK8sClient := fake.NewClient()

		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
		oldTransitionTime := condition.LastTransitionTime
		require.NotNil(t, condition)
		require.NotEmpty(t, oldTransitionTime)
		assert.Equal(t, conditions.DaemonSetSetCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)

		err = reconciler.Reconcile(context.Background())
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

		conditions.SetDaemonSetCreated(dk.Conditions(), conditionType, "this is a test")

		reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)

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

		boomClient := createBOOMK8sClient()

		reconciler := NewReconciler(boomClient,
			boomClient, dk)

		err := reconciler.Reconcile(context.Background())

		require.Error(t, err)
		require.Len(t, *dk.Conditions(), 1)
		condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
		assert.Equal(t, conditions.KubeAPIErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func TestGenerateDaemonSet(t *testing.T) {
	t.Run("generate daemonset", func(t *testing.T) {
		dk := createDynakube(true)

		reconciler := NewReconciler(nil,
			nil, dk)
		daemonset, err := reconciler.generateDaemonSet()
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
		require.Len(t, daemonset.Spec.Template.Annotations, 1)
		assert.Contains(t, daemonset.Spec.Template.Annotations, tokenSecretHashAnnotation)
		assert.Equal(t, serviceAccountName, daemonset.Spec.Template.Spec.ServiceAccountName)
		assert.Empty(t, daemonset.Spec.Template.Spec.DNSPolicy)
		assert.Empty(t, daemonset.Spec.Template.Spec.PriorityClassName)
		assert.Empty(t, daemonset.Spec.Template.Spec.Tolerations)
		assert.Len(t, daemonset.Spec.Template.Spec.ImagePullSecrets, 1)
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

		reconciler := NewReconciler(nil,
			nil, dk)
		daemonset, err := reconciler.generateDaemonSet()
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

		reconciler := NewReconciler(nil,
			nil, dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Subset(t, daemonset.Annotations, customAnnotations)
		assert.Subset(t, daemonset.Spec.Template.Annotations, customAnnotations)
	})

	t.Run("respect priority class", func(t *testing.T) {
		customClass := "custom-class"

		dk := createDynakube(true)
		dk.KSPM().PriorityClassName = customClass

		reconciler := NewReconciler(nil,
			nil, dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Equal(t, customClass, daemonset.Spec.Template.Spec.PriorityClassName)
	})

	t.Run("respect custom pull-secret", func(t *testing.T) {
		customPullSecret := "custom-pull-secret"

		dk := createDynakube(true)
		dk.Spec.CustomPullSecret = customPullSecret

		reconciler := NewReconciler(nil,
			nil, dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Contains(t, daemonset.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: customPullSecret})
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
		reconciler := NewReconciler(nil,
			nil, dk)
		daemonset, err := reconciler.generateDaemonSet()
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
		reconciler := NewReconciler(nil,
			nil, dk)
		daemonset, err := reconciler.generateDaemonSet()
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
		reconciler := NewReconciler(nil,
			nil, dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Equal(t, daemonset.Spec.Template.Spec.Affinity.NodeAffinity, customNodeAffinity)
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
			Kspm:   kspmSpec,
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.KubeMonCapability.DisplayName,
				},
			},
		},
		Status: dynakube.DynaKubeStatus{
			ActiveGate: activegate.Status{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: "test-tenant",
				},
			},
			Kspm: kspm.Status{
				TokenSecretHash: "some-hash",
			},
		},
	}
}

func createBOOMK8sClient() client.Client {
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
