package statefulset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testKubeUID       = "test-uid"
	testConfigHash    = "test-hash"
	testDynakubeName  = "test-dynakube"
	testNamespaceName = "test-namespace"
	testVersion       = "test-version"
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
				VersionStatus: dynatracev1beta1.VersionStatus{
					Version: testVersion,
				},
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
		appLabels := kubeobjects.NewAppLabels(kubeobjects.ActiveGateComponentLabel, builder.dynakube.Name, builder.capability.ShortName(), testVersion)
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
		appLabels := kubeobjects.NewAppLabels(kubeobjects.ActiveGateComponentLabel, builder.dynakube.Name, builder.capability.ShortName(), testVersion)
		expectedTemplateLabels := appLabels.BuildLabels()
		expectedTemplateLabels["test"] = "test"

		builder.addLabels(&sts)

		require.NotEmpty(t, sts.Spec.Template.Labels)
		assert.Equal(t, expectedTemplateLabels, sts.Spec.Template.Labels)
	})
	t.Run("use custom image", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Spec.ActiveGate.Image = "test"
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)
		sts := appsv1.StatefulSet{}
		appLabels := kubeobjects.NewAppLabels(kubeobjects.ActiveGateComponentLabel, builder.dynakube.Name, builder.capability.ShortName(), kubeobjects.CustomImageLabelValue)
		expectedLabels := appLabels.BuildLabels()

		builder.addLabels(&sts)

		require.NotEmpty(t, sts.ObjectMeta.Labels)
		assert.Equal(t, expectedLabels, sts.ObjectMeta.Labels)
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
		assert.Equal(t, dynakube.PullSecret(), spec.ImagePullSecrets[0].Name)
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
		sts := appsv1.StatefulSet{}

		builder.addTemplateSpec(&sts)
		spec := sts.Spec.Template.Spec
		assert.Equal(t, testTopologyConstraint, spec.TopologySpreadConstraints)
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
		capEnv := kubeobjects.FindEnvVar(envs, consts.EnvDtCapabilities)
		require.NotNil(t, capEnv)
		assert.Equal(t, multiCapability.ArgName(), capEnv.Value)
		namespaceEnv := kubeobjects.FindEnvVar(envs, consts.EnvDtIdSeedNamespace)
		require.NotNil(t, namespaceEnv)
		assert.Equal(t, dynakube.Namespace, namespaceEnv.Value)
		idEnv := kubeobjects.FindEnvVar(envs, consts.EnvDtIdSeedClusterId)
		require.NotNil(t, idEnv)
		assert.Equal(t, testKubeUID, idEnv.Value)
		metadataEnv := kubeobjects.FindEnvVar(envs, consts.EnvDtDeploymentMetadata)
		require.NotNil(t, metadataEnv)
		assert.NotEmpty(t, metadataEnv.Value)
	})

	t.Run("adds extra envs", func(t *testing.T) {
		testEnvs := []corev1.EnvVar{
			{
				Name:  "test-env-key-1",
				Value: "test-env-value-1",
			},
			{
				Name:  "test-env-key-2",
				Value: "test-env-value-2",
			},
		}
		dynakube := getTestDynakube()
		dynakube.Spec.ActiveGate.Env = testEnvs
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)

		envs := builder.buildCommonEnvs()

		require.NotEmpty(t, envs)
		for _, env := range testEnvs {
			assert.Contains(t, envs, env)
		}
	})

	t.Run("adds group env", func(t *testing.T) {
		testGroup := "test-group"
		dynakube := getTestDynakube()
		dynakube.Spec.ActiveGate.Group = testGroup
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)

		envs := builder.buildCommonEnvs()

		require.NotEmpty(t, envs)
		groupEnv := kubeobjects.FindEnvVar(envs, consts.EnvDtGroup)
		require.NotNil(t, groupEnv)
		assert.Equal(t, multiCapability.Properties().Group, groupEnv.Value)
	})

	t.Run("adds group env", func(t *testing.T) {
		testNetworkZone := "test-zone"
		dynakube := getTestDynakube()
		dynakube.Spec.NetworkZone = testNetworkZone
		multiCapability := capability.NewMultiCapability(&dynakube)
		builder := NewStatefulSetBuilder(testKubeUID, testConfigHash, dynakube, multiCapability)

		envs := builder.buildCommonEnvs()

		require.NotEmpty(t, envs)
		zoneEnv := kubeobjects.FindEnvVar(envs, consts.EnvDtNetworkZone)
		require.NotNil(t, zoneEnv)
		assert.Equal(t, dynakube.Spec.NetworkZone, zoneEnv.Value)
	})
}
