package statefulset

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder/modifiers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testKubeUID       = "test-uid"
	testConfigHash    = "test-hash"
	testDynakubeName  = "test-dynakube"
	testNamespaceName = "test-namespace"
)

var (
	testReplicas int32 = 69
)

func getTestDynakube() dynatracev1beta1.DynaKube {
	return dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDynakubeName,
			Namespace:   testNamespaceName,
			Annotations: map[string]string{},
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.RoutingCapability.DisplayName,
				},
				CapabilityProperties: dynatracev1beta1.CapabilityProperties{
					Replicas: &testReplicas,
				},
			},
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			ActiveGate: dynatracev1beta1.ActiveGateStatus{
				VersionStatus: status.VersionStatus{},
			},
		},
	}
}

func TestGetBaseObjectMeta(t *testing.T) {
	dynakube := getTestDynakube()

	t.Run("creating object meta", func(t *testing.T) {
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)

		objectMeta := builder.getBaseObjectMeta()

		require.NotEmpty(t, objectMeta)
		assert.Contains(t, objectMeta.Name, dynakube.Name)
		assert.Contains(t, objectMeta.Name, multiCapability.ShortName())
		assert.NotNil(t, objectMeta.Annotations)
	})
	t.Run("default annotations", func(t *testing.T) {
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts, _ := builder.CreateStatefulSet(nil)
		expectedTemplateAnnotations := map[string]string{
			consts.AnnotationActiveGateConfigurationHash: testConfigHash,
		}

		require.NotEmpty(t, sts.Spec.Template.Labels)
		assert.Equal(t, expectedTemplateAnnotations, sts.Spec.Template.Annotations)
	})
	t.Run("has default node affinity", func(t *testing.T) {
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts, _ := builder.CreateStatefulSet(nil)
		expectedNodeSelectorTerms := []corev1.NodeSelectorTerm{
			{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "kubernetes.io/arch",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"amd64", "arm64", "ppc64le"},
					},
					{
						Key:      "kubernetes.io/os",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"linux"},
					},
				},
			}}

		require.NotEmpty(t, sts.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)
		assert.Contains(t, sts.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, expectedNodeSelectorTerms[0])
	})
	t.Run("add annotations", func(t *testing.T) {
		dynakube.Spec.ActiveGate.Annotations = map[string]string{
			"test": "test",
		}
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts, _ := builder.CreateStatefulSet(nil)
		expectedTemplateAnnotations := map[string]string{
			consts.AnnotationActiveGateConfigurationHash: testConfigHash,
			"test": "test",
		}

		require.NotEmpty(t, sts.Spec.Template.Labels)
		assert.Equal(t, expectedTemplateAnnotations, sts.Spec.Template.Annotations)
	})
}

func TestGetBaseSpec(t *testing.T) {
	dynakube := getTestDynakube()

	t.Run("creating base statefulset spec", func(t *testing.T) {
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)

		stsSpec := builder.getBaseSpec()

		require.NotEmpty(t, stsSpec)
		assert.Equal(t, &testReplicas, stsSpec.Replicas)
		require.NotNil(t, stsSpec.Template.Annotations)
		assert.Equal(t, testConfigHash, stsSpec.Template.Annotations[consts.AnnotationActiveGateConfigurationHash])
	})
}

func TestAddLabels(t *testing.T) {
	t.Run("adds labels", func(t *testing.T) {
		dynakube := getTestDynakube()
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts := appsv1.StatefulSet{}
		appLabels := labels.NewAppLabels(labels.ActiveGateComponentLabel, builder.dynakube.Name, builder.capability.ShortName(), "")
		expectedLabels := appLabels.BuildLabels()
		expectedSelectorLabels := metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}

		builder.addLabels(&sts)

		require.NotEmpty(t, sts.ObjectMeta.Labels)
		assert.Equal(t, expectedLabels, sts.ObjectMeta.Labels)
		assert.Equal(t, expectedSelectorLabels, *sts.Spec.Selector)
		assert.Equal(t, expectedLabels, sts.Spec.Template.Labels)
	})

	t.Run("merge labels", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Spec.ActiveGate.Labels = map[string]string{
			"test": "test",
		}
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts := appsv1.StatefulSet{}
		appLabels := labels.NewAppLabels(labels.ActiveGateComponentLabel, builder.dynakube.Name, builder.capability.ShortName(), "")
		expectedTemplateLabels := appLabels.BuildLabels()
		expectedTemplateLabels["test"] = "test"

		builder.addLabels(&sts)

		require.NotEmpty(t, sts.Spec.Template.Labels)
		assert.Equal(t, expectedTemplateLabels, sts.Spec.Template.Labels)
	})
}

func TestAddTemplateSpec(t *testing.T) {
	t.Run("adds template spec", func(t *testing.T) {
		dynakube := getTestDynakube()
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts := appsv1.StatefulSet{}

		builder.addTemplateSpec(&sts)
		spec := sts.Spec.Template.Spec

		assert.NotEmpty(t, spec.Containers)
		assert.NotEmpty(t, spec.Affinity)
		assert.Equal(t, dynakube.PullSecretName(), spec.ImagePullSecrets[0].Name)
	})

	t.Run("adds capability specific stuff", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Spec.ActiveGate.Capabilities = append(dynakube.Spec.ActiveGate.Capabilities, dynatracev1beta1.KubeMonCapability.DisplayName)
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts := appsv1.StatefulSet{}

		builder.addTemplateSpec(&sts)
		spec := sts.Spec.Template.Spec
		assert.Contains(t, spec.ServiceAccountName, dynakube.ActiveGateServiceAccountName())
	})

	t.Run("set node selector", func(t *testing.T) {
		dynakube := getTestDynakube()
		testNodeSelector := map[string]string{
			"test": "test",
		}
		dynakube.Spec.ActiveGate.NodeSelector = testNodeSelector

		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts := appsv1.StatefulSet{}

		builder.addTemplateSpec(&sts)
		spec := sts.Spec.Template.Spec

		assert.Equal(t, testNodeSelector, spec.NodeSelector)
	})
	t.Run("set tolerations", func(t *testing.T) {
		dynakube := getTestDynakube()
		testTolerations := []corev1.Toleration{
			{
				Key:      "test",
				Operator: "test",
				Value:    "test",
				Effect:   "test",
			},
		}
		dynakube.Spec.ActiveGate.Tolerations = testTolerations
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts := appsv1.StatefulSet{}

		builder.addTemplateSpec(&sts)
		spec := sts.Spec.Template.Spec

		for _, toleration := range testTolerations {
			assert.Contains(t, spec.Tolerations, toleration)
		}
	})
	t.Run("set DNSPolicy", func(t *testing.T) {
		dynakube := getTestDynakube()
		testDNSPolicy := "test"
		dynakube.Spec.ActiveGate.DNSPolicy = corev1.DNSPolicy(testDNSPolicy)
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts := appsv1.StatefulSet{}

		builder.addTemplateSpec(&sts)
		spec := sts.Spec.Template.Spec
		assert.Equal(t, corev1.DNSPolicy(testDNSPolicy), spec.DNSPolicy)
	})
	t.Run("set priorityClass", func(t *testing.T) {
		dynakube := getTestDynakube()
		testPriorityClass := "test"
		dynakube.Spec.ActiveGate.PriorityClassName = testPriorityClass
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts := appsv1.StatefulSet{}

		builder.addTemplateSpec(&sts)
		spec := sts.Spec.Template.Spec

		assert.Equal(t, testPriorityClass, spec.PriorityClassName)
	})
	t.Run("default topologyConstraint", func(t *testing.T) {
		dynakube := getTestDynakube()

		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts, err := builder.CreateStatefulSet(nil)
		require.NoError(t, err)

		assert.Equal(t, builder.defaultTopologyConstraints(), sts.Spec.Template.Spec.TopologySpreadConstraints)
	})
	t.Run("set topologyConstraint", func(t *testing.T) {
		dynakube := getTestDynakube()
		testTopologyConstraint := []corev1.TopologySpreadConstraint{
			{
				TopologyKey: "test",
			},
		}
		dynakube.Spec.ActiveGate.TopologySpreadConstraints = testTopologyConstraint
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts, err := builder.CreateStatefulSet(nil)
		require.NoError(t, err)

		assert.Equal(t, testTopologyConstraint, sts.Spec.Template.Spec.TopologySpreadConstraints)
	})
	t.Run("default readinessProbe timeout is 2s", func(t *testing.T) {
		dynakube := getTestDynakube()
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts, err := builder.CreateStatefulSet(nil)
		require.NoError(t, err)

		assert.Equal(t, int32(2), sts.Spec.Template.Spec.Containers[0].ReadinessProbe.TimeoutSeconds)
	})
}

func TestBuildBaseContainer(t *testing.T) {
	t.Run("build container", func(t *testing.T) {
		dynakube := getTestDynakube()
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)

		containers := builder.buildBaseContainer()

		require.Len(t, containers, 1)
		container := containers[0]
		assert.Equal(t, dynakube.ActiveGateImage(), container.Image)
		assert.NotEmpty(t, container.Env)
		assert.NotNil(t, container.ReadinessProbe)
		assert.NotNil(t, container.SecurityContext)
	})
}

func TestBuildCommonEnvs(t *testing.T) {
	t.Run("build envs", func(t *testing.T) {
		dynakube := getTestDynakube()
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)

		envs := builder.buildCommonEnvs()

		require.NotEmpty(t, envs)
		capEnv := env.FindEnvVar(envs, consts.EnvDtCapabilities)
		require.NotNil(t, capEnv)
		assert.Equal(t, multiCapability.ArgName(), capEnv.Value)

		namespaceEnv := env.FindEnvVar(envs, consts.EnvDtIdSeedNamespace)
		require.NotNil(t, namespaceEnv)
		assert.Equal(t, dynakube.Namespace, namespaceEnv.Value)

		idEnv := env.FindEnvVar(envs, consts.EnvDtIdSeedClusterId)
		require.NotNil(t, idEnv)
		assert.Equal(t, testKubeUID, idEnv.Value)

		metadataEnv := env.FindEnvVar(envs, deploymentmetadata.EnvDtDeploymentMetadata)
		require.NotNil(t, metadataEnv)
		assert.NotEmpty(t, metadataEnv.ValueFrom.ConfigMapKeyRef)
		assert.Equal(t, deploymentmetadata.ActiveGateMetadataKey, metadataEnv.ValueFrom.ConfigMapKeyRef.Key)
		assert.Equal(t, deploymentmetadata.GetDeploymentMetadataConfigMapName(dynakube.Name), metadataEnv.ValueFrom.ConfigMapKeyRef.Name)

		// metrics-ingest disabled -> HTTP port disabled
		dtHttpPortEnv := env.FindEnvVar(envs, consts.EnvDtHttpPort)
		require.Nil(t, dtHttpPortEnv)
	})

	t.Run("adds extra envs with overrides", func(t *testing.T) {
		testEnvs := []corev1.EnvVar{
			{
				Name:  "test-env-key-1",
				Value: "test-env-value-1",
			},
			{
				Name:  "test-env-key-2",
				Value: "test-env-value-2",
			},
			{
				Name:  "DT_ID_SEED_NAMESPACE",
				Value: "ns-override",
			},
		}
		dynakube := getTestDynakube()
		dynakube.Spec.ActiveGate.Env = testEnvs
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)

		envs := builder.buildCommonEnvs()

		require.NotEmpty(t, envs)

		for _, env := range testEnvs {
			require.Contains(t, envs, env)
		}

		idx := slices.IndexFunc(envs, func(env corev1.EnvVar) bool {
			return env.Name == "DT_ID_SEED_NAMESPACE"
		})
		assert.Equal(t, "ns-override", envs[idx].Value)
	})

	t.Run("adds group env", func(t *testing.T) {
		testGroup := "test-group"
		dynakube := getTestDynakube()
		dynakube.Spec.ActiveGate.Group = testGroup
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)

		envs := builder.buildCommonEnvs()

		require.NotEmpty(t, envs)
		groupEnv := env.FindEnvVar(envs, consts.EnvDtGroup)
		require.NotNil(t, groupEnv)
		assert.Equal(t, multiCapability.Properties().Group, groupEnv.Value)
	})

	t.Run("metrics-ingest env", func(t *testing.T) {
		dynakube := getTestDynakube()

		activegate.SwitchCapability(&dynakube, dynatracev1beta1.RoutingCapability, false)
		activegate.SwitchCapability(&dynakube, dynatracev1beta1.MetricsIngestCapability, true)

		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)

		envs := builder.buildCommonEnvs()

		require.NotEmpty(t, envs)
		dtHttpPortEnv := env.FindEnvVar(envs, consts.EnvDtHttpPort)
		require.NotNil(t, dtHttpPortEnv)
		assert.Equal(t, strconv.Itoa(consts.HttpContainerPort), dtHttpPortEnv.Value)
	})

	t.Run("adds group env", func(t *testing.T) {
		testNetworkZone := "test-zone"
		dynakube := getTestDynakube()
		dynakube.Spec.NetworkZone = testNetworkZone
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)

		envs := builder.buildCommonEnvs()

		require.NotEmpty(t, envs)
		zoneEnv := env.FindEnvVar(envs, consts.EnvDtNetworkZone)
		require.NotNil(t, zoneEnv)
		assert.Equal(t, dynakube.Spec.NetworkZone, zoneEnv.Value)
	})
}

func TestSecurityContexts(t *testing.T) {
	t.Run("containers have the same security context if read-only filesystem", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Spec.ActiveGate.Capabilities = append(dynakube.Spec.ActiveGate.Capabilities, dynatracev1beta1.KubeMonCapability.DisplayName)

		multiCapability := capability.NewMultiCapability(&dynakube)

		statefulsetBuilder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts, _ := statefulsetBuilder.CreateStatefulSet([]builder.Modifier{
			modifiers.NewKubernetesMonitoringModifier(dynakube, multiCapability),
			modifiers.NewReadOnlyModifier(dynakube),
		})

		require.NotEmpty(t, sts)
		require.Truef(t, reflect.DeepEqual(sts.Spec.Template.Spec.InitContainers[0].SecurityContext, sts.Spec.Template.Spec.Containers[0].SecurityContext), "InitContainer and Container have different SecurityContexts")
	})
}
