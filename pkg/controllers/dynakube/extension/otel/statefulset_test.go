package otel

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/tls"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/utils"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
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
	testDynakubeName   = "dynakube"
	testNamespaceName  = "dynatrace"
	testOtelPullSecret = "otel-pull-secret"
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
			Templates: dynakube.TemplatesSpec{OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{}},
		},
	}
}

func getStatefulset(t *testing.T, dk *dynakube.DynaKube) *appsv1.StatefulSet {
	mockK8sClient := fake.NewClient(dk)
	mockK8sClient = mockTLSSecret(t, mockK8sClient, dk)

	err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
	require.NoError(t, err)

	statefulSet := &appsv1.StatefulSet{}
	err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.ExtensionsCollectorStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
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
	t.Run("extensions are disabled", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions.Enabled = false
		conditions.SetStatefulSetCreated(dk.Conditions(), otelControllerStatefulSetConditionType, dk.ExtensionsCollectorStatefulsetName())

		mockK8sClient := fake.NewClient(dk)

		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.ExtensionsCollectorStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.Error(t, err)

		assert.True(t, errors.IsNotFound(err))

		assert.Empty(t, dk.Conditions())
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

		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.ExtensionsCollectorStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.NoError(t, err)

		originalSecretHash := statefulSet.Spec.Template.Annotations[consts.ExtensionsAnnotationSecretHash]

		// then update the TLS Secret and call reconcile again
		updatedTLSSecret := getTLSSecret(tls.GetTLSSecretName(dk), dk.Namespace, "updated-cert", "updated-key")
		err = mockK8sClient.Update(context.Background(), &updatedTLSSecret)
		require.NoError(t, err)

		err = reconciler.Reconcile(context.Background())
		require.NoError(t, err)
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.ExtensionsCollectorStatefulsetName(), Namespace: dk.Namespace}, statefulSet)
		require.NoError(t, err)

		resultingSecretHash := statefulSet.Spec.Template.Annotations[consts.ExtensionsAnnotationSecretHash]

		// original hash and resulting hash should be different, value got updated on reconcile
		assert.NotEqual(t, originalSecretHash, resultingSecretHash)
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

func TestServiceAccountName(t *testing.T) {
	t.Run("serviceAccountName is set", func(t *testing.T) {
		statefulSet := getStatefulset(t, getTestDynakube())

		assert.Equal(t, serviceAccountName, statefulSet.Spec.Template.Spec.ServiceAccountName)
		assert.Equal(t, serviceAccountName, statefulSet.Spec.Template.Spec.DeprecatedServiceAccount)
	})
}

func TestTopologySpreadConstraints(t *testing.T) {
	t.Run("the default TopologySpreadConstraints", func(t *testing.T) {
		dk := getTestDynakube()
		statefulSet := getStatefulset(t, dk)
		appLabels := buildAppLabels(dk.Name)
		assert.Equal(t, utils.BuildTopologySpreadConstraints(dk.Spec.Templates.OpenTelemetryCollector.TopologySpreadConstraints, appLabels), statefulSet.Spec.Template.Spec.TopologySpreadConstraints)
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

		dk.Spec.Templates.OpenTelemetryCollector.TopologySpreadConstraints = customTopologySpreadConstraints

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, customTopologySpreadConstraints, statefulSet.Spec.Template.Spec.TopologySpreadConstraints)
	})
}

func TestEnvironmentVariables(t *testing.T) {
	t.Run("environment variables", func(t *testing.T) {
		dk := getTestDynakube()

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, corev1.EnvVar{Name: envShards, Value: fmt.Sprintf("%d", getReplicas(dk))}, statefulSet.Spec.Template.Spec.Containers[0].Env[0])
		assert.Equal(t, corev1.EnvVar{Name: envPodNamePrefix, Value: dk.Name + "-extensions-collector"}, statefulSet.Spec.Template.Spec.Containers[0].Env[1])
		assert.Equal(t, corev1.EnvVar{Name: envPodName, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['statefulset.kubernetes.io/pod-name']",
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[2])
		assert.Equal(t, corev1.EnvVar{Name: envShardId, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['apps.kubernetes.io/pod-index']",
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[3])
		assert.Equal(t, corev1.EnvVar{Name: envOTLPgrpcPort, Value: defaultOLTPgrpcPort}, statefulSet.Spec.Template.Spec.Containers[0].Env[4])
		assert.Equal(t, corev1.EnvVar{Name: envOTLPhttpPort, Value: defaultOLTPhttpPort}, statefulSet.Spec.Template.Spec.Containers[0].Env[5])
		assert.Equal(t, corev1.EnvVar{Name: envOTLPtoken, ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: dk.ExtensionsTokenSecretName()},
				Key:                  consts.OtelcTokenSecretKey,
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[6])
		assert.Equal(t, corev1.EnvVar{Name: envEECDStoken, ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: dk.ExtensionsTokenSecretName()},
				Key:                  consts.OtelcTokenSecretKey,
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[7])
		assert.Equal(t, corev1.EnvVar{Name: envCertDir, Value: customEecTLSCertificatePath}, statefulSet.Spec.Template.Spec.Containers[0].Env[8])
		assert.Equal(t, corev1.EnvVar{Name: envK8sClusterName, Value: dk.Name}, statefulSet.Spec.Template.Spec.Containers[0].Env[9])
		assert.Equal(t, corev1.EnvVar{Name: envK8sClusterUuid, Value: dk.Status.KubeSystemUUID}, statefulSet.Spec.Template.Spec.Containers[0].Env[10])
		assert.Equal(t, corev1.EnvVar{Name: envDTentityK8sCluster, Value: dk.Status.KubernetesClusterMEID}, statefulSet.Spec.Template.Spec.Containers[0].Env[11])
	})
	t.Run("environment variables with trustedCA", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.TrustedCAs = "test-trusted-ca"

		statefulSet := getStatefulset(t, dk)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envTrustedCAs, Value: trustedCAVolumePath})
	})

	t.Run("environment variables with custom EEC TLS certificate", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "test-tls-ca"

		statefulSet := getStatefulset(t, dk)

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envEECcontrollerTLS, Value: customEecTLSCertificateFullPath})
	})
}

func TestAffinity(t *testing.T) {
	t.Run("affinity", func(t *testing.T) {
		dk := getTestDynakube()
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
		dk.Spec.CustomPullSecret = testOtelPullSecret

		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.Spec.Template.Spec.ImagePullSecrets, 2)
		assert.Equal(t, dk.Name+dynakube.PullSecretSuffix, statefulSet.Spec.Template.Spec.ImagePullSecrets[0].Name)
		assert.Equal(t, dk.Spec.CustomPullSecret, statefulSet.Spec.Template.Spec.ImagePullSecrets[1].Name)
	})
}

func TestResources(t *testing.T) {
	t.Run("no resources", func(t *testing.T) {
		dk := getTestDynakube()
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

		assert.Equal(t, dk.Spec.Templates.OpenTelemetryCollector.Resources, statefulSet.Spec.Template.Spec.Containers[0].Resources)
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
		dk.Spec.Templates.OpenTelemetryCollector.Labels = customLabels

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
		dk.Spec.Templates.OpenTelemetryCollector.Annotations = customAnnotations

		statefulSet := getStatefulset(t, dk)

		require.Len(t, statefulSet.ObjectMeta.Annotations, 1)
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
		dk.Spec.Templates.OpenTelemetryCollector.Tolerations = customTolerations

		statefulSet := getStatefulset(t, dk)

		assert.Equal(t, customTolerations, statefulSet.Spec.Template.Spec.Tolerations)
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

func TestVolumes(t *testing.T) {
	t.Run("volume mounts with trusted CAs", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.TrustedCAs = "test-trusted-cas"
		statefulSet := getStatefulset(t, dk)

		expectedVolumeMount := corev1.VolumeMount{
			Name:      caCertsVolumeName,
			MountPath: trustedCAVolumeMountPath,
			ReadOnly:  true,
		}
		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
	})
	t.Run("volume mounts without trusted CAs", func(t *testing.T) {
		dk := getTestDynakube()
		statefulSet := getStatefulset(t, dk)

		expectedVolumeMount := corev1.VolumeMount{
			Name:      caCertsVolumeName,
			MountPath: trustedCAVolumeMountPath,
			ReadOnly:  true,
		}

		assert.NotContains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
	})
	t.Run("volumes and volume mounts with custom EEC TLS certificate", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "test-tls-name"
		statefulSet := getStatefulset(t, dk)

		expectedVolumeMount := corev1.VolumeMount{
			Name:      dk.ExtensionsTLSSecretName(),
			MountPath: customEecTLSCertificatePath,
			ReadOnly:  true,
		}

		expectedVolume := corev1.Volume{Name: dk.Spec.Templates.ExtensionExecutionController.TlsRefName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dk.Spec.Templates.ExtensionExecutionController.TlsRefName,
					Items: []corev1.KeyToPath{
						{
							Key:  consts.TLSCrtDataName,
							Path: consts.TLSCrtDataName,
						},
					},
				},
			}}

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts, expectedVolumeMount)
		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})
	t.Run("volumes with trusted CAs", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.TrustedCAs = "test-trusted-cas"
		statefulSet := getStatefulset(t, dk)

		expectedVolume := corev1.Volume{
			Name: caCertsVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: dk.Spec.TrustedCAs,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "certs",
							Path: trustedCAsFile,
						},
					},
				},
			},
		}
		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})
	t.Run("volumes with otel token", func(t *testing.T) {
		dk := getTestDynakube()
		statefulSet := getStatefulset(t, dk)

		expectedVolume := corev1.Volume{
			Name: consts.ExtensionsTokensVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dk.ExtensionsTokenSecretName(),
					Items: []corev1.KeyToPath{
						{
							Key:  consts.OtelcTokenSecretKey,
							Path: consts.OtelcTokenSecretKey,
						},
					},
					DefaultMode: address.Of(int32(420)),
				},
			},
		}

		assert.Contains(t, statefulSet.Spec.Template.Spec.Volumes, expectedVolume)
	})
}
