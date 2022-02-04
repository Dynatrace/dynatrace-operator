package statefulset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/customproperties"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName                 = "test-name"
	testNamespace            = "test-namespace"
	testKey                  = "test-key"
	testValue                = "test-value"
	testUID                  = "test-uid"
	routingStatefulSetSuffix = "-router"
	testFeature              = "router"
	testDNSPolicy            = corev1.DNSPolicy("dns")
)

func TestNewStatefulSetBuilder(t *testing.T) {
	stsBuilder := NewStatefulSetProperties(&dynatracev1beta1.DynaKube{}, &dynatracev1beta1.CapabilityProperties{}, testUID, testValue, "", "", "", nil, nil, nil)
	assert.NotNil(t, stsBuilder)
	assert.NotNil(t, stsBuilder.DynaKube)
	assert.NotNil(t, stsBuilder.CapabilityProperties)
	assert.NotNil(t, stsBuilder.customPropertiesHash)
	assert.NotEmpty(t, stsBuilder.customPropertiesHash)
	assert.NotEmpty(t, stsBuilder.kubeSystemUID)
}

func TestStatefulSetBuilder_Build(t *testing.T) {
	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
	sts, err := CreateStatefulSet(NewStatefulSetProperties(instance, capabilityProperties, "", "", testFeature, "", "", nil, nil, nil))

	assert.NoError(t, err)
	assert.NotNil(t, sts)
	assert.Equal(t, instance.Name+routingStatefulSetSuffix, sts.Name)
	assert.Equal(t, instance.Namespace, sts.Namespace)
	assert.Equal(t, map[string]string{
		KeyDynatrace:  ValueActiveGate,
		KeyActiveGate: instance.Name,
		KeyFeature:    testFeature,
	}, sts.Labels)
	assert.Equal(t, instance.Spec.ActiveGate.Replicas, sts.Spec.Replicas)
	assert.Equal(t, appsv1.ParallelPodManagement, sts.Spec.PodManagementPolicy)
	assert.Equal(t, metav1.LabelSelector{
		MatchLabels: BuildLabelsFromInstance(instance, testFeature),
	}, *sts.Spec.Selector)
	assert.NotEqual(t, corev1.PodTemplateSpec{}, sts.Spec.Template)
	assert.Equal(t, buildLabels(instance, testFeature, capabilityProperties), sts.Spec.Template.Labels)
	assert.Equal(t, sts.Labels, sts.Spec.Template.Labels)
	assert.NotEqual(t, corev1.PodSpec{}, sts.Spec.Template.Spec)
	assert.Contains(t, sts.Annotations, kubeobjects.AnnotationHash)

	storedHash := sts.Annotations[kubeobjects.AnnotationHash]
	sts.Annotations = map[string]string{}
	hash, err := kubeobjects.GenerateHash(sts)
	assert.NoError(t, err)
	assert.Equal(t, storedHash, hash)

	t.Run(`template has annotations`, func(t *testing.T) {
		sts, _ := CreateStatefulSet(NewStatefulSetProperties(instance, capabilityProperties, "", testValue, "", "", "", nil, nil, nil))
		assert.Equal(t, map[string]string{
			annotationVersion:         instance.Status.ActiveGate.Version,
			annotationCustomPropsHash: testValue,
		}, sts.Spec.Template.Annotations)
	})
}

func TestStatefulSet_TemplateSpec(t *testing.T) {
	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
	templateSpec := buildTemplateSpec(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))

	assert.NotEqual(t, corev1.PodSpec{}, templateSpec)
	assert.NotEmpty(t, templateSpec.Containers)
	assert.Equal(t, capabilityProperties.NodeSelector, templateSpec.NodeSelector)

	assert.NotEmpty(t, templateSpec.Affinity)
	assert.NotEmpty(t, templateSpec.Affinity.NodeAffinity)
	assert.NotEmpty(t, templateSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
	assert.NotEmpty(t, templateSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)
	assert.Contains(t, templateSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
		corev1.NodeSelectorTerm{MatchExpressions: kubeobjects.AffinityNodeRequirement()})

	assert.Equal(t, capabilityProperties.Tolerations, templateSpec.Tolerations)
	assert.Equal(t, 1, len(templateSpec.Volumes))
	assert.Contains(t, templateSpec.Volumes, corev1.Volume{
		Name: tenantSecretVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: instance.Name + activegate.AGSecretSuffix,
			},
		},
	})
	assert.Equal(t, 1, len(templateSpec.ImagePullSecrets))
	assert.Contains(t, templateSpec.ImagePullSecrets, corev1.LocalObjectReference{Name: instance.Name + dtpullsecret.PullSecretSuffix})
}

func TestStatefulSet_Container(t *testing.T) {
	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
	container := buildContainer(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))

	assert.Equal(t, activegateContainerName, container.Name)
	assert.Equal(t, instance.ActiveGateImage(), container.Image)
	assert.Empty(t, container.Resources)
	assert.Equal(t, corev1.PullAlways, container.ImagePullPolicy)
	assert.NotEmpty(t, container.Env)
	assert.Empty(t, container.Args)
	assert.Equal(t, 1, len(container.VolumeMounts))
	assert.NotNil(t, container.ReadinessProbe)
}

func TestStatefulSet_Volumes(t *testing.T) {
	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties

	t.Run(`without custom properties`, func(t *testing.T) {
		volumes := buildVolumes(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))

		assert.Equal(t, 1, len(volumes))
	})
	t.Run(`custom properties from value string`, func(t *testing.T) {
		capabilityProperties.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{
			Value: testValue,
		}
		volumes := buildVolumes(NewStatefulSetProperties(instance, capabilityProperties, "", "", testFeature, "", "", nil, nil, nil))
		expectedSecretName := instance.Name + "-router-" + customproperties.Suffix

		require.Equal(t, 2, len(volumes))

		customPropertiesVolume := volumes[1]
		assert.Equal(t, customproperties.VolumeName, customPropertiesVolume.Name)
		assert.NotNil(t, customPropertiesVolume.VolumeSource)
		assert.NotNil(t, customPropertiesVolume.VolumeSource.Secret)
		assert.Equal(t, expectedSecretName, customPropertiesVolume.Secret.SecretName)
		assert.Equal(t, []corev1.KeyToPath{
			{Key: customproperties.DataKey, Path: customproperties.DataPath},
		}, customPropertiesVolume.Secret.Items)
	})
	t.Run(`custom properties from valueFrom`, func(t *testing.T) {
		capabilityProperties.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{
			ValueFrom: testKey,
		}
		volumes := buildVolumes(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))
		expectedSecretName := testKey

		require.Equal(t, 2, len(volumes))

		customPropertiesVolume := volumes[1]
		assert.Equal(t, customproperties.VolumeName, customPropertiesVolume.Name)
		assert.NotNil(t, customPropertiesVolume.VolumeSource)
		assert.NotNil(t, customPropertiesVolume.VolumeSource.Secret)
		assert.Equal(t, expectedSecretName, customPropertiesVolume.Secret.SecretName)
		assert.Equal(t, []corev1.KeyToPath{
			{Key: customproperties.DataKey, Path: customproperties.DataPath},
		}, customPropertiesVolume.Secret.Items)
	})
}

func TestStatefulSet_Env(t *testing.T) {
	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(testUID, deploymentmetadata.DeploymentTypeActiveGate)

	t.Run(`without proxy`, func(t *testing.T) {
		envVars := buildEnvs(NewStatefulSetProperties(instance, capabilityProperties, testUID, "", testFeature, "MSGrouter", "", nil, nil, nil))
		assert.Equal(t, 7, len(envVars))
		assert.Equal(t, envVars, []corev1.EnvVar{
			{
				Name: dtServer,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: instance.Name + dynatracev1beta1.AGTenantSecretSuffix,
						},
						Key: activegate.CommunicationEndpointsName,
					},
				},
			},
			{
				Name: dtTenant,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: instance.Name + dynatracev1beta1.AGTenantSecretSuffix,
						},
						Key: activegate.TenantUuidName,
					},
				},
			},
			{Name: dtCapabilities, Value: "MSGrouter"},
			{Name: dtIdSeedNamespace, Value: instance.Namespace},
			{Name: dtIdSeedClusterId, Value: testUID},
			{Name: dtDeploymentMetadata, Value: deploymentMetadata.AsString()},
			{Name: testKey, Value: testValue},
		})
	})
	t.Run(`with networkzone`, func(t *testing.T) {
		instance := buildTestInstance()
		instance.Spec.NetworkZone = testName
		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
		envVars := buildEnvs(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))

		assert.NotEmpty(t, envVars)

		assert.Contains(t, envVars, corev1.EnvVar{
			Name:  dtNetworkZone,
			Value: testName,
		})
	})
	t.Run(`with group`, func(t *testing.T) {
		instance := buildTestInstance()
		instance.Spec.ActiveGate.Group = testValue
		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
		envVars := buildEnvs(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))

		assert.NotEmpty(t, envVars)

		assert.Contains(t, envVars, corev1.EnvVar{
			Name:  dtGroup,
			Value: testValue,
		})
	})
}

func TestStatefulSet_VolumeMounts(t *testing.T) {
	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties

	t.Run(`without custom properties`, func(t *testing.T) {
		volumeMounts := buildVolumeMounts(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))
		assert.Equal(t, 1, len(volumeMounts))
	})
	t.Run(`with custom properties`, func(t *testing.T) {
		capabilityProperties.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{Value: testValue}
		volumeMounts := buildVolumeMounts(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))

		assert.NotEmpty(t, volumeMounts)
		assert.Contains(t, volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      customproperties.VolumeName,
			MountPath: customproperties.MountPath,
			SubPath:   customproperties.DataPath,
		})
	})
	t.Run(`with proxy from value`, func(t *testing.T) {
		instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		volumeMounts := buildVolumeMounts(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))

		assert.Contains(t, volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretHostMountPath,
			SubPath:   InternalProxySecretHost,
		})

		assert.Contains(t, volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretPortMountPath,
			SubPath:   InternalProxySecretPort,
		})

		assert.Contains(t, volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretUsernameMountPath,
			SubPath:   InternalProxySecretUsername,
		})

		assert.Contains(t, volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretPasswordMountPath,
			SubPath:   InternalProxySecretPassword,
		})
	})
	t.Run(`with proxy from value source`, func(t *testing.T) {
		instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{ValueFrom: testName}
		volumeMounts := buildVolumeMounts(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))

		assert.Contains(t, volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretHostMountPath,
			SubPath:   InternalProxySecretHost,
		})

		assert.Contains(t, volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretPortMountPath,
			SubPath:   InternalProxySecretPort,
		})

		assert.Contains(t, volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretUsernameMountPath,
			SubPath:   InternalProxySecretUsername,
		})

		assert.Contains(t, volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      InternalProxySecretVolumeName,
			MountPath: InternalProxySecretPasswordMountPath,
			SubPath:   InternalProxySecretPassword,
		})
	})
}

func TestStatefulSet_Resources(t *testing.T) {
	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties

	quantityCpuLimit := resource.NewScaledQuantity(700, resource.Milli)
	quantityMemoryLimit := resource.NewScaledQuantity(7, resource.Giga)
	quantityCpuRequest := resource.NewScaledQuantity(500, resource.Milli)
	quantityMemoryRequest := resource.NewScaledQuantity(5, resource.Giga)

	instance.Spec.ActiveGate.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    *quantityCpuLimit,
			corev1.ResourceMemory: *quantityMemoryLimit,
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    *quantityCpuRequest,
			corev1.ResourceMemory: *quantityMemoryRequest,
		},
	}

	container := buildContainer(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))

	assert.True(t, quantityCpuLimit.Equal(container.Resources.Limits[corev1.ResourceCPU]))
	assert.True(t, quantityMemoryLimit.Equal(container.Resources.Limits[corev1.ResourceMemory]))
	assert.True(t, quantityCpuRequest.Equal(container.Resources.Requests[corev1.ResourceCPU]))
	assert.True(t, quantityMemoryRequest.Equal(container.Resources.Requests[corev1.ResourceMemory]))
}

func TestStatefulSet_DNSPolicy(t *testing.T) {
	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties

	podSpec := buildTemplateSpec(NewStatefulSetProperties(instance, capabilityProperties, "", "", "", "", "", nil, nil, nil))

	assert.Equal(t, testDNSPolicy, podSpec.DNSPolicy)
}

func buildTestInstance() *dynatracev1beta1.DynaKube {
	replicas := int32(3)
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.RoutingCapability.DisplayName,
				},
				DNSPolicy: testDNSPolicy,
				CapabilityProperties: dynatracev1beta1.CapabilityProperties{
					Replicas:    &replicas,
					Tolerations: []corev1.Toleration{{Value: testValue}},
					NodeSelector: map[string]string{
						testKey: testValue,
					},
					Env: []corev1.EnvVar{
						{Name: testKey, Value: testValue},
					},
				},
			},
		},
	}
}
