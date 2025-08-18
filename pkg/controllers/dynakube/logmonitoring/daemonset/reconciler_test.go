package daemonset

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
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
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	dkName      = "test-name"
	dkNamespace = "test-namespace"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("Only clean up if not standalone", func(t *testing.T) {
		dk := createDynakube(true)
		dk.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		conditions.SetDaemonSetCreated(dk.Conditions(), ConditionType, "testing")

		previousDaemonSet := appsv1.DaemonSet{}
		previousDaemonSet.Name = dk.LogMonitoring().GetDaemonSetName()
		previousDaemonSet.Namespace = dk.Namespace
		mockK8sClient := fake.NewClient(&previousDaemonSet)

		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		var daemonset appsv1.DaemonSet
		err = mockK8sClient.Get(ctx, types.NamespacedName{
			Name:      dk.LogMonitoring().GetDaemonSetName(),
			Namespace: dk.Namespace,
		}, &daemonset)
		require.True(t, k8serrors.IsNotFound(err))

		condition := meta.FindStatusCondition(*dk.Conditions(), ConditionType)
		require.Nil(t, condition)
	})

	t.Run("Create and update works with minimal setup", func(t *testing.T) {
		dk := createDynakube(true)

		mockK8sClient := fake.NewClient()

		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), ConditionType)
		require.NotNil(t, condition)
		oldTransitionTime := condition.LastTransitionTime
		require.NotEmpty(t, oldTransitionTime)
		assert.Equal(t, conditions.DaemonSetSetCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)

		err = reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var daemonset appsv1.DaemonSet
		err = mockK8sClient.Get(ctx, types.NamespacedName{
			Name:      dk.LogMonitoring().GetDaemonSetName(),
			Namespace: dk.Namespace,
		}, &daemonset)
		require.False(t, k8serrors.IsNotFound(err))
		assert.NotEmpty(t, daemonset)
	})

	t.Run("Only runs when required, and cleans up condition + secret", func(t *testing.T) {
		dk := createDynakube(false)

		previousDaemonSet := appsv1.DaemonSet{}
		previousDaemonSet.Name = dk.LogMonitoring().GetDaemonSetName()
		previousDaemonSet.Namespace = dk.Namespace
		mockK8sClient := fake.NewClient(&previousDaemonSet)

		conditions.SetDaemonSetCreated(dk.Conditions(), ConditionType, "this is a test")

		reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.Empty(t, *dk.Conditions())

		var daemonset appsv1.DaemonSet
		err = mockK8sClient.Get(ctx, types.NamespacedName{
			Name:      dk.LogMonitoring().GetDaemonSetName(),
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
		condition := meta.FindStatusCondition(*dk.Conditions(), ConditionType)
		assert.Equal(t, conditions.KubeAPIErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})

	t.Run("requeues when all scopes are there and no clusterMEID set", func(t *testing.T) {
		dk := createDynakube(true)
		dk.Status.KubernetesClusterMEID = ""

		mockK8sClient := fake.NewClient()

		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, dk)

		err := reconciler.Reconcile(context.Background())

		require.Error(t, err)
		require.EqualError(t, err, KubernetesSettingsNotAvailableError.Error())
	})
}

func TestGenerateDaemonSet(t *testing.T) {
	t.Run("generate daemonset", func(t *testing.T) {
		dk := createDynakube(true)

		reconciler := NewReconciler(nil, fake.NewClient(), dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Len(t, daemonset.Spec.Template.Spec.InitContainers, 1)
		assert.Len(t, daemonset.Spec.Template.Spec.Containers, 1)
		assert.NotEmpty(t, daemonset.Spec.Template.Spec.Volumes)
		assert.Equal(t, dk.LogMonitoring().GetDaemonSetName(), daemonset.Name)
		assert.Equal(t, dk.Namespace, daemonset.Namespace)
		assert.NotEmpty(t, daemonset.Labels)
		assert.NotEmpty(t, daemonset.Spec.Template.Labels)
		assert.NotEmpty(t, daemonset.Spec.Template.Spec.Affinity)
		assert.Subset(t, daemonset.Spec.Template.Labels, daemonset.Spec.Selector.MatchLabels)
		require.Len(t, daemonset.Annotations, 1)
		assert.Contains(t, daemonset.Annotations, hasher.AnnotationHash)
		require.Len(t, daemonset.Spec.Template.Annotations, 1)
		assert.Contains(t, daemonset.Spec.Template.Annotations, annotationTenantTokenHash)
		assert.Equal(t, serviceAccountName, daemonset.Spec.Template.Spec.ServiceAccountName)
		assert.Empty(t, daemonset.Spec.Template.Spec.DNSPolicy)
		assert.Empty(t, daemonset.Spec.Template.Spec.PriorityClassName)
		assert.Empty(t, daemonset.Spec.Template.Spec.Tolerations)
		assert.Len(t, daemonset.Spec.Template.Spec.ImagePullSecrets, 1)
		require.NotNil(t, daemonset.Spec.UpdateStrategy.RollingUpdate)
		assert.Equal(t, intstr.FromInt(dk.FF().GetOneAgentMaxUnavailable()), *daemonset.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable)
	})

	t.Run("respect custom labels", func(t *testing.T) {
		customLabels := map[string]string{
			"custom": "label",
		}

		dk := createDynakube(true)
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			Labels: customLabels,
		}

		reconciler := NewReconciler(nil, fake.NewClient(), dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Subset(t, daemonset.Spec.Template.Labels, customLabels)
	})

	t.Run("respect annotations", func(t *testing.T) {
		customAnnotations := map[string]string{
			"custom": "annotation",
		}
		testTokenHash := "testTokenHash"

		dk := createDynakube(true)
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			Annotations: customAnnotations,
		}
		dk.Status.OneAgent.ConnectionInfoStatus.TenantTokenHash = testTokenHash

		reconciler := NewReconciler(nil, fake.NewClient(), dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Subset(t, daemonset.Spec.Template.Annotations, customAnnotations)
		assert.Equal(t, testTokenHash, daemonset.Spec.Template.Annotations[annotationTenantTokenHash])
	})

	t.Run("respect dns policy", func(t *testing.T) {
		customPolicy := corev1.DNSClusterFirst

		dk := createDynakube(true)
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			DNSPolicy: customPolicy,
		}

		reconciler := NewReconciler(nil, fake.NewClient(), dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Equal(t, customPolicy, daemonset.Spec.Template.Spec.DNSPolicy)
	})

	t.Run("respect priority class", func(t *testing.T) {
		customClass := "custom-class"

		dk := createDynakube(true)
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			PriorityClassName: customClass,
		}

		reconciler := NewReconciler(nil, fake.NewClient(), dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Equal(t, customClass, daemonset.Spec.Template.Spec.PriorityClassName)
	})

	t.Run("respect custom pull-secret", func(t *testing.T) {
		customPullSecret := "custom-pull-secret"

		dk := createDynakube(true)
		dk.Spec.CustomPullSecret = customPullSecret

		reconciler := NewReconciler(nil, fake.NewClient(), dk)
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
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			Tolerations: customTolerations,
		}
		reconciler := NewReconciler(nil, fake.NewClient(), dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Equal(t, daemonset.Spec.Template.Spec.Tolerations, customTolerations)
	})

	t.Run("respect custom nodeselector", func(t *testing.T) {
		customNodeSelector := map[string]string{
			"some.nodeSelector.key": "true",
		}

		dk := createDynakube(true)
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			NodeSelector: customNodeSelector,
		}
		reconciler := NewReconciler(nil, fake.NewClient(), dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		assert.Equal(t, daemonset.Spec.Template.Spec.NodeSelector, customNodeSelector)
	})

	t.Run("generate a daemonset with no kubernetes cluster name set in env and arg section if no MEID and all scopes set", func(t *testing.T) {
		dk := createDynakube(true)
		dk.Status.KubernetesClusterMEID = ""
		dk.Status.KubernetesClusterName = ""

		reconciler := NewReconciler(nil, fake.NewClient(), dk)
		daemonset, err := reconciler.generateDaemonSet()
		require.NoError(t, err)
		require.NotNil(t, daemonset)

		init := daemonset.Spec.Template.Spec.InitContainers[0]
		require.NotContains(t, init.Args, fmt.Sprintf("-p dt.entity.kubernetes_cluster=$(%s)", entityEnv))

		found := false
		for _, e := range init.Env {
			if e.Name == entityEnv && e.Value == dk.Status.KubernetesClusterMEID {
				found = true
			}
		}

		require.False(t, found)
	})

	t.Run("both scopes set, MEID missing - wait, DS not created", func(t *testing.T) {
		dk := createDynakube(true)
		dk.Status.KubernetesClusterMEID = ""
		dk.Status.KubernetesClusterName = ""
		setScopes(dk, true, true)

		mockK8sClient := fake.NewClient()
		reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.Error(t, err)
		require.ErrorContains(t, err, "missing information about the kubernetes monitored-entity")

		var ds appsv1.DaemonSet
		err = mockK8sClient.Get(context.Background(), types.NamespacedName{Name: dk.LogMonitoring().GetDaemonSetName(), Namespace: dk.Namespace}, &ds)
		require.Error(t, err)
		require.ErrorContains(t, err, "not found")
	})

	t.Run("both scopes set AND MEID set, check args and envs of DS", func(t *testing.T) {
		dk := createDynakube(true)
		setScopes(dk, true, true)

		mockK8sClient := fake.NewClient()
		reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)

		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var ds appsv1.DaemonSet

		err = mockK8sClient.Get(context.Background(), types.NamespacedName{Name: dk.LogMonitoring().GetDaemonSetName(), Namespace: dk.Namespace}, &ds)
		require.NoError(t, err)

		initContainer := ds.Spec.Template.Spec.InitContainers[0]

		require.Contains(t, initContainer.Args, fmt.Sprintf("-p dt.entity.kubernetes_cluster=$(%s)", entityEnv))

		found := false
		for _, env := range initContainer.Env {
			if env.Name == entityEnv && env.Value == dk.Status.KubernetesClusterMEID {
				found = true

				break
			}
		}
		require.True(t, found)
	})
}

func createDynakube(isEnabled bool) *dynakube.DynaKube {
	var logMonitoring *logmonitoring.Spec
	if isEnabled {
		logMonitoring = &logmonitoring.Spec{}
	}

	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dkNamespace,
			Name:      dkName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:        "test-url",
			LogMonitoring: logMonitoring,
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: oneagent.Status{
				ConnectionInfoStatus: oneagent.ConnectionInfoStatus{
					ConnectionInfo: communication.ConnectionInfo{
						TenantUUID: "test-uuid",
					},
				},
			},
			KubernetesClusterMEID: "test-cluster-me-id",
			KubernetesClusterName: "test-cluster-name",
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

func setScopes(dk *dynakube.DynaKube, read, write bool) {
	set := func(t string, ok bool) {
		if ok {
			meta.SetStatusCondition(dk.Conditions(), metav1.Condition{Type: t, Status: metav1.ConditionTrue})
		} else {
			meta.RemoveStatusCondition(dk.Conditions(), t)
		}
	}

	set(dtclient.ConditionTypeAPITokenSettingsRead, read)
	set(dtclient.ConditionTypeAPITokenSettingsWrite, write)
}
