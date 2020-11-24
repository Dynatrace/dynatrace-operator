package kubemon

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/customproperties"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testUID       = "test-uid"
	testId        = "test-id"
	testKey       = "key"
	testValue     = "value"
	testValueFrom = "valueFrom"
	testName      = "test-name"
	testNamespace = "test-namespace"
	testEndpoint  = "http://test-endpoint"
)

func TestNewStatefulSet(t *testing.T) {
	instance := v1alpha1.DynaKube{}

	sts := newStatefulSet(instance, testUID)
	assert.NotNil(t, sts)
	assert.Equal(t, &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        v1alpha1.Name,
			Namespace:   instance.Namespace,
			Labels:      buildLabels(&instance),
			Annotations: map[string]string{},
		},
		Spec: v1.StatefulSetSpec{
			Replicas: instance.Spec.KubernetesMonitoringSpec.Replicas,
			Selector: buildLabelSelector(&instance),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: buildLabels(&instance)},
				Spec:       buildTemplateSpec(&instance, testUID),
			},
		},
	}, sts)
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
					SecretName: fmt.Sprintf("-kubernetes-monitoring-%s", customproperties.Suffix),
					Items: []corev1.KeyToPath{
						{Key: customproperties.DataKey, Path: customproperties.DataPath},
					},
				},
			}})
	})
}

func TestBuildEnvs_WithProxy(t *testing.T) {
	t.Run(`BuildEnvs with Proxy from Value`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				Proxy: &v1alpha1.DynaKubeProxy{
					Value: testValue,
				}}}

		envs := buildEnvs(instance, testUID)
		assert.NotEmpty(t, envs)

		var proxyEnvVar *corev1.EnvVar
		for _, envVar := range envs {
			if envVar.Name == ProxyEnv {
				proxyEnvVar = &envVar
			}
		}

		assert.NotNil(t, proxyEnvVar)
		if proxyEnvVar != nil {
			// Check for nil value to make linter happy
			assert.Equal(t, testValue, proxyEnvVar.Value)
			assert.Nil(t, proxyEnvVar.ValueFrom)
		}
	})
	t.Run(`BuildEnvs with Proxy from ValueFrom`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				Proxy: &v1alpha1.DynaKubeProxy{
					ValueFrom: testKey,
				}}}

		envs := buildEnvs(instance, testUID)
		assert.NotEmpty(t, envs)

		var proxyEnvVar *corev1.EnvVar
		for _, envVar := range envs {
			if envVar.Name == ProxyEnv {
				proxyEnvVar = &envVar
			}
		}

		assert.NotNil(t, proxyEnvVar)
		if proxyEnvVar != nil {
			// Check for nil value to make linter happy
			assert.Equal(t, &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: testKey},
					Key:                  ProxyKey,
				},
			}, proxyEnvVar.ValueFrom)
			assert.Empty(t, proxyEnvVar.Value)
		}
	})
}

func TestBuildArgs(t *testing.T) {
	t.Run(`BuildArgs with network zone`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				NetworkZone: testValue,
			}}
		args := buildArgs(instance)

		assert.NotEmpty(t, args)
		assert.Contains(t, args, fmt.Sprintf(`--networkzone="%s"`, testValue))
	})
	t.Run(`BuildArgs with proxy`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				Proxy: &v1alpha1.DynaKubeProxy{
					Value: testValue,
				},
			}}
		args := buildArgs(instance)

		assert.NotEmpty(t, args)
		assert.Contains(t, args, ProxyArg)
	})
	t.Run(`BuildArgs with activation group`, func(t *testing.T) {
		instance := &v1alpha1.DynaKube{
			Spec: v1alpha1.DynaKubeSpec{
				KubernetesMonitoringSpec: v1alpha1.KubernetesMonitoringSpec{
					Group: testValue,
				}}}
		args := buildArgs(instance)

		assert.NotEmpty(t, args)
		assert.Contains(t, args, fmt.Sprintf(`--group="%s"`, testValue))
	})
}
