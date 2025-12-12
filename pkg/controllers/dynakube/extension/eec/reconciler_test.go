package eec

import (
	"strconv"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	eecConsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8saffinity"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8stopology"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}},
			Templates: dynakube.TemplatesSpec{
				ExtensionExecutionController: extensions.ExecutionControllerSpec{
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

	err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
	require.NoError(t, err)

	statefulSet := &appsv1.StatefulSet{}
	err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.Extensions().GetExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
	require.NoError(t, err)

	return statefulSet
}

func mockTLSSecret(t *testing.T, client client.Client, dk *dynakube.DynaKube) client.Client {
	tlsSecret := getTLSSecret(dk.Extensions().GetTLSSecretName(), dk.Namespace, "super-cert", "super-key")

	err := client.Create(t.Context(), &tlsSecret)
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

func disableAutomaticAGCertificate(dk *dynakube.DynaKube) {
	dk.Annotations[exp.AGAutomaticTLSCertificateKey] = "false"
}

func TestConditions(t *testing.T) {
	t.Run("no kubeSystemUUID", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Status.KubeSystemUUID = ""

		mockK8sClient := fake.NewClient(dk)

		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
		require.Error(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.Extensions().GetExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.Error(t, err)

		assert.True(t, errors.IsNotFound(err))
	})

	t.Run("no tenantUUID", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Status.ActiveGate.ConnectionInfo.TenantUUID = ""

		mockK8sClient := fake.NewClient(dk)

		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
		require.Error(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.Extensions().GetExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.Error(t, err)

		assert.True(t, errors.IsNotFound(err))
	})

	t.Run("extensions are disabled", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions = nil
		conditions.SetStatefulSetCreated(dk.Conditions(), extensionControllerStatefulSetConditionType, dk.Extensions().GetExecutionControllerStatefulsetName())

		mockK8sClient := fake.NewClient(dk)

		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
		require.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.Extensions().GetExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
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
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = ""
		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.Template.Annotations, 1)
		assert.NotEmpty(t, statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash])
		assert.Contains(t, statefulSet.Annotations, hasher.AnnotationHash)
	})
	t.Run("annotation is set with tlsRefName", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = "dummy-secret"
		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.Template.Annotations, 1)
		assert.NotEmpty(t, statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash])
		assert.Contains(t, statefulSet.Annotations, hasher.AnnotationHash)
	})
	t.Run("annotation is updated when TLS Secret gets updated", func(t *testing.T) {
		statefulSet := &appsv1.StatefulSet{}
		dk := getTestDynakube()

		// first reconcile a basic setup - TLS Secret gets created
		mockK8sClient := fake.NewClient(dk)
		mockK8sClient = mockTLSSecret(t, mockK8sClient, dk)

		reconciler := NewReconciler(mockK8sClient, mockK8sClient, dk)
		err := reconciler.Reconcile(t.Context())
		require.NoError(t, err)

		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.Extensions().GetExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.NoError(t, err)

		originalSecretHash := statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash]

		// then update the TLS Secret and call reconcile again
		updatedTLSSecret := getTLSSecret(dk.Extensions().GetTLSSecretName(), dk.Namespace, "updated-cert", "updated-key")
		err = mockK8sClient.Update(t.Context(), &updatedTLSSecret)
		require.NoError(t, err)

		err = reconciler.Reconcile(t.Context())
		require.NoError(t, err)
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.Extensions().GetExecutionControllerStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.NoError(t, err)

		resultingSecretHash := statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash]

		// original hash and resulting hash should be different, value got updated on reconcile
		assert.NotEqual(t, originalSecretHash, resultingSecretHash)
	})
}

func TestTopologySpreadConstraints(t *testing.T) {
	t.Run("the default TopologySpreadConstraints", func(t *testing.T) {
		dk := getTestDynakube()
		statefulSet := getStatefulset(t, dk)
		appLabels := buildAppLabels(dk.Name)
		assert.Equal(t, k8stopology.MaxOnePerNode(appLabels), statefulSet.Spec.Template.Spec.TopologySpreadConstraints)
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

		assert.Equal(t, corev1.EnvVar{Name: envTenantID, Value: dk.Status.ActiveGate.ConnectionInfo.TenantUUID}, statefulSet.Spec.Template.Spec.Containers[0].Env[0])
		assert.Equal(t, corev1.EnvVar{Name: envServerURL, Value: buildActiveGateServiceName(dk) + "." + dk.Namespace + ":443"}, statefulSet.Spec.Template.Spec.Containers[0].Env[1])
		assert.Equal(t, corev1.EnvVar{Name: envEecTokenPath, Value: eecTokenMountPath + "/" + eecConsts.TokenSecretKey}, statefulSet.Spec.Template.Spec.Containers[0].Env[2])
		assert.Equal(t, corev1.EnvVar{Name: envEecIngestPort, Value: strconv.Itoa(consts.ExtensionsDatasourceTargetPort)}, statefulSet.Spec.Template.Spec.Containers[0].Env[3])
		assert.Equal(t, corev1.EnvVar{Name: envExtensionsModuleExecPathName, Value: envExtensionsModuleExecPath}, statefulSet.Spec.Template.Spec.Containers[0].Env[4])
		assert.Equal(t, corev1.EnvVar{Name: envDsInstallDirName, Value: envDsInstallDir}, statefulSet.Spec.Template.Spec.Containers[0].Env[5])
		assert.Equal(t, corev1.EnvVar{Name: envK8sClusterID, Value: dk.Status.KubeSystemUUID}, statefulSet.Spec.Template.Spec.Containers[0].Env[6])
		assert.Equal(t, corev1.EnvVar{Name: envK8sExtServiceURL, Value: "https://" + dk.Name + eecConsts.ExtensionControllerSuffix + "." + dk.Namespace}, statefulSet.Spec.Template.Spec.Containers[0].Env[7])
		assert.Equal(t, corev1.EnvVar{Name: envDSTokenPath, Value: eecTokenMountPath + "/" + consts.DatasourceTokenSecretKey}, statefulSet.Spec.Template.Spec.Containers[0].Env[8])
		assert.Equal(t, corev1.EnvVar{Name: envHTTPSCertPathPem, Value: envEecHTTPSCertPathPem}, statefulSet.Spec.Template.Spec.Containers[0].Env[9])
		assert.Equal(t, corev1.EnvVar{Name: envHTTPSPrivKeyPathPem, Value: envEecHTTPSPrivKeyPathPem}, statefulSet.Spec.Template.Spec.Containers[0].Env[10])
		assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envRuntimeConfigMountPath, Value: customConfigMountPath + "/" + runtimeConfigurationFilename})
		assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envCustomCertificateMountPath, Value: customCertificateMountPath})
	})

	t.Run("environment variables with custom EEC tls certificate", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = "custom-tls"

		statefulSet := getStatefulset(t, dk)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envHTTPSCertPathPem, Value: envEecHTTPSCertPathPem})
		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envHTTPSPrivKeyPathPem, Value: envEecHTTPSPrivKeyPathPem})
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
	t.Run("volume mounts, AG cert disabled", func(t *testing.T) {
		dk := getTestDynakube()
		disableAutomaticAGCertificate(dk)
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
			{
				Name:      activeGateTrustedCertVolumeName,
				MountPath: activeGateTrustedCertMountPath,
				ReadOnly:  true,
			},
		}
		assert.Equal(t, expectedVolumeMounts, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts)
	})

	t.Run("volume mounts with PVC, AG cert disabled", func(t *testing.T) {
		dk := getTestDynakube()
		disableAutomaticAGCertificate(dk)
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
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = "custom-tls"
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

	t.Run("volume mounts with custom configuration, AG cert disabled", func(t *testing.T) {
		dk := getTestDynakube()
		disableAutomaticAGCertificate(dk)
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
			{
				Name:      activeGateTrustedCertVolumeName,
				MountPath: activeGateTrustedCertMountPath,
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

		expectedAffinity := k8saffinity.NewMultiArchNodeAffinity()
		assert.Equal(t, expectedAffinity, *statefulSet.Spec.Template.Spec.Affinity)
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

		assert.Equal(t, appLabels.BuildLabels(), statefulSet.Labels)
		assert.Equal(t, &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}, statefulSet.Spec.Selector)
		assert.Equal(t, appLabels.BuildLabels(), statefulSet.Spec.Template.Labels)
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

		assert.Equal(t, appLabels.BuildLabels(), statefulSet.Labels)
		assert.Equal(t, &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}, statefulSet.Spec.Selector)
		assert.Equal(t, podLabels, statefulSet.Spec.Template.Labels)
	})
}

func TestAnnotations(t *testing.T) {
	t.Run("the default annotations", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.Len(t, statefulSet.Annotations, 2)
		require.Len(t, statefulSet.Spec.Template.Annotations, 1)
		assert.NotEmpty(t, statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash])
	})

	t.Run("custom annotations", func(t *testing.T) {
		dk := getTestDynakube()
		customAnnotations := map[string]string{
			"a": "b",
		}
		dk.Spec.Templates.ExtensionExecutionController.Annotations = customAnnotations

		statefulSet := getStatefulset(t, dk)

		assert.Len(t, statefulSet.Annotations, 2)
		assert.Empty(t, statefulSet.Annotations["a"])
		require.Len(t, statefulSet.Spec.Template.Annotations, 2)
		assert.Equal(t, "b", statefulSet.Spec.Template.Annotations["a"])
		assert.NotEmpty(t, statefulSet.Spec.Template.Annotations[api.AnnotationExtensionsSecretHash])
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
	testDefaultPVCSpec := func(statefulSet *appsv1.StatefulSet) {
		require.Len(t, statefulSet.Spec.VolumeClaimTemplates, 1)

		require.Len(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.AccessModes, 1)
		assert.Equal(t, corev1.ReadWriteOnce, statefulSet.Spec.VolumeClaimTemplates[0].Spec.AccessModes[0])

		assert.Empty(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.Resources.Limits)
		require.Len(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests, 1)
		assert.Equal(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage], resource.MustParse("1Gi"))

		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.Selector)
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.StorageClassName)
		assert.Empty(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.VolumeName)
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.VolumeMode)
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.DataSource)
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.DataSourceRef)

		require.NotNil(t, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy)
		assert.Equal(t, appsv1.DeletePersistentVolumeClaimRetentionPolicyType, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted)
		assert.Equal(t, appsv1.DeletePersistentVolumeClaimRetentionPolicyType, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled)
	}

	t.Run("no PVC spec, UseEphemeralVolume set to false", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = false
		statefulSet := getStatefulset(t, dk)

		testDefaultPVCSpec(statefulSet)
	})
	t.Run("no PVC spec, UseEphemeralVolume set to true", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = true
		statefulSet := getStatefulset(t, dk)

		assert.Empty(t, statefulSet.Spec.VolumeClaimTemplates)
		assert.Nil(t, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy)
	})

	t.Run("empty PVC spec, UseEphemeralVolume set to false", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = false
		dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim = &corev1.PersistentVolumeClaimSpec{}
		statefulSet := getStatefulset(t, dk)

		assert.Len(t, statefulSet.Spec.VolumeClaimTemplates, 1)
	})
	t.Run("empty PVC spec, UseEphemeralVolume set to true", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = true
		dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim = &corev1.PersistentVolumeClaimSpec{}
		statefulSet := getStatefulset(t, dk)

		assert.Empty(t, statefulSet.Spec.VolumeClaimTemplates)
		assert.Nil(t, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy)
	})

	t.Run("PVC spec, UseEphemeralVolume set to false", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = false
		dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim = &corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOncePod,
			},
		}
		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.VolumeClaimTemplates, 1)
		require.Len(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.AccessModes, 1)
		assert.Equal(t, corev1.ReadWriteOncePod, statefulSet.Spec.VolumeClaimTemplates[0].Spec.AccessModes[0])
	})
	t.Run("PVC spec, UseEphemeralVolume set to true", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = true
		dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim = &corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOncePod,
			},
		}
		statefulSet := getStatefulset(t, dk)

		assert.Empty(t, statefulSet.Spec.VolumeClaimTemplates)
		assert.Nil(t, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy)
	})

	t.Run("PVC spec transferred to statefulset", func(t *testing.T) {
		storageClassName := "standard"

		dk := getTestDynakube()

		dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim = &corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOncePod},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("2Gi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("4Gi"),
				},
			},
			StorageClassName: &storageClassName,
		}

		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.VolumeClaimTemplates, 1)
		require.Len(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.AccessModes, 1)
		assert.Equal(t, dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim.AccessModes[0], statefulSet.Spec.VolumeClaimTemplates[0].Spec.AccessModes[0])
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.Selector)
		assert.Equal(t, dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim.Resources.Limits, statefulSet.Spec.VolumeClaimTemplates[0].Spec.Resources.Limits)
		assert.Equal(t, dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim.Resources.Requests, statefulSet.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests)
		assert.Empty(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.VolumeName)
		assert.Equal(t, *dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim.StorageClassName, *statefulSet.Spec.VolumeClaimTemplates[0].Spec.StorageClassName)
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.VolumeMode)
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.DataSource)
		assert.Nil(t, statefulSet.Spec.VolumeClaimTemplates[0].Spec.DataSourceRef)

		require.NotNil(t, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy)
		assert.Equal(t, appsv1.DeletePersistentVolumeClaimRetentionPolicyType, statefulSet.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted)
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
	t.Run("volumes without PVC, AG cert disabled", func(t *testing.T) {
		dk := getTestDynakube()
		disableAutomaticAGCertificate(dk)
		dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = true

		statefulSet := getStatefulset(t, dk)

		mode := int32(420)
		expectedVolumes := []corev1.Volume{
			{
				Name: consts.ExtensionsTokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.Extensions().GetTokenSecretName(),
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
						SecretName: dk.Extensions().GetTLSSecretName(),
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

	t.Run("volumes without PVC", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = true

		statefulSet := getStatefulset(t, dk)

		mode := int32(420)
		expectedVolumes := []corev1.Volume{
			{
				Name: consts.ExtensionsTokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.Extensions().GetTokenSecretName(),
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
						SecretName: dk.Extensions().GetTLSSecretName(),
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
				Name: activeGateTrustedCertVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						DefaultMode: &mode,
						SecretName:  dk.ActiveGate().GetTLSSecretName(),
						Items: []corev1.KeyToPath{
							{
								Key:  activeGateTrustedCertSecretKeyPath,
								Path: activeGateTrustedCertSecretKeyPath,
							},
						},
					},
				},
			},
		}

		assert.Equal(t, expectedVolumes, statefulSet.Spec.Template.Spec.Volumes)
	})

	t.Run("volumes with PVC, AG cert disabled", func(t *testing.T) {
		dk := getTestDynakube()
		disableAutomaticAGCertificate(dk)
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
						SecretName:  dk.Extensions().GetTokenSecretName(),
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
						SecretName: dk.Extensions().GetTLSSecretName(),
					},
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
						SecretName:  dk.Extensions().GetTokenSecretName(),
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
						SecretName: dk.Extensions().GetTLSSecretName(),
					},
				},
			},
			{
				Name: activeGateTrustedCertVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						DefaultMode: &mode,
						SecretName:  dk.ActiveGate().GetTLSSecretName(),
						Items: []corev1.KeyToPath{
							{
								Key:  activeGateTrustedCertSecretKeyPath,
								Path: activeGateTrustedCertSecretKeyPath,
							},
						},
					},
				},
			},
		}

		assert.Equal(t, expectedVolumes, statefulSet.Spec.Template.Spec.Volumes)
	})

	t.Run("volumes without PVC and with custom configuration, AG cert disabled", func(t *testing.T) {
		dk := getTestDynakube()
		disableAutomaticAGCertificate(dk)
		dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = true
		dk.Spec.Templates.ExtensionExecutionController.CustomConfig = testCustomConfigConfigMapName

		statefulSet := getStatefulset(t, dk)

		mode := int32(420)
		expectedVolumes := []corev1.Volume{
			{
				Name: consts.ExtensionsTokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.Extensions().GetTokenSecretName(),
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
						SecretName: dk.Extensions().GetTLSSecretName(),
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

	t.Run("volumes without PVC and with custom configuration", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.UseEphemeralVolume = true
		dk.Spec.Templates.ExtensionExecutionController.CustomConfig = testCustomConfigConfigMapName

		statefulSet := getStatefulset(t, dk)

		mode := int32(420)
		expectedVolumes := []corev1.Volume{
			{
				Name: consts.ExtensionsTokensVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dk.Extensions().GetTokenSecretName(),
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
						SecretName: dk.Extensions().GetTLSSecretName(),
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
			{
				Name: activeGateTrustedCertVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						DefaultMode: &mode,
						SecretName:  dk.ActiveGate().GetTLSSecretName(),
						Items: []corev1.KeyToPath{
							{
								Key:  activeGateTrustedCertSecretKeyPath,
								Path: activeGateTrustedCertSecretKeyPath,
							},
						},
					},
				},
			},
		}

		assert.Equal(t, expectedVolumes, statefulSet.Spec.Template.Spec.Volumes)
	})

	t.Run("Custom EEC tls certificate is mounted to EEC", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TLSRefName = "custom-tls"
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
	expectedAutoAgCertVolume := corev1.Volume{
		Name: activeGateTrustedCertVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				DefaultMode: &defaultMode,
				SecretName:  testDynakubeName + activegate.TLSSecretSuffix,
				Items: []corev1.KeyToPath{
					{
						Key:  activeGateTrustedCertSecretKeyPath,
						Path: activeGateTrustedCertSecretKeyPath,
					},
				},
			},
		},
	}

	t.Run("volumes with custom ActiveGate tls certificate", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.ActiveGate.TLSSecretName = tlsSecretName
		statefulSet := getStatefulset(t, dk)

		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Containers)
		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Volumes)

		require.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, expectedEnvVar)
		require.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
		require.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})

	t.Run("volumes with automatically created ActiveGate tls certificate", func(t *testing.T) {
		dk := getTestDynakube()
		statefulSet := getStatefulset(t, dk)

		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Containers)
		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Volumes)

		require.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, expectedEnvVar)
		require.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
		require.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedAutoAgCertVolume)
	})

	t.Run("volumes without custom ActiveGate tls certificate", func(t *testing.T) {
		dk := getTestDynakube()
		disableAutomaticAGCertificate(dk)
		statefulSet := getStatefulset(t, dk)

		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Containers)
		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Volumes)

		require.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, expectedEnvVar)
		require.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
		require.NotContains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})

	t.Run("volumes with TrustedCAs certificates, AG cert disabled", func(t *testing.T) {
		dk := getTestDynakube()
		disableAutomaticAGCertificate(dk)
		dk.Spec.TrustedCAs = "custom-tls"
		statefulSet := getStatefulset(t, dk)

		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Containers)
		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Volumes)

		require.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, expectedEnvVar)
		require.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
		require.NotContains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})

	t.Run("volumes with TrustedCAs certificates and automatically created ActiveGate tls certificate", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.TrustedCAs = "custom-tls"
		statefulSet := getStatefulset(t, dk)

		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Containers)
		require.NotEmpty(t, statefulSet.Spec.Template.Spec.Volumes)

		require.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, expectedEnvVar)
		require.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
		require.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedAutoAgCertVolume)
	})
}
