package kubemon

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const (
	testUID       = "test-uid"
	testId        = "test-id"
	testKey       = "key"
	testValue     = "value"
	testValueFrom = "valueFrom"
	testClass     = "test-class"
	testName      = "test-name"
	testNamespace = "test-namespace"
	testToken     = "test-token"
	testEndpoint  = "http://test-endpoint"
)

func TestNewStatefulSet(t *testing.T) {
	instance := v1alpha1.DynaKube{}
	tenantInfo := dtclient.TenantInfo{}

	expectedObjectMeta := buildObjectMeta(&instance)
	expectedSpecs := buildSpec(&instance, &tenantInfo, testUID)

	sts := newStatefulSet(instance, &tenantInfo, testUID)
	assert.NotNil(t, sts)
	assert.Equal(t, expectedObjectMeta, sts.ObjectMeta)
	assert.Equal(t, expectedSpecs, sts.Spec)
}

func TestBuildObjectMeta(t *testing.T) {
	const instanceNamespace = "test-namespace"
	instance := &v1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instanceNamespace,
		}}

	stsObjectMeta := buildObjectMeta(instance)
	assert.NotNil(t, stsObjectMeta)
	assert.Equal(t, v1alpha1.Name, stsObjectMeta.Name)
	assert.Equal(t, instanceNamespace, stsObjectMeta.Namespace)
	assert.Equal(t, buildLabels(instance), stsObjectMeta.Labels)
	assert.Empty(t, stsObjectMeta.Annotations)
}

func TestBuildSpec(t *testing.T) {
	var replicas int32 = 2
	tenantInfo := &dtclient.TenantInfo{}
	instance := &v1alpha1.DynaKube{
		Spec: v1alpha1.DynaKubeSpec{
			KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
				Replicas: &replicas,
			},
		},
	}

	specs := buildSpec(instance, tenantInfo, testUID)

	assert.NotNil(t, specs)
	assert.Equal(t, replicas, *specs.Replicas)
	assert.Equal(t, buildLabelSelector(instance), specs.Selector)
	assert.Equal(t, buildTemplate(instance, tenantInfo, testUID), specs.Template)
}

func TestBuildLabels(t *testing.T) {
	const testName = "test-instance"
	instance := &v1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
			Labels: map[string]string{
				testKey: testValue,
			},
		},
	}
	labels := buildLabels(instance)

	assert.NotEmpty(t, labels)
	assert.Equal(t, map[string]string{
		"dynatrace":  "activegate",
		"activegate": testName,
		testKey:      testValue,
	}, labels)
}

func TestBuildLabelSelector(t *testing.T) {
	const testName = "test-instance"
	instance := &v1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: testName,
			Labels: map[string]string{
				testKey: testValue,
			},
		},
	}
	expectedLabelSelector := metav1.LabelSelector{
		MatchLabels: BuildLabelsFromInstance(instance),
	}
	labelSelector := buildLabelSelector(instance)

	assert.NotNil(t, labelSelector)
	assert.Equal(t, expectedLabelSelector, *labelSelector)
}

func TestBuildTemplate(t *testing.T) {
	tenantInfo := &dtclient.TenantInfo{}
	instance := &v1alpha1.DynaKube{}
	template := buildTemplate(instance, tenantInfo, testUID)

	assert.NotNil(t, template)
	assert.NotEmpty(t, template.Labels)
	assert.Equal(t, map[string]string{
		"dynatrace":  "activegate",
		"activegate": "",
	}, template.Labels)
	assert.Equal(t, buildTemplateSpec(instance, tenantInfo, testUID), template.Spec)
}

func TestBuildTemplateSpec(t *testing.T) {
	t.Run(`BuildTemplateSpec with default instance`, func(t *testing.T) {
		tenantInfo := &dtclient.TenantInfo{}
		instance := &v1alpha1.DynaKube{}
		templateSpec := buildTemplateSpec(instance, tenantInfo, testUID)

		assert.NotNil(t, templateSpec)
		assert.NotEmpty(t, templateSpec.Containers)
		assert.Equal(t, []corev1.Container{buildContainer(instance, tenantInfo, testUID)}, templateSpec.Containers)
		assert.Equal(t, corev1.DNSPolicy(""), templateSpec.DNSPolicy)
		assert.Empty(t, templateSpec.NodeSelector)
		assert.Equal(t, buildServiceAccountName(instance), templateSpec.ServiceAccountName)
		assert.Empty(t, templateSpec.Tolerations)
		assert.Equal(t, "", templateSpec.PriorityClassName)
		assert.Equal(t, buildVolumes(instance), templateSpec.Volumes)
		assert.Equal(t, buildImagePullSecrets(instance), templateSpec.ImagePullSecrets)
	})
	t.Run(`BuildTemplateSpec with values`, func(t *testing.T) {
		tenantInfo := &dtclient.TenantInfo{}
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					DNSPolicy: corev1.DNSClusterFirst,
					NodeSelector: map[string]string{
						testKey: testValue,
					},
					Tolerations: []corev1.Toleration{
						{Key: testKey, Value: testValue},
					},
					PriorityClassName: testClass,
				},
			},
		}
		templateSpec := buildTemplateSpec(instance, tenantInfo, testUID)

		assert.NotNil(t, templateSpec)
		assert.Equal(t, corev1.DNSClusterFirst, templateSpec.DNSPolicy)
		assert.Equal(t, map[string]string{testKey: testValue}, templateSpec.NodeSelector)
		assert.Contains(t, templateSpec.Tolerations, corev1.Toleration{Key: testKey, Value: testValue})
		assert.Equal(t, testClass, templateSpec.PriorityClassName)
	})
}

func TestBuildContainer(t *testing.T) {
	tenantInfo := &dtclient.TenantInfo{}
	instance := &v1alpha1.DynaKube{}
	container := buildContainer(instance, tenantInfo, testUID)

	assert.NotNil(t, container)
	assert.Equal(t, v1alpha1.OperatorName, container.Name)
	assert.Equal(t, corev1.PullAlways, container.ImagePullPolicy)
}

func TestBuildServiceAccountName(t *testing.T) {
	instance := &v1alpha1.DynaKube{}
	assert.Equal(t, MonitoringServiceAccount, buildServiceAccountName(instance))

	instance = &v1alpha1.DynaKube{
		Spec: v1alpha1.DynaKubeSpec{
			KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
				ServiceAccountName: testName,
			},
		},
	}
	assert.Equal(t, testName, buildServiceAccountName(instance))
}

func TestBuildNodeAffinity(t *testing.T) {
	nodeAffinity := buildNodeAffinity()
	assert.NotNil(t, nodeAffinity)
	assert.NotEmpty(t, nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
}

func TestBuildNodeSelectorForAffinity(t *testing.T) {
	nodeSelector := buildNodeSelectorForAffinity()
	assert.NotNil(t, nodeSelector)
	assert.NotEmpty(t, nodeSelector.NodeSelectorTerms)
	assert.Contains(t, nodeSelector.NodeSelectorTerms, corev1.NodeSelectorTerm{MatchExpressions: buildKubernetesBetaArchExpression()})
	assert.Contains(t, nodeSelector.NodeSelectorTerms, corev1.NodeSelectorTerm{MatchExpressions: buildKubernetesArchExpression()})
}

func TestBuildKubernetesExpressions(t *testing.T) {
	archExpression := buildKubernetesArchExpression()
	betaArchExpression := buildKubernetesBetaArchExpression()

	assert.NotNil(t, archExpression)
	assert.NotNil(t, betaArchExpression)
	assert.Equal(t, []corev1.NodeSelectorRequirement{
		{
			Key:      KubernetesArch,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{AMD64, ARM64},
		},
		{
			Key:      KubernetesOs,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{LINUX},
		},
	}, archExpression)
	assert.Equal(t, []corev1.NodeSelectorRequirement{
		{
			Key:      KubernetesBetaArch,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{AMD64, ARM64},
		},
		{
			Key:      KubernetesBetaOs,
			Operator: corev1.NodeSelectorOpIn,
			Values:   []string{LINUX},
		},
	}, betaArchExpression)
}

func TestBuildVolumes(t *testing.T) {
	t.Run(`BuildVolumes with default instance`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{}
		volumes := buildVolumes(instance)
		assert.Empty(t, volumes)
	})
	t.Run(`BuildVolumes with Value and ValueFrom given`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					CustomProperties: &v1alpha1.DynaKubeValueSource{
						Value:     testValue,
						ValueFrom: testValueFrom,
					}}}}
		volumes := buildVolumes(instance)
		assert.NotEmpty(t, volumes)
		assert.Contains(t, volumes, corev1.Volume{
			Name: customproperties.VolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: testValueFrom,
					Items: []corev1.KeyToPath{
						{Key: customproperties.DataKey, Path: customproperties.DataPath},
					},
				},
			}})
	})
	t.Run(`BuildVolumes with Value given`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					CustomProperties: &v1alpha1.DynaKubeValueSource{
						Value: testValue,
					}}}}
		volumes := buildVolumes(instance)
		assert.NotEmpty(t, volumes)
		assert.Contains(t, volumes, corev1.Volume{
			Name: customproperties.VolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("-kubernetes-monitoring%s", customproperties.Suffix),
					Items: []corev1.KeyToPath{
						{Key: customproperties.DataKey, Path: customproperties.DataPath},
					},
				},
			}})
	})
}

func TestBuildVolumeMounts(t *testing.T) {
	t.Run(`BuildVolumeMounts with default instance`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{}
		volumeMounts := buildVolumeMounts(instance)
		assert.Empty(t, volumeMounts)
	})
	t.Run(`BuildVolumeMounts with custom properties`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					CustomProperties: &v1alpha1.DynaKubeValueSource{
						Value: testValue,
					}}}}
		volumeMounts := buildVolumeMounts(instance)
		assert.NotEmpty(t, volumeMounts)
		assert.Contains(t, volumeMounts, corev1.VolumeMount{
			ReadOnly:  true,
			Name:      customproperties.VolumeName,
			MountPath: customproperties.MountPath,
		})
	})
}

func TestBuildEnvs(t *testing.T) {
	t.Run(`BuildEnvs with default values`, func(t *testing.T) {
		tenantInfo := &dtclient.TenantInfo{}
		instance := &v1alpha1.DynaKube{}
		envs := buildEnvs(instance, tenantInfo, testUID)

		assert.NotEmpty(t, envs)
		assert.Contains(t, envs, corev1.EnvVar{Name: DtTenant, Value: tenantInfo.ID})
		assert.Contains(t, envs, corev1.EnvVar{Name: DtToken, Value: tenantInfo.Token})
		assert.Contains(t, envs, corev1.EnvVar{Name: DtServer, Value: tenantInfo.CommunicationEndpoint})
		assert.Contains(t, envs, corev1.EnvVar{Name: DtCapabilities, Value: CapabilityEnv})
		assert.Contains(t, envs, corev1.EnvVar{Name: DtIdSeedNamespace, Value: instance.Namespace})
		assert.Contains(t, envs, corev1.EnvVar{Name: DtIdSeedClusterId, Value: testUID})
	})
	t.Run(`BuildEnvs with custom values`, func(t *testing.T) {
		tenantInfo := &dtclient.TenantInfo{
			ID:                    testId,
			Token:                 testToken,
			Endpoints:             []string{testEndpoint},
			CommunicationEndpoint: testEndpoint,
		}
		instance := &v1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
			}}
		envs := buildEnvs(instance, tenantInfo, testUID)

		assert.NotEmpty(t, envs)
		assert.Contains(t, envs, corev1.EnvVar{Name: DtTenant, Value: testId})
		assert.Contains(t, envs, corev1.EnvVar{Name: DtToken, Value: testToken})
		assert.Contains(t, envs, corev1.EnvVar{Name: DtServer, Value: testEndpoint})
		assert.Contains(t, envs, corev1.EnvVar{Name: DtCapabilities, Value: CapabilityEnv})
		assert.Contains(t, envs, corev1.EnvVar{Name: DtIdSeedNamespace, Value: testNamespace})
		assert.Contains(t, envs, corev1.EnvVar{Name: DtIdSeedClusterId, Value: testUID})
	})
	t.Run(`BuildEnvs with proxy`, func(t *testing.T) {
		tenantInfo := &dtclient.TenantInfo{}
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				Proxy: &v1alpha1.DynaKubeProxy{
					Value: testEndpoint,
				}}}
		envs := buildEnvs(instance, tenantInfo, testUID)

		assert.NotEmpty(t, envs)
		assert.Contains(t, envs, corev1.EnvVar{Name: ProxyEnv, Value: testEndpoint})

		instance = &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				Proxy: &v1alpha1.DynaKubeProxy{
					ValueFrom: testName,
				}}}
		envs = buildEnvs(instance, tenantInfo, testUID)

		assert.NotEmpty(t, envs)
		assert.Contains(t, envs, corev1.EnvVar{Name: ProxyEnv, ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: testName},
				Key:                  ProxyKey,
			}}})
	})
	t.Run(`BuildEnvs with custom environment variables`, func(t *testing.T) {
		tenantInfo := &dtclient.TenantInfo{}
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					Env: []corev1.EnvVar{
						{
							Name:  testName,
							Value: testValue,
						},
						{
							Name: testId,
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{Name: testName},
									Key:                  testKey,
								}},
						},
					},
				}}}
		envs := buildEnvs(instance, tenantInfo, testUID)
		assert.NotEmpty(t, envs)
		assert.Contains(t, envs, corev1.EnvVar{Name: testName, Value: testValue})
		assert.Contains(t, envs, corev1.EnvVar{Name: testId, ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: testName},
				Key:                  testKey,
			}},
		})
	})
}

func TestBuildArgs(t *testing.T) {
	t.Run(`BuildArgs with default instance`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{}
		args := buildArgs(instance)
		assert.NotEmpty(t, args)
		assert.Contains(t, args, DtTenantArg)
		assert.Contains(t, args, DtTokenArg)
		assert.Contains(t, args, DtServerArg)
		assert.Contains(t, args, DtCapabilitiesArg)
	})

	t.Run(`BuildArgs with network zone`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				NetworkZone: testName,
			}}
		args := buildArgs(instance)
		assert.NotEmpty(t, args)
		assert.Contains(t, args, DtTenantArg)
		assert.Contains(t, args, DtTokenArg)
		assert.Contains(t, args, DtServerArg)
		assert.Contains(t, args, DtCapabilitiesArg)
		assert.Contains(t, args, fmt.Sprintf(`--networkzone="%s"`, testName))
	})
	t.Run(`BuildArgs with proxy`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				Proxy: &v1alpha1.DynaKubeProxy{
					Value: testEndpoint,
				}}}
		args := buildArgs(instance)
		assert.NotEmpty(t, args)
		assert.Contains(t, args, DtTenantArg)
		assert.Contains(t, args, DtTokenArg)
		assert.Contains(t, args, DtServerArg)
		assert.Contains(t, args, DtCapabilitiesArg)
		assert.Contains(t, args, ProxyArg)
	})
	t.Run(`BuildArgs with activation group`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					Group: testName,
				}}}
		args := buildArgs(instance)
		assert.NotEmpty(t, args)
		assert.Contains(t, args, DtTenantArg)
		assert.Contains(t, args, DtTokenArg)
		assert.Contains(t, args, DtServerArg)
		assert.Contains(t, args, DtCapabilitiesArg)
		assert.Contains(t, args, fmt.Sprintf(`--group="%s"`, testName))
	})
	t.Run(`BuildArgs with many values`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				NetworkZone: testName,
				Proxy: &v1alpha1.DynaKubeProxy{
					Value: testEndpoint,
				},
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					Group: testName,
				}}}
		args := buildArgs(instance)
		assert.NotEmpty(t, args)
		assert.Contains(t, args, DtTenantArg)
		assert.Contains(t, args, DtTokenArg)
		assert.Contains(t, args, DtServerArg)
		assert.Contains(t, args, DtCapabilitiesArg)
		assert.Contains(t, args, ProxyArg)
		assert.Contains(t, args, fmt.Sprintf(`--networkzone="%s"`, testName))
		assert.Contains(t, args, fmt.Sprintf(`--group="%s"`, testName))
	})
}
