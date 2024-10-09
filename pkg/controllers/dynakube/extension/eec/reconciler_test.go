package eec

import (
	"strconv"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/tls"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/utils"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
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
	testDynakubeName              = "dynakube"
	testNamespaceName             = "dynatrace"
	testEecPullSecret             = "eec-pull-secret"
	testEecImageRepository        = "repo/dynatrace-eec"
	testEecImageTag               = "1.289.0"
	testTenantUUID                = "abc12345"
	testKubeSystemUUID            = "12345"
	testCustomConfigConfigMapName = "eec-custom-config"
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
				Enabled: true,
			},
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: dynakube.ExtensionExecutionControllerSpec{
					ImageRef: image.Ref{
						Repository: testEecImageRepository,
						Tag:        testEecImageTag,
					},
				},
			},
		},

		Status: dynakube.DynaKubeStatus{
			ActiveGate: activegate.Status{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: testTenantUUID,
				},
				VersionStatus: status.VersionStatus{},
			},
			KubeSystemUUID: testKubeSystemUUID,
		},
	}
}

func getStatefulset(t *testing.T, dk *dynakube.DynaKube) *appsv1.StatefulSet {
	mockK8sClient := fake.NewClient(dk)
	mockK8sClient = mockTLSSecret(t, mockK8sClient, dk)

	err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
	require.NoError(t, err)

	statefulSet := &appsv1.StatefulSet{}
	err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.ExtensionsExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
	require.NoError(t, err)

	return statefulSet
}

func mockTLSSecret(t *testing.T, client client.Client, dk *dynakube.DynaKube) client.Client {
	tlsSecret := getTLSSecret(tls.GetTLSSecretName(dk), dk.Namespace, "super-cert", "super-key")

	err := client.Create(context.Background(), &tlsSecret)
	require.NoError(t, err)

	return client
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

func TestConditions(t *testing.T) {
	t.Run("no kubeSystemUUID", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Status.KubeSystemUUID = ""

		mockK8sClient := fake.NewClient(dk)

		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.Error(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.ExtensionsExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.Error(t, err)

		assert.True(t, errors.IsNotFound(err))
	})

	t.Run("no tenantUUID", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Status.ActiveGate.ConnectionInfo.TenantUUID = ""

		mockK8sClient := fake.NewClient(dk)

		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.Error(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.ExtensionsExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.Error(t, err)

		assert.True(t, errors.IsNotFound(err))
	})

	t.Run("extensions are disabled", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions.Enabled = false
		conditions.SetStatefulSetCreated(dk.Conditions(), extensionsControllerStatefulSetConditionType, dk.ExtensionsExecutionControllerStatefulsetName())

		mockK8sClient := fake.NewClient(dk)

		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.ExtensionsExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.Error(t, err)

		assert.True(t, errors.IsNotFound(err))

		assert.Empty(t, dk.Conditions())
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

func TestSecretHashAnnotation(t *testing.T) {
	t.Run("annotation is set with self-signed tls secret", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = ""
		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.Template.Annotations, 1)
		assert.NotEmpty(t, statefulSet.Spec.Template.Annotations[consts.ExtensionsAnnotationSecretHash])
	})
	t.Run("annotation is set with tlsRefName", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "dummy-secret"
		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.Template.Annotations, 1)
		assert.NotEmpty(t, statefulSet.Spec.Template.Annotations[consts.ExtensionsAnnotationSecretHash])
	})
	t.Run("annotation is updated when TLS Secret gets updated", func(t *testing.T) {
		statefulSet := &appsv1.StatefulSet{}
		dk := getTestDynakube()

		// first reconcile a basic setup - TLS Secret gets created
		mockK8sClient := fake.NewClient(dk)
		mockK8sClient = mockTLSSecret(t, mockK8sClient, dk)

		reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)
		err := reconciler.Reconcile(context.Background())
		require.NoError(t, err)

		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.ExtensionsExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.NoError(t, err)

		originalSecretHash := statefulSet.Spec.Template.Annotations[consts.ExtensionsAnnotationSecretHash]

		// then update the TLS Secret and call reconcile again
		updatedTLSSecret := getTLSSecret(tls.GetTLSSecretName(dk), dk.Namespace, "updated-cert", "updated-key")
		err = mockK8sClient.Update(context.Background(), &updatedTLSSecret)
		require.NoError(t, err)

		err = reconciler.Reconcile(context.Background())
		require.NoError(t, err)
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.ExtensionsExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.NoError(t, err)

		resultingSecretHash := statefulSet.Spec.Template.Annotations[consts.ExtensionsAnnotationSecretHash]

		// original hash and resulting hash should be different, value got updated on reconcile
		assert.NotEqual(t, originalSecretHash, resultingSecretHash)
	})
}

func TestTopologySpreadConstraints(t *testing.T) {
	t.Run("the default TopologySpreadConstraints", func(t *testing.T) {
		dk := getTestDynakube()
		statefulSet := getStatefulset(t, dk)
		appLabels := buildAppLabels(dk.Name)
		assert.Equal(t, utils.BuildTopologySpreadConstraints(dk.Spec.Templates.ExtensionExecutionController.TopologySpreadConstraints, appLabels), statefulSet.Spec.Template.Spec.TopologySpreadConstraints)
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
	t.Run("default environment variables", func(t *testing.T) {
		dk := getTestDynakube()

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, corev1.EnvVar{Name: envTenantId, Value: dk.Status.ActiveGate.ConnectionInfo.TenantUUID}, statefulSet.Spec.Template.Spec.Containers[0].Env[0])
		assert.Equal(t, corev1.EnvVar{Name: envServerUrl, Value: buildActiveGateServiceName(dk) + "." + dk.Namespace + ".svc.cluster.local:443"}, statefulSet.Spec.Template.Spec.Containers[0].Env[1])
		assert.Equal(t, corev1.EnvVar{Name: envEecTokenPath, Value: eecTokenMountPath + "/" + consts.EecTokenSecretKey}, statefulSet.Spec.Template.Spec.Containers[0].Env[2])
		assert.Equal(t, corev1.EnvVar{Name: envEecIngestPort, Value: strconv.Itoa(int(collectorPort))}, statefulSet.Spec.Template.Spec.Containers[0].Env[3])
		assert.Equal(t, corev1.EnvVar{Name: envExtensionsModuleExecPathName, Value: envExtensionsModuleExecPath}, statefulSet.Spec.Template.Spec.Containers[0].Env[4])
		assert.Equal(t, corev1.EnvVar{Name: envDsInstallDirName, Value: envDsInstallDir}, statefulSet.Spec.Template.Spec.Containers[0].Env[5])
		assert.Equal(t, corev1.EnvVar{Name: envK8sClusterId, Value: dk.Status.KubeSystemUUID}, statefulSet.Spec.Template.Spec.Containers[0].Env[6])
		assert.Equal(t, corev1.EnvVar{Name: envK8sExtServiceUrl, Value: "https://" + dk.Name + consts.ExtensionsControllerSuffix + "." + dk.Namespace}, statefulSet.Spec.Template.Spec.Containers[0].Env[7])
		assert.Equal(t, corev1.EnvVar{Name: envDSTokenPath, Value: eecTokenMountPath + "/" + consts.OtelcTokenSecretKey}, statefulSet.Spec.Template.Spec.Containers[0].Env[8])
		assert.Equal(t, corev1.EnvVar{Name: envHttpsCertPathPem, Value: envEecHttpsCertPathPem}, statefulSet.Spec.Template.Spec.Containers[0].Env[9])
		assert.Equal(t, corev1.EnvVar{Name: envHttpsPrivKeyPathPem, Value: envEecHttpsPrivKeyPathPem}, statefulSet.Spec.Template.Spec.Containers[0].Env[10])
		assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envRuntimeConfigMountPath, Value: customConfigMountPath + "/" + runtimeConfigurationFilename})
		assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envCustomCertificateMountPath, Value: customCertificateMountPath})
	})

	t.Run("environment variables with custom EEC tls certificate", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "custom-tls"

		statefulSet := getStatefulset(t, dk)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envHttpsCertPathPem, Value: envEecHttpsCertPathPem})
		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envHttpsPrivKeyPathPem, Value: envEecHttpsPrivKeyPathPem})
	})

	t.Run("environment variables with custom EEC config", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.CustomConfig = "abc"

		statefulSet := getStatefulset(t, dk)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envRuntimeConfigMountPath, Value: customConfigMountPath + "/" + runtimeConfigurationFilename})
	})

	t.Run("environment variables with certificate for extension signature verification", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.CustomExtensionCertificates = "test"

		statefulSet := getStatefulset(t, dk)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envCustomCertificateMountPath, Value: customCertificateMountPath})
	})
}

func TestVolumeMounts(t *testing.T) {
	t.Run("volume mounts", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		expectedVolumeMounts := []corev1.VolumeMount{
			{
				Name:      consts.ExtensionsTokensVolumeName,
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
				ReadOnly:  false,
			},
			{
				Name:      httpsCertVolumeName,
				MountPath: httpsCertMountPath,
				ReadOnly:  true,
			},
		}
		assert.Equal(t, expectedVolumeMounts, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts)
	})

	t.Run("volume mounts with PVC", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim = &corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		}

		statefulSet := getStatefulset(t, dk)

		expectedVolumeMounts := []corev1.VolumeMount{
			{
				Name:      consts.ExtensionsTokensVolumeName,
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
				ReadOnly:  false,
			},
			{
				Name:      httpsCertVolumeName,
				MountPath: httpsCertMountPath,
				ReadOnly:  true,
			},
		}
		assert.Equal(t, expectedVolumeMounts, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts)
	})

	t.Run("volume mounts when set custom EEC tls certificate", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "custom-tls"
		statefulSet := getStatefulset(t, dk)

		expectedVolumeMount := corev1.VolumeMount{
			Name:      httpsCertVolumeName,
			MountPath: httpsCertMountPath,
			ReadOnly:  true,
		}
		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
	})

	t.Run("volume mounts when set certificate for extension signature verification", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.CustomExtensionCertificates = "custom-certs"
		statefulSet := getStatefulset(t, dk)

		expectedVolumeMount := corev1.VolumeMount{
			Name:      customCertificateVolumeName,
			MountPath: customCertificateMountPath,
			ReadOnly:  true,
		}
		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
	})

	t.Run("volume mounts with custom configuration", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.CustomConfig = testCustomConfigConfigMapName

		statefulSet := getStatefulset(t, dk)

		expectedVolumeMounts := []corev1.VolumeMount{
			{
				Name:      consts.ExtensionsTokensVolumeName,
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
				ReadOnly:  false,
			},
			{
				Name:      httpsCertVolumeName,
				MountPath: httpsCertMountPath,
				ReadOnly:  true,
			},
			{
				Name:      customConfigVolumeName,
				MountPath: customConfigMountPath,
				ReadOnly:  true,
			},
		}
		assert.Equal(t, expectedVolumeMounts, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts)
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
		require.Len(t, statefulSet.Spec.Template.ObjectMeta.Annotations, 1)
		assert.NotEmpty(t, statefulSet.Spec.Template.ObjectMeta.Annotations[consts.ExtensionsAnnotationSecretHash])
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
		require.Len(t, statefulSet.Spec.Template.ObjectMeta.Annotations, 2)
		assert.Equal(t, "b", statefulSet.Spec.Template.ObjectMeta.Annotations["a"])
		assert.NotEmpty(t, statefulSet.Spec.Template.ObjectMeta.Annotations[consts.ExtensionsAnnotationSecretHash])
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

func TestPersistentVolumeClaim(t *testing.T) {
	t.Run("no PVC spec", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.Empty(t, statefulSet.Spec.VolumeClaimTemplates)
	})
	t.Run("PVC spec transferred to statefulset", func(t *testing.T) {
		storageClassName := "standard"

		dk := getTestDynakube()

		dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim = &corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOncePod},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
			StorageClassName: &storageClassName,
		}

		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.VolumeClaimTemplates, 1)
		require.Len(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.AccessModes, 1)
		assert.Equal(t, dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim.AccessModes[0], statefulSet.Spec.VolumeClaimTemplates[0].Spec.AccessModes[0])
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.Selector)
		assert.Nil(t, dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim.Resources.Limits)
		assert.Equal(t, dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim.Resources.Requests, statefulSet.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests)
		assert.Empty(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.VolumeName)
		assert.Equal(t, *dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim.StorageClassName, *statefulSet.Spec.VolumeClaimTemplates[0].Spec.StorageClassName)
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.VolumeMode)
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.DataSource)
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.DataSourceRef)
	})
}

func TestPersistentVolumeClaimRetentionPolicy(t *testing.T) {
	t.Run("the default retention policy", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.Nil(t, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy)
	})
	t.Run("custom persistent volume claim retention policy", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaimRetentionPolicy = &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
			WhenDeleted: appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
			WhenScaled:  appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
		}

		statefulSet := getStatefulset(t, dk)

		require.NotNil(t, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy)
		assert.Equal(t, appsv1.RetainPersistentVolumeClaimRetentionPolicyType, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted)
		assert.Equal(t, appsv1.DeletePersistentVolumeClaimRetentionPolicyType, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled)
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
	t.Run("securityContext is set", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		require.NotNil(t, statefulSet.Spec.Template.Spec.SecurityContext)
		assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, statefulSet.Spec.Template.Spec.SecurityContext.SeccompProfile.Type)

		require.NotNil(t, statefulSet.Spec.Template.Spec.Containers[0].SecurityContext)
		assert.Equal(t, int64(1001), *statefulSet.Spec.Template.Spec.Containers[0].SecurityContext.RunAsUser)
		assert.Equal(t, int64(1001), *statefulSet.Spec.Template.Spec.Containers[0].SecurityContext.RunAsGroup)
		assert.False(t, *statefulSet.Spec.Template.Spec.Containers[0].SecurityContext.Privileged)
		assert.True(t, *statefulSet.Spec.Template.Spec.Containers[0].SecurityContext.RunAsNonRoot)
		assert.True(t, *statefulSet.Spec.Template.Spec.Containers[0].SecurityContext.ReadOnlyRootFilesystem)
		assert.False(t, *statefulSet.Spec.Template.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation)

		require.Len(t, statefulSet.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities.Drop, 1)
		assert.Equal(t, corev1.Capability("ALL"), statefulSet.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities.Drop[0])
	})
}

func TestUpdateStrategy(t *testing.T) {
	t.Run("the default update strategy is set", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.NotNil(t, statefulSet.Spec.UpdateStrategy.RollingUpdate.Partition)
		assert.NotEmpty(t, statefulSet.Spec.UpdateStrategy.Type)
	})
}

func TestVolumes(t *testing.T) {
	t.Run("volumes without PVC", func(t *testing.T) {
		dk := getTestDynakube()

		statefulSet := getStatefulset(t, dk)

		mode := int32(420)
		expectedVolumes := []corev1.Volume{
			{
				Name: consts.ExtensionsTokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.ExtensionsTokenSecretName(),
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
				Name: httpsCertVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: dk.ExtensionsTLSSecretName(),
					},
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
				Name: consts.ExtensionsTokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.ExtensionsTokenSecretName(),
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
				Name: httpsCertVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: dk.ExtensionsTLSSecretName(),
					},
				},
			},
		}

		assert.Equal(t, expectedVolumes, statefulSet.Spec.Template.Spec.Volumes)
	})

	t.Run("volumes without PVC and with custom configuration", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.CustomConfig = testCustomConfigConfigMapName

		statefulSet := getStatefulset(t, dk)

		mode := int32(420)
		expectedVolumes := []corev1.Volume{
			{
				Name: consts.ExtensionsTokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.ExtensionsTokenSecretName(),
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
				Name: httpsCertVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: dk.ExtensionsTLSSecretName(),
					},
				},
			},
			{
				Name: runtimeVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},

			{
				Name: customConfigVolumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: testCustomConfigConfigMapName,
						},
					},
				},
			},
		}

		assert.Equal(t, expectedVolumes, statefulSet.Spec.Template.Spec.Volumes)
	})
	t.Run("Custom EEC tls certificate is mounted to EEC", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "custom-tls"
		statefulSet := getStatefulset(t, dk)

		expectedVolume := corev1.Volume{
			Name: httpsCertVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "custom-tls",
				},
			},
		}

		require.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})

	t.Run("volumes with certificate for extension signature verification", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.CustomExtensionCertificates = "custom-certs"
		statefulSet := getStatefulset(t, dk)

		expectedVolume := corev1.Volume{
			Name: customCertificateVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "custom-certs",
				},
			},
		}

		require.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})
}

func TestActiveGateVolumes(t *testing.T) {
	tlsSecretName := "ag-ca"
	expectedEnvVar := corev1.EnvVar{Name: envActiveGateTrustedCertName, Value: envActiveGateTrustedCert}
	expectedVolumeMount := corev1.VolumeMount{
		Name:      activeGateTrustedCertVolumeName,
		MountPath: activeGateTrustedCertMountPath,
		ReadOnly:  true,
	}
	defaultMode := int32(420)
	expectedVolume := corev1.Volume{
		Name: activeGateTrustedCertVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				DefaultMode: &defaultMode,
				SecretName:  tlsSecretName,
				Items: []corev1.KeyToPath{
					{
						Key:  activeGateTrustedCertSecretKeyPath,
						Path: activeGateTrustedCertSecretKeyPath,
					},
				},
			},
		},
	}

	t.Run("ActiveGate tls certificate is mounted to EEC", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.ActiveGate.TlsSecretName = tlsSecretName
		statefulSet := getStatefulset(t, dk)

		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Containers)
		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Volumes)

		require.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, expectedEnvVar)
		require.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
		require.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})
	t.Run("ActiveGate tls certificate is not mounted to EEC", func(t *testing.T) {
		dk := getTestDynakube()
		statefulSet := getStatefulset(t, dk)

		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Containers)
		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Volumes)

		require.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, expectedEnvVar)
		require.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
		require.NotContains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})
}
