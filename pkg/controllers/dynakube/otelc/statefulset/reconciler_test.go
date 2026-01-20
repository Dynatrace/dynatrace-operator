package statefulset

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	otelcconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/topology"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testDynakubeName          = "dynakube"
	testNamespaceName         = "dynatrace"
	testOtelPullSecret        = "otelc-pull-secret"
	testTelemetryIngestSecret = "test-ts-secret"
	testKubeSystemUUID        = "123e4567-e89b-12d3-a456-426614174000"
	testKubernetesClusterName = "test-cluster"
	testKubernetesClusterMEID = "12345678901234567890"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("Create and update works with minimal setup", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()

		mockK8sClient := fake.NewClient()
		mockK8sClient = mockTLSSecret(t, mockK8sClient, dk)

		reconciler := NewReconciler(mockK8sClient,
			mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), conditionType)
		oldTransitionTime := condition.LastTransitionTime
		require.NotNil(t, condition)
		require.NotEmpty(t, oldTransitionTime)
		assert.Equal(t, conditions.StatefulSetCreatedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)

		err = reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		var sts appsv1.StatefulSet
		err = mockK8sClient.Get(ctx, types.NamespacedName{
			Name:      dk.OtelCollectorStatefulsetName(),
			Namespace: dk.Namespace,
		}, &sts)
		require.False(t, k8serrors.IsNotFound(err))
		assert.NotEmpty(t, sts)
	})
	t.Run("Only runs when required, and cleans up condition + statefulset", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.Extensions = nil

		previousSts := appsv1.StatefulSet{}
		previousSts.Name = dk.OtelCollectorStatefulsetName()
		previousSts.Namespace = dk.Namespace
		mockK8sClient := fake.NewClient(&previousSts)
		mockK8sClient = mockTLSSecret(t, mockK8sClient, dk)

		conditions.SetStatefulSetCreated(dk.Conditions(), conditionType, "this is a test")

		reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)
		err := reconciler.Reconcile(ctx)

		require.NoError(t, err)
		assert.Empty(t, *dk.Conditions())

		var sts appsv1.StatefulSet
		err = mockK8sClient.Get(ctx, types.NamespacedName{
			Name:      dk.OtelCollectorStatefulsetName(),
			Namespace: dk.Namespace,
		}, &sts)
		require.True(t, k8serrors.IsNotFound(err))
	})
}

func TestSecretHashAnnotation(t *testing.T) {
	t.Run("annotation is set with self-signed tls secret", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = ""
		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.Template.Annotations, 1)
		assert.NotEmpty(t, statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash])
	})
	t.Run("annotation is set with tlsRefName", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = "dummy-secret"
		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.Template.Annotations, 1)
		assert.NotEmpty(t, statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash])
	})
	t.Run("annotation is updated when TLS Secret gets updated", func(t *testing.T) {
		statefulSet := &appsv1.StatefulSet{}
		dk := getTestDynakubeWithExtensions()

		// first reconcile a basic setup - TLS Secret gets created
		mockK8sClient := fake.NewClient(dk)
		mockK8sClient = mockTLSSecret(t, mockK8sClient, dk)

		reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)
		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.OtelCollectorStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.NoError(t, err)

		originalSecretHash := statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash]

		// then update the TLS Secret and call reconcile again
		updatedTLSSecret := getTLSSecret(dk.Extensions().GetTLSSecretName(), dk.Namespace, "updated-cert", "updated-key")
		err = mockK8sClient.Update(context.Background(), &updatedTLSSecret)
		require.NoError(t, err)

		err = reconciler.Reconcile(context.Background())
		require.NoError(t, err)
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.OtelCollectorStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.NoError(t, err)

		resultingSecretHash := statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash]

		// original hash and resulting hash should be different, value got updated on reconcile
		assert.NotEqual(t, originalSecretHash, resultingSecretHash)
	})
}

func TestStatefulsetBase(t *testing.T) {
	t.Run("replicas", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakubeWithExtensions())

		assert.Equal(t, int32(1), *statefulSet.Spec.Replicas)
	})

	t.Run("pod management policy", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakubeWithExtensions())

		assert.Equal(t, appsv1.ParallelPodManagement, statefulSet.Spec.PodManagementPolicy)
	})
}

func TestServiceAccountName(t *testing.T) {
	t.Run("serviceAccountName is set", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakubeWithExtensions())

		assert.Equal(t, serviceAccountName, statefulSet.Spec.Template.Spec.ServiceAccountName)
		assert.Equal(t, serviceAccountName, statefulSet.Spec.Template.Spec.DeprecatedServiceAccount)
	})
}

func TestTopologySpreadConstraints(t *testing.T) {
	t.Run("the default TopologySpreadConstraints", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		statefulSet := getStatefulset(t, dk)
		appLabels := buildAppLabels(dk.Name)
		assert.Equal(t, topology.MaxOnePerNode(appLabels), statefulSet.Spec.Template.Spec.TopologySpreadConstraints)
	})

	t.Run("custom TopologySpreadConstraints", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()

		customTopologySpreadConstraints := []corev1.TopologySpreadConstraint{
			{
				MaxSkew:           2,
				TopologyKey:       "kubernetes.io/hostname",
				WhenUnsatisfiable: "DoNotSchedule",
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"a": "b",
					},
				},
			},
		}

		dk.Spec.Templates.OpenTelemetryCollector.TopologySpreadConstraints = customTopologySpreadConstraints

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, customTopologySpreadConstraints, statefulSet.Spec.Template.Spec.TopologySpreadConstraints)
	})
}

func TestAffinity(t *testing.T) {
	t.Run("affinity", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		statefulSet := getStatefulset(t, dk)

		expectedAffinity := node.Affinity()

		assert.Equal(t, expectedAffinity, *statefulSet.Spec.Template.Spec.Affinity)
	})
}

func TestImagePullSecrets(t *testing.T) {
	t.Run("the default image pull secret only", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakubeWithExtensions())

		assert.Len(t, statefulSet.Spec.Template.Spec.ImagePullSecrets, 1)
	})

	t.Run("custom pull secret", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.CustomPullSecret = testOtelPullSecret

		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.Template.Spec.ImagePullSecrets, 2)
		assert.Equal(t, dk.Name+dynakube.PullSecretSuffix, statefulSet.Spec.Template.Spec.ImagePullSecrets[0].Name)
		assert.Equal(t, dk.Spec.CustomPullSecret, statefulSet.Spec.Template.Spec.ImagePullSecrets[1].Name)
	})
}

func TestResources(t *testing.T) {
	t.Run("no resources", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		statefulSet := getStatefulset(t, dk)

		assert.Empty(t, statefulSet.Spec.Template.Spec.Containers[0].Resources)
	})

	t.Run("custom resources", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		dk.Spec.Templates.ExtensionExecutionController.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		}

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, dk.Spec.Templates.OpenTelemetryCollector.Resources, statefulSet.Spec.Template.Spec.Containers[0].Resources)
	})
}

func TestLabels(t *testing.T) {
	t.Run("the default labels", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()

		statefulSet := getStatefulset(t, dk)

		appLabels := buildAppLabels(dk.Name)

		assert.Equal(t, appLabels.BuildLabels(), statefulSet.Labels)
		assert.Equal(t, &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}, statefulSet.Spec.Selector)
		assert.Equal(t, appLabels.BuildLabels(), statefulSet.Spec.Template.Labels)
	})

	t.Run("custom labels", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		customLabels := map[string]string{
			"a": "b",
		}
		dk.Spec.Templates.OpenTelemetryCollector.Labels = customLabels

		statefulSet := getStatefulset(t, dk)

		appLabels := buildAppLabels(dk.Name)
		podLabels := maputils.MergeMap(customLabels, appLabels.BuildLabels())

		assert.Equal(t, appLabels.BuildLabels(), statefulSet.Labels)
		assert.Equal(t, &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}, statefulSet.Spec.Selector)
		assert.Equal(t, podLabels, statefulSet.Spec.Template.Labels)
	})
}

func TestAnnotations(t *testing.T) {
	t.Run("the default annotations", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakubeWithExtensions())

		assert.Len(t, statefulSet.Annotations, 1)
		require.Len(t, statefulSet.Spec.Template.Annotations, 1)
		assert.NotEmpty(t, statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash])
	})

	t.Run("custom annotations", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()
		customAnnotations := map[string]string{
			"a": "b",
		}
		dk.Spec.Templates.OpenTelemetryCollector.Annotations = customAnnotations

		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Annotations, 1)
		assert.Empty(t, statefulSet.Annotations["a"])
		require.Len(t, statefulSet.Spec.Template.Annotations, 2)
		assert.Equal(t, "b", statefulSet.Spec.Template.Annotations["a"])
		assert.NotEmpty(t, statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash])
	})
}

func TestTolerations(t *testing.T) {
	t.Run("the default tolerations", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakubeWithExtensions())

		assert.Empty(t, statefulSet.Spec.Template.Spec.Tolerations)
	})

	t.Run("custom tolerations", func(t *testing.T) {
		dk := getTestDynakubeWithExtensions()

		customTolerations := []corev1.Toleration{
			{
				Key:      "a",
				Operator: corev1.TolerationOpEqual,
				Value:    "b",
				Effect:   corev1.TaintEffectNoSchedule,
			},
		}
		dk.Spec.Templates.OpenTelemetryCollector.Tolerations = customTolerations

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, customTolerations, statefulSet.Spec.Template.Spec.Tolerations)
	})
}

func TestSecurityContext(t *testing.T) {
	t.Run("the default securityContext is set", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakubeWithExtensions())

		assert.NotNil(t, statefulSet.Spec.Template.Spec.SecurityContext)
		assert.NotNil(t, statefulSet.Spec.Template.Spec.Containers[0].SecurityContext)
	})
}

func TestUpdateStrategy(t *testing.T) {
	t.Run("the default update strategy is set", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakubeWithExtensions())

		assert.NotNil(t, statefulSet.Spec.UpdateStrategy.RollingUpdate.Partition)
		assert.NotEmpty(t, statefulSet.Spec.UpdateStrategy.Type)
	})
}

func getTestDynakubeWithExtensions() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDynakubeName,
			Namespace:   testNamespaceName,
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{
			Extensions: &extensions.Spec{},
			Templates:  dynakube.TemplatesSpec{OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{}},
		},
		Status: dynakube.DynaKubeStatus{
			KubeSystemUUID:        testKubeSystemUUID,
			KubernetesClusterMEID: testKubernetesClusterMEID,
			KubernetesClusterName: testKubernetesClusterName,
		},
	}
}

func getTestDynakube() *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDynakubeName,
			Namespace:   testNamespaceName,
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{},
		Status: dynakube.DynaKubeStatus{
			KubeSystemUUID:        testKubeSystemUUID,
			KubernetesClusterMEID: testKubernetesClusterMEID,
			KubernetesClusterName: testKubernetesClusterName,
		},
	}

	return dk
}

func getStatefulset(t *testing.T, dk *dynakube.DynaKube, objs ...client.Object) *appsv1.StatefulSet {
	mockK8sClient := fake.NewClient(dk)
	mockK8sClient = mockTLSSecret(t, mockK8sClient, dk)

	for _, obj := range objs {
		err := mockK8sClient.Create(context.Background(), obj)
		require.NoError(t, err)
	}

	err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
	require.NoError(t, err)

	statefulSet := &appsv1.StatefulSet{}
	err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.OtelCollectorStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
	require.NoError(t, err)

	return statefulSet
}

func mockTLSSecret(t *testing.T, client client.Client, dk *dynakube.DynaKube) client.Client {
	tlsSecret := getTLSSecret(dk.Extensions().GetTLSSecretName(), dk.Namespace, "super-cert", "super-key")

	err := client.Create(context.Background(), &tlsSecret)
	require.NoError(t, err)

	return client
}

func getTokens(name string, namespace string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			dtclient.APIToken:        []byte("test"),
			dtclient.DataIngestToken: []byte("test"),
		},
	}
}

func getTLSSecret(name string, namespace string, crt string, key string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			consts.TLSCrtDataName: []byte(crt),
			consts.TLSKeyDataName: []byte(key),
		},
	}
}

func getConfigConfigMap(name string, namespace string) corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + otelcconsts.TelemetryCollectorConfigmapSuffix,
			Namespace: namespace,
		},
		Data: map[string]string{
			otelcconsts.ConfigFieldName: "test",
		},
	}
}
