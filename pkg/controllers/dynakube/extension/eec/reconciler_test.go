package eec

import (
	"strconv"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testDynakubeName       = "dynakube"
	testNamespaceName      = "dynatrace"
	testEecPullSecret      = "eec-pull-secret"
	testEecImageRepository = "repo/dynatrace-eec"
	testEecImageTag        = "1.289.0"
	testTenantUUID         = "abc12345"
	testKubeSystemUUID     = "12345"
)

func getTestDynakube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDynakubeName,
			Namespace:   testNamespaceName,
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{
			Extensions: dynakube.ExtensionsSpec{
				Prometheus: dynakube.PrometheusSpec{
					Enabled: true,
				},
			},
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
					ImageRef: dynakube.ImageRefSpec{
						Repository: testEecImageRepository,
						Tag:        testEecImageTag,
					},
				},
			},
		},

		Status: dynakube.DynaKubeStatus{
			ActiveGate: dynakube.ActiveGateStatus{
				ConnectionInfoStatus: dynakube.ActiveGateConnectionInfoStatus{
					ConnectionInfoStatus: dynakube.ConnectionInfoStatus{
						TenantUUID: testTenantUUID,
					},
				},
				VersionStatus: status.VersionStatus{},
			},
			KubeSystemUUID: testKubeSystemUUID,
		},
	}
}

func getStatefulset(t *testing.T, dk *dynakube.DynaKube) *appsv1.StatefulSet {
	mockK8sClient := fake.NewClient(dk)

	err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
	require.NoError(t, err)

	statefulSet := &appsv1.StatefulSet{}
	err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: statefulsetName, Namespace: dk.Namespace}, statefulSet)
	require.NoError(t, err)

	return statefulSet
}

func TestConditions(t *testing.T) {
	t.Run("no kubeSystemUUID", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Status.KubeSystemUUID = ""

		mockK8sClient := fake.NewClient(dk)

		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.Error(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: statefulsetName, Namespace: dk.Namespace}, statefulSet)
		require.Error(t, err)

		assert.True(t, errors.IsNotFound(err))
	})

	t.Run("no tenantUUID", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Status.ActiveGate.ConnectionInfoStatus.TenantUUID = ""

		mockK8sClient := fake.NewClient(dk)

		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.Error(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: statefulsetName, Namespace: dk.Namespace}, statefulSet)
		require.Error(t, err)

		assert.True(t, errors.IsNotFound(err))
	})
}

func TestStatefulsetBase(t *testing.T) {
	t.Run("replicas", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.Equal(t, int32(1), *statefulSet.Spec.Replicas)
	})

	t.Run("pod management policy", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.Equal(t, appsv1.ParallelPodManagement, statefulSet.Spec.PodManagementPolicy)
	})
}

func TestTopologySpreadConstraints(t *testing.T) {
	t.Run("the default TopologySpreadConstraints", func(t *testing.T) {
		dk := getTestDynakube()
		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, buildDefaultTopologySpreadConstraints(dk.Name), statefulSet.Spec.Template.Spec.TopologySpreadConstraints)
	})

	t.Run("custom TopologySpreadConstraints", func(t *testing.T) {
		dk := getTestDynakube()

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

		dk.Spec.Templates.ExtensionExecutionController.TopologySpreadConstraints = customTopologySpreadConstraints

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, customTopologySpreadConstraints, statefulSet.Spec.Template.Spec.TopologySpreadConstraints)
	})
}

func TestEnvironmentVariables(t *testing.T) {
	t.Run("environment variables", func(t *testing.T) {
		dk := getTestDynakube()

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, corev1.EnvVar{Name: envTenantId, Value: dk.Status.ActiveGate.ConnectionInfoStatus.TenantUUID}, statefulSet.Spec.Template.Spec.Containers[0].Env[0])
		assert.Equal(t, corev1.EnvVar{Name: envServerUrl, Value: buildActiveGateServiceName(dk) + "." + dk.Namespace + ".svc.cluster.local:443"}, statefulSet.Spec.Template.Spec.Containers[0].Env[1])
		assert.Equal(t, corev1.EnvVar{Name: envEecTokenPath, Value: eecTokenMountPath + "/" + eecFile}, statefulSet.Spec.Template.Spec.Containers[0].Env[2])
		assert.Equal(t, corev1.EnvVar{Name: envEecIngestPort, Value: strconv.Itoa(int(collectorPort))}, statefulSet.Spec.Template.Spec.Containers[0].Env[3])
		assert.Equal(t, corev1.EnvVar{Name: envExtensionsConfPathName, Value: envExtensionsConfPath}, statefulSet.Spec.Template.Spec.Containers[0].Env[4])
		assert.Equal(t, corev1.EnvVar{Name: envExtensionsModuleExecPathName, Value: envExtensionsModuleExecPath}, statefulSet.Spec.Template.Spec.Containers[0].Env[5])
		assert.Equal(t, corev1.EnvVar{Name: envDsInstallDirName, Value: envDsInstallDir}, statefulSet.Spec.Template.Spec.Containers[0].Env[6])
		assert.Equal(t, corev1.EnvVar{Name: envK8sClusterId, Value: dk.Status.KubeSystemUUID}, statefulSet.Spec.Template.Spec.Containers[0].Env[7])
	})
}

func TestVolumes(t *testing.T) {
	t.Run("volume mounts", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		expectedVolumeMounts := []corev1.VolumeMount{
			{
				Name:      tokensVolumeName,
				MountPath: eecTokenMountPath,
				ReadOnly:  true,
			},
			{
				Name:      logVolumeName,
				MountPath: logMountPath,
				ReadOnly:  false,
			},
			{
				Name:      runtimeVolumeName,
				MountPath: runtimeMountPath,
				ReadOnly:  false,
			},
			{
				Name:      configurationVolumeName,
				MountPath: configurationMountPath,
				ReadOnly:  true,
			},
		}
		assert.Equal(t, expectedVolumeMounts, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts)
	})

	t.Run("volumes without PVC", func(t *testing.T) {
		dk := getTestDynakube()

		statefulSet := getStatefulset(t, dk)

		mode := int32(420)
		expectedVolumes := []corev1.Volume{
			{
				Name: tokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.Name + consts.SecretSuffix,
						DefaultMode: &mode,
					},
				},
			},
			{
				Name: logVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: configurationVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: runtimeVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}

		assert.Equal(t, expectedVolumes, statefulSet.Spec.Template.Spec.Volumes)
	})

	t.Run("volumes with PVC", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim = &corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		}

		statefulSet := getStatefulset(t, dk)

		mode := int32(420)
		expectedVolumes := []corev1.Volume{
			{
				Name: tokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.Name + consts.SecretSuffix,
						DefaultMode: &mode,
					},
				},
			},
			{
				Name: logVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: configurationVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: runtimeVolumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: runtimePersistentVolumeClaimName,
					},
				},
			},
		}

		assert.Equal(t, expectedVolumes, statefulSet.Spec.Template.Spec.Volumes)
	})
}

func TestAffinity(t *testing.T) {
	t.Run("affinity", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim = &corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		}

		statefulSet := getStatefulset(t, dk)

		expectedAffinity := &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: node.AffinityNodeRequirementForSupportedArches(),
						},
					},
				},
			},
		}

		assert.Equal(t, expectedAffinity, statefulSet.Spec.Template.Spec.Affinity)
	})
}

func TestImagePullSecrets(t *testing.T) {
	t.Run("the default image pull secret only", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.Len(t, statefulSet.Spec.Template.Spec.ImagePullSecrets, 1)
	})

	t.Run("custom pull secret", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.CustomPullSecret = testEecPullSecret

		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.Template.Spec.ImagePullSecrets, 2)
		assert.Equal(t, dk.Name+dynakube.PullSecretSuffix, statefulSet.Spec.Template.Spec.ImagePullSecrets[0].Name)
		assert.Equal(t, dk.Spec.CustomPullSecret, statefulSet.Spec.Template.Spec.ImagePullSecrets[1].Name)
	})
}

func TestResources(t *testing.T) {
	t.Run("no resources", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.CustomPullSecret = testEecPullSecret

		statefulSet := getStatefulset(t, dk)

		assert.Empty(t, statefulSet.Spec.Template.Spec.Containers[0].Resources)
	})

	t.Run("custom resources", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		}

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, dk.Spec.Templates.ExtensionExecutionController.Resources, statefulSet.Spec.Template.Spec.Containers[0].Resources)
	})
}

func TestLabels(t *testing.T) {
	t.Run("the default labels", func(t *testing.T) {
		dk := getTestDynakube()

		statefulSet := getStatefulset(t, dk)

		appLabels := buildAppLabels(dk.Name)

		assert.Equal(t, appLabels.BuildLabels(), statefulSet.ObjectMeta.Labels)
		assert.Equal(t, &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}, statefulSet.Spec.Selector)
		assert.Equal(t, appLabels.BuildLabels(), statefulSet.Spec.Template.ObjectMeta.Labels)
	})

	t.Run("custom labels", func(t *testing.T) {
		dk := getTestDynakube()
		customLabels := map[string]string{
			"a": "b",
		}
		dk.Spec.Templates.ExtensionExecutionController.Labels = customLabels

		statefulSet := getStatefulset(t, dk)

		appLabels := buildAppLabels(dk.Name)
		podLabels := maputils.MergeMap(customLabels, appLabels.BuildLabels())

		assert.Equal(t, appLabels.BuildLabels(), statefulSet.ObjectMeta.Labels)
		assert.Equal(t, &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}, statefulSet.Spec.Selector)
		assert.Equal(t, podLabels, statefulSet.Spec.Template.ObjectMeta.Labels)
	})
}

func TestAnnotations(t *testing.T) {
	t.Run("the default annotations", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.Len(t, statefulSet.ObjectMeta.Annotations, 1)
		assert.Empty(t, statefulSet.Spec.Template.ObjectMeta.Annotations)
	})

	t.Run("custom annotations", func(t *testing.T) {
		dk := getTestDynakube()
		customAnnotations := map[string]string{
			"a": "b",
		}
		dk.Spec.Templates.ExtensionExecutionController.Annotations = customAnnotations

		statefulSet := getStatefulset(t, dk)

		assert.Len(t, statefulSet.ObjectMeta.Annotations, 1)
		assert.Empty(t, statefulSet.ObjectMeta.Annotations["a"])
		assert.Len(t, statefulSet.Spec.Template.ObjectMeta.Annotations, 1)
		assert.Equal(t, customAnnotations, statefulSet.Spec.Template.ObjectMeta.Annotations)
	})
}

func TestTolerations(t *testing.T) {
	t.Run("the default tolerations", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.Empty(t, statefulSet.Spec.Template.Spec.Tolerations)
	})

	t.Run("custom tolerations", func(t *testing.T) {
		dk := getTestDynakube()

		customTolerations := []corev1.Toleration{
			{
				Key:      "a",
				Operator: corev1.TolerationOpEqual,
				Value:    "b",
				Effect:   corev1.TaintEffectNoSchedule,
			},
		}
		dk.Spec.Templates.ExtensionExecutionController.Tolerations = customTolerations

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, customTolerations, statefulSet.Spec.Template.Spec.Tolerations)
	})
}

func TestPersistentVolumeClaimRetentionPolicy(t *testing.T) {
	t.Run("the default retention policy", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.Nil(t, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy)
	})
	t.Run("custom persistent volume claim retention policy", func(t *testing.T) {
		// TODO: do we want to use statefulset.VolumeClaimTemplates
	})
}

func TestServiceAccountName(t *testing.T) {
	t.Run("serviceAccountName is set", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.Equal(t, serviceAccountName, statefulSet.Spec.Template.Spec.ServiceAccountName)
		assert.Equal(t, serviceAccountName, statefulSet.Spec.Template.Spec.DeprecatedServiceAccount)
	})
}

func TestSecurityContext(t *testing.T) {
	t.Run("the default securityContext is set", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.NotNil(t, statefulSet.Spec.Template.Spec.SecurityContext)
		assert.NotNil(t, statefulSet.Spec.Template.Spec.Containers[0].SecurityContext)
	})
}

func TestUpdateStrategy(t *testing.T) {
	t.Run("the default update strategy is set", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.NotNil(t, statefulSet.Spec.UpdateStrategy.RollingUpdate.Partition)
		assert.NotEmpty(t, statefulSet.Spec.UpdateStrategy.Type)
	})
}
