package otel

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/utils"
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
				Prometheus: dynakube.PrometheusSpec{
					Enabled: true,
				},
			},
			Templates: dynakube.TemplatesSpec{OpenTelemetryCollector: dynakube.OpenTelemetryCollectorSpec{}},
		},
	}
}

func getStatefulset(t *testing.T, dk *dynakube.DynaKube) *appsv1.StatefulSet {
	mockK8sClient := fake.NewClient(dk)

	err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
	require.NoError(t, err)

	statefulSet := &appsv1.StatefulSet{}
	err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dynakube.ExtensionsCollectorStatefulsetName, Namespace: dk.Namespace}, statefulSet)
	require.NoError(t, err)

	return statefulSet
}

func TestConditions(t *testing.T) {
	t.Run("prometheus is disabled", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Extensions.Prometheus.Enabled = false
		conditions.SetStatefulSetCreated(dk.Conditions(), otelControllerStatefulSetConditionType, dynakube.ExtensionsCollectorStatefulsetName)

		mockK8sClient := fake.NewClient(dk)

		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dynakube.ExtensionsCollectorStatefulsetName, Namespace: dk.Namespace}, statefulSet)
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
		assert.Equal(t, corev1.EnvVar{Name: envPodNamePrefix, Value: defaultPodNamePrefix}, statefulSet.Spec.Template.Spec.Containers[0].Env[1])
		assert.Equal(t, corev1.EnvVar{Name: envPodName, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['statefulset.kubernetes.io/pod-name']",
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[2])
		assert.Equal(t, corev1.EnvVar{Name: envShardId, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.labels['app.kubernetes.io/pod-index']",
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[3])
		assert.Equal(t, corev1.EnvVar{Name: envOTLPgrpcPort, Value: defaultOLTPgrpcPort}, statefulSet.Spec.Template.Spec.Containers[0].Env[4])
		assert.Equal(t, corev1.EnvVar{Name: envOTLPhttpPort, Value: defaultOLTPhttpPort}, statefulSet.Spec.Template.Spec.Containers[0].Env[5])
		assert.Equal(t, corev1.EnvVar{Name: envOTLPtoken, ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: dk.Name + consts.SecretSuffix},
				Key:                  tokenSecretKey,
			},
		}}, statefulSet.Spec.Template.Spec.Containers[0].Env[6])
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

		assert.Contains(t, statefulSet.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: envEECcontrollerTls, Value: customEecTlsCertificateFullPath})
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
		assert.Empty(t, statefulSet.Spec.Template.ObjectMeta.Annotations)
	})

	t.Run("custom annotations", func(t *testing.T) {
		dk := getTestDynakube()
		customAnnotations := map[string]string{
			"a": "b",
		}
		dk.Spec.Templates.OpenTelemetryCollector.Annotations = customAnnotations

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

		expectedVolumeMounts := []corev1.VolumeMount{
			{
				Name:      caCertsVolumeName,
				MountPath: trustedCAVolumeMountPath,
				ReadOnly:  true,
			},
		}
		assert.Equal(t, expectedVolumeMounts, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
	t.Run("volumes and volume mounts without trusted CAs", func(t *testing.T) {
		dk := getTestDynakube()
		statefulSet := getStatefulset(t, dk)

		var expectedVolumeMounts []corev1.VolumeMount

		var expectedVolumes []corev1.Volume

		assert.Equal(t, expectedVolumeMounts, statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts)
		assert.Equal(t, expectedVolumes, statefulSet.Spec.Template.Spec.Volumes)
	})

	t.Run("volumes and volume mounts with custom EEC TLS certificate", func(t *testing.T) {
		dk := getTestDynakube()
		dk.Spec.Templates.ExtensionExecutionController.TlsRefName = "test-tls-name"
		statefulSet := getStatefulset(t, dk)

		expectedVolumeMount := corev1.VolumeMount{
			Name:      consts.ExtensionsCustomTlsCertificate,
			MountPath: customEecTlsCertificatePath,
			ReadOnly:  true,
		}

		expectedVolume := corev1.Volume{Name: consts.ExtensionsCustomTlsCertificate,
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

		expectedVolumes := []corev1.Volume{
			{
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
			},
		}

		assert.Equal(t, expectedVolumes, statefulSet.Spec.Template.Spec.Volumes)
	})
}
