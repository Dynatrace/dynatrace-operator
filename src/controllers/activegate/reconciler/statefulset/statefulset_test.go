package statefulset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/customproperties"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/version"
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
	stsProps := statefulSetProperties{
		DynaKube:             instance,
		CapabilityProperties: capabilityProperties,
		feature:              testFeature,
	}
	sts, err := CreateStatefulSet(NewStatefulSetProperties(instance, capabilityProperties, "", "", testFeature, "", "", nil, nil, nil))

	assert.NoError(t, err)
	assert.NotNil(t, sts)
	assert.Equal(t, instance.Name+routingStatefulSetSuffix, sts.Name)
	assert.Equal(t, instance.Namespace, sts.Namespace)
	assert.Equal(t, map[string]string{
		kubeobjects.AppComponentLabel: ActiveGateComponentName,
		kubeobjects.AppCreatedByLabel: instance.Name,
		kubeobjects.FeatureLabel:      testFeature,
		kubeobjects.AppNameLabel:      version.AppName,
		kubeobjects.AppVersionLabel:   version.Version,
	}, sts.Labels)
	assert.Equal(t, instance.Spec.ActiveGate.Replicas, sts.Spec.Replicas)
	assert.Equal(t, appsv1.ParallelPodManagement, sts.Spec.PodManagementPolicy)
	assert.Equal(t, metav1.LabelSelector{
		MatchLabels: stsProps.buildMatchLabels(),
	}, *sts.Spec.Selector)
	assert.NotEqual(t, corev1.PodTemplateSpec{}, sts.Spec.Template)
	assert.Equal(t, stsProps.buildLabels(), sts.Spec.Template.Labels)
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
	checkCoreProperties := func(instance *dynatracev1beta1.DynaKube, templateSpec *corev1.PodSpec) {
		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties

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

		assert.NotEmpty(t, templateSpec.ImagePullSecrets)
		assert.Contains(t, templateSpec.ImagePullSecrets, corev1.LocalObjectReference{Name: instance.Name + dtpullsecret.PullSecretSuffix})
	}

	checkVolumeMounts := func(expected bool, templateSpec *corev1.PodSpec) {
		assert.Equalf(t, expected, kubeobjects.VolumeIsDefined(templateSpec.Volumes, dataSourceMetadata),
			"Expected that volume mount %s has a predefined pod volume", dataSourceMetadata,
		)
		assert.Equalf(t, expected, kubeobjects.VolumeIsDefined(templateSpec.Volumes, eecLogs),
			"Expected that volume mount %s has a predefined pod volume", eecLogs,
		)
		assert.Equalf(t, expected, kubeobjects.VolumeIsDefined(templateSpec.Volumes, dataSourceStatsdLogs),
			"Expected that volume mount %s has a predefined pod volume", dataSourceStatsdLogs,
		)
	}

	t.Run("DynaKube with PriorityClassName set", func(t *testing.T) {
		const customPriorityClassName = "custom-priority-class"
		instance := buildTestInstance()
		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties

		instance.Spec.ActiveGate.PriorityClassName = customPriorityClassName
		templateSpec := buildTemplateSpec(NewStatefulSetProperties(instance, capabilityProperties, "", "", "test-feature", "", "", nil, nil, nil))
		assert.Equal(t, customPriorityClassName, templateSpec.PriorityClassName)
	})

	t.Run("DynaKube with PriorityClassName empty", func(t *testing.T) {
		instance := buildTestInstance()
		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties

		templateSpec := buildTemplateSpec(NewStatefulSetProperties(instance, capabilityProperties, "", "", "test-feature", "", "", nil, nil, nil))
		assert.Equal(t, "", templateSpec.PriorityClassName)
	})

	t.Run("DynaKube with TopologySpreadConstraints set", func(t *testing.T) {
		instance := buildTestInstance()
		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties

		tsc := corev1.TopologySpreadConstraint{
			MaxSkew:           1,
			TopologyKey:       "",
			WhenUnsatisfiable: "",
			LabelSelector:     nil,
		}
		expected := []corev1.TopologySpreadConstraint{tsc, tsc, tsc}

		instance.Spec.ActiveGate.TopologySpreadConstraints = expected
		templateSpec := buildTemplateSpec(NewStatefulSetProperties(instance, capabilityProperties, "", "", "test-feature", "", "", nil, nil, nil))
		assert.Equal(t, len(expected), len(templateSpec.TopologySpreadConstraints))
	})

	t.Run("DynaKube with TopologySpreadConstraints empty", func(t *testing.T) {
		instance := buildTestInstance()
		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties

		templateSpec := buildTemplateSpec(NewStatefulSetProperties(instance, capabilityProperties, "", "", "test-feature", "", "", nil, nil, nil))
		assert.Nil(t, templateSpec.TopologySpreadConstraints)
	})

	t.Run("DynaKube without StatsD", func(t *testing.T) {
		instance := buildTestInstance()
		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
		templateSpec := buildTemplateSpec(NewStatefulSetProperties(instance, capabilityProperties,
			"", "", "test-feature", "", "", nil, nil, nil))

		checkCoreProperties(instance, &templateSpec)

		assert.False(t, instance.NeedsStatsd())
		checkVolumeMounts(false, &templateSpec)
	})

	t.Run("DynaKube with StatsD enabled", func(t *testing.T) {
		instance := buildTestInstance()
		if !instance.NeedsStatsd() {
			instance.Spec.ActiveGate.Capabilities = append(instance.Spec.ActiveGate.Capabilities,
				dynatracev1beta1.StatsdIngestCapability.DisplayName,
			)
		}

		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
		templateSpec := buildTemplateSpec(NewStatefulSetProperties(instance, capabilityProperties,
			"", "", "test-feature", "", "", nil, nil, nil))

		checkCoreProperties(instance, &templateSpec)

		assert.True(t, instance.NeedsStatsd())
		checkVolumeMounts(true, &templateSpec)
	})
}

func TestStatefulSet_Container(t *testing.T) {
	checkCoreProperties := func(activeGateContainer *corev1.Container, dynakube *dynatracev1beta1.DynaKube) {
		assert.Equal(t, capability.ActiveGateContainerName, activeGateContainer.Name)
		assert.Equal(t, dynakube.ActiveGateImage(), activeGateContainer.Image)
		assert.Empty(t, activeGateContainer.Resources)
		assert.Equal(t, corev1.PullAlways, activeGateContainer.ImagePullPolicy)
		assert.NotEmpty(t, activeGateContainer.Env)
		assert.Empty(t, activeGateContainer.Args)
	}

	checkSecurityProperties := func(activeGateContainer *corev1.Container, dynakube *dynatracev1beta1.DynaKube) {
		assert.Equal(t, *activeGateContainer.SecurityContext.Privileged, false)
		assert.Equal(t, *activeGateContainer.SecurityContext.AllowPrivilegeEscalation, false)
		assert.Equal(t, *activeGateContainer.SecurityContext.ReadOnlyRootFilesystem, dynakube.FeatureActiveGateReadOnlyFilesystem())
		assert.Equal(t, *activeGateContainer.SecurityContext.RunAsNonRoot, true)
		assert.Equal(t, activeGateContainer.SecurityContext.SeccompProfile.Type, corev1.SeccompProfileTypeRuntimeDefault)
		assert.Equal(t, len(activeGateContainer.SecurityContext.Capabilities.Drop), 1)
		assert.Equal(t, activeGateContainer.SecurityContext.Capabilities.Drop[0], corev1.Capability("all"))
	}

	checkVolumes := func(activeGateContainer *corev1.Container, dynakube *dynatracev1beta1.DynaKube) {
		for _, directory := range buildActiveGateMountPoints(dynakube.NeedsStatsd(), dynakube.FeatureActiveGateReadOnlyFilesystem()) {
			assert.Truef(t, kubeobjects.MountPathIsIn(activeGateContainer.VolumeMounts, directory),
				"Expected that ActiveGate container defines mount point %s", directory,
			)
			assert.Truef(t, kubeobjects.MountPathIsReadOnlyOrReadWrite(activeGateContainer.VolumeMounts, directory, kubeobjects.ReadWriteMountPath),
				"Expected that ActiveGate container mount point %s is mounted ReadWrite", directory,
			)
		}

		assert.Equalf(t, dynakube.NeedsStatsd(), kubeobjects.MountPathIsIn(activeGateContainer.VolumeMounts, extensionsLogsDir+"/eec"),
			"Expected that ActiveGate container defines mount point %s if and only if StatsD ingest is enabled", extensionsLogsDir+"/eec",
		)
		assert.Equalf(t, dynakube.NeedsStatsd(), kubeobjects.MountPathIsIn(activeGateContainer.VolumeMounts, extensionsLogsDir+"/statsd"),
			"Expected that ActiveGate container defines mount point %s if and only if StatsD ingest is enabled", extensionsLogsDir+"/statsd",
		)
	}

	checkAnnotations := func(sts *appsv1.StatefulSet, dynakube *dynatracev1beta1.DynaKube) {
		if dynakube.FeatureActiveGateAppArmor() {
			assert.Truef(t, sts.Spec.Template.ObjectMeta.Annotations[annotationActiveGateContainerAppArmor] == "runtime/default",
				"'%s' is invalid (%s)", annotationActiveGateContainerAppArmor, sts.Spec.Template.ObjectMeta.Annotations[annotationActiveGateContainerAppArmor])
		} else {
			_, ok := sts.Spec.Template.ObjectMeta.Annotations[annotationActiveGateContainerAppArmor]
			assert.Falsef(t, ok, "'%s'found)", annotationActiveGateContainerAppArmor)
		}
	}

	test := func(ro bool, statsd bool) {
		instance := buildTestInstance()
		if ro {
			instance.Annotations[dynatracev1beta1.AnnotationFeatureActiveGateReadOnlyFilesystem] = "true"
		}
		if statsd {
			instance.Spec.ActiveGate.Capabilities = append(instance.Spec.ActiveGate.Capabilities, dynatracev1beta1.StatsdIngestCapability.DisplayName)
		}
		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
		stsProperties := NewStatefulSetProperties(instance, capabilityProperties,
			"", "", "", "", "", nil, nil, nil)
		sts, err := CreateStatefulSet(stsProperties)
		assert.Nil(t, err)
		extraContainerBuilders := getContainerBuilders(stsProperties)
		containers := buildContainers(stsProperties, extraContainerBuilders)
		activeGateContainer := containers[0]

		checkCoreProperties(&activeGateContainer, instance)
		checkSecurityProperties(&activeGateContainer, instance)
		checkVolumes(&activeGateContainer, instance)
		checkAnnotations(sts, instance)
		assert.NotNil(t, activeGateContainer.ReadinessProbe)
	}

	t.Run("DynaKube with RW filesystem and StatsD disabled", func(t *testing.T) {
		test(false, false)
	})

	t.Run("DynaKube with RW filesystem and StatsD enabled", func(t *testing.T) {
		test(false, true)
	})

	t.Run("DynaKube with RO filesystem and StatsD disabled", func(t *testing.T) {
		test(true, false)
	})

	t.Run("DynaKube with RO filesystem and StatsD enabled", func(t *testing.T) {
		test(true, true)
	})

	t.Run("DynaKube with AppArmor enabled", func(t *testing.T) {
		instance := buildTestInstance()
		instance.Annotations[dynatracev1beta1.AnnotationFeatureActiveGateAppArmor] = "true"
		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
		stsProperties := NewStatefulSetProperties(instance, capabilityProperties,
			"", "", "", "", "", nil, nil, nil)
		sts, err := CreateStatefulSet(stsProperties)
		assert.Nil(t, err)
		extraContainerBuilders := getContainerBuilders(stsProperties)
		containers := buildContainers(stsProperties, extraContainerBuilders)
		activeGateContainer := containers[0]

		checkAnnotations(sts, instance)
		assert.NotNil(t, activeGateContainer.ReadinessProbe)
	})
}

func TestStatefulSet_Volumes(t *testing.T) {
	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties

	t.Run(`without custom properties`, func(t *testing.T) {
		stsProperties := NewStatefulSetProperties(instance, capabilityProperties,
			"", "", "", "", "",
			nil, nil, nil,
		)
		volumes := buildVolumes(stsProperties, getContainerBuilders(stsProperties))
		assert.Falsef(t, kubeobjects.VolumeIsDefined(volumes, customproperties.VolumeName),
			"Expected that volume %s is not defined if there are no custom properties", customproperties.VolumeName,
		)
	})
	t.Run(`with FeatureDisableActivegateRawImage=false`, func(t *testing.T) {
		instanceRawImg := instance.DeepCopy()
		instanceRawImg.Annotations[dynatracev1beta1.AnnotationFeatureDisableActiveGateRawImage] = "false"

		stsProperties := NewStatefulSetProperties(instanceRawImg, capabilityProperties,
			"", "", "", "", "",
			nil, nil, nil,
		)
		volumes := buildVolumes(stsProperties, getContainerBuilders(stsProperties))

		require.Equal(t, 1, len(volumes))

		assert.Falsef(t, kubeobjects.VolumeIsDefined(volumes, customproperties.VolumeName),
			"Expected that volume %s is not defined if there are no custom properties", customproperties.VolumeName,
		)
	})
	t.Run(`custom properties from value string`, func(t *testing.T) {
		capabilityProperties.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{
			Value: testValue,
		}
		stsProperties := NewStatefulSetProperties(instance, capabilityProperties,
			"", "", testFeature, "", "",
			nil, nil, nil,
		)
		volumes := buildVolumes(stsProperties, getContainerBuilders(stsProperties))
		expectedSecretName := instance.Name + "-router-" + customproperties.Suffix

		require.Equal(t, 2, len(volumes))

		_, err := kubeobjects.GetVolumeByName(volumes, tenantSecretVolumeName)
		assert.NoError(t, err)

		customPropertiesVolume, err := kubeobjects.GetVolumeByName(volumes, customproperties.VolumeName)
		assert.NoError(t, err)
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
		stsProperties := NewStatefulSetProperties(instance, capabilityProperties,
			"", "", "", "", "",
			nil, nil, nil,
		)
		volumes := buildVolumes(stsProperties, getContainerBuilders(stsProperties))
		expectedSecretName := testKey

		require.Equal(t, 2, len(volumes))

		_, err := kubeobjects.GetVolumeByName(volumes, tenantSecretVolumeName)
		assert.NoError(t, err)

		customPropertiesVolume, err := kubeobjects.GetVolumeByName(volumes, customproperties.VolumeName)
		assert.NoError(t, err)
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
	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(testUID, DeploymentTypeActiveGate)

	t.Run(`with FeatureDisableActivegateRawImage=true`, func(t *testing.T) {
		instanceRawImg := instance.DeepCopy()
		instanceRawImg.Annotations[dynatracev1beta1.AnnotationFeatureDisableActiveGateRawImage] = "true"

		envVars := buildEnvs(NewStatefulSetProperties(instanceRawImg, capabilityProperties,
			testUID, "", testFeature, "MSGrouter", "",
			nil, nil, nil,
		))

		expectedEnvVars := []corev1.EnvVar{
			{Name: dtCapabilities, Value: "MSGrouter"},
			{Name: dtIdSeedNamespace, Value: instanceRawImg.Namespace},
			{Name: dtIdSeedClusterId, Value: testUID},
			{Name: dtDeploymentMetadata, Value: deploymentMetadata.AsString()},
			{Name: testKey, Value: testValue},
		}

		assert.ElementsMatch(t, expectedEnvVars, envVars)

	})
	t.Run(`without proxy`, func(t *testing.T) {
		envVars := buildEnvs(NewStatefulSetProperties(instance, capabilityProperties,
			testUID, "", testFeature, "MSGrouter", "",
			nil, nil, nil,
		))

		expectedEnvVars := []corev1.EnvVar{
			{
				Name: dtServer,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: instance.Name + dynatracev1beta1.TenantSecretSuffix,
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
							Name: instance.Name + dynatracev1beta1.TenantSecretSuffix,
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
		}

		assert.ElementsMatch(t, expectedEnvVars, envVars)
	})
	t.Run(`with networkzone`, func(t *testing.T) {
		instance := buildTestInstance()
		instance.Spec.NetworkZone = testName
		capabilityProperties := &instance.Spec.ActiveGate.CapabilityProperties
		envVars := buildEnvs(NewStatefulSetProperties(instance, capabilityProperties,
			"", "", "", "", "",
			nil, nil, nil,
		))

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
		envVars := buildEnvs(NewStatefulSetProperties(instance, capabilityProperties,
			"", "", "", "", "",
			nil, nil, nil,
		))

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
		volumeMounts := buildVolumeMounts(NewStatefulSetProperties(instance, capabilityProperties,
			"", "", "", "", "",
			nil, nil, nil,
		))
		assert.Falsef(t, kubeobjects.MountPathIsIn(volumeMounts, customproperties.MountPath),
			"Expected that there is no mount point %s if there are no custom properties", customproperties.MountPath,
		)
	})
	t.Run(`with custom properties`, func(t *testing.T) {
		capabilityProperties.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{Value: testValue}
		volumeMounts := buildVolumeMounts(NewStatefulSetProperties(instance, capabilityProperties,
			"", "", "", "", "",
			nil, nil, nil,
		))

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

	container := buildActiveGateContainer(NewStatefulSetProperties(instance, capabilityProperties,
		"", "", "", "", "",
		nil, nil, nil,
	))

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
			Name:        testName,
			Namespace:   testNamespace,
			Annotations: make(map[string]string),
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://testing.dev.dynatracelabs.com/api",
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

func buildActiveGateMountPoints(statsd bool, readOnly bool) []string {
	var mountPoints []string
	if readOnly || statsd {
		mountPoints = append(mountPoints, capability.ActiveGateGatewayConfigMountPoint)
	}
	if readOnly {
		mountPoints = append(mountPoints,
			capability.ActiveGateGatewayTempMountPoint,
			capability.ActiveGateGatewayDataMountPoint,
			capability.ActiveGateLogMountPoint,
			capability.ActiveGateTmpMountPoint)
	}
	return mountPoints
}
