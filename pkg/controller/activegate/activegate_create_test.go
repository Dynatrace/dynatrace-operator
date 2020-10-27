package activegate

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestCreateStatefulSet(t *testing.T) {
	r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
	assert.NotNil(t, r)
	assert.NoError(t, err)

	result, err := r.newStatefulSetForCR(instance, &dtclient.TenantInfo{}, "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreate(t *testing.T) {
	t.Run("create with custom properties", testCreateCustomProperties)
	t.Run("create with network zone", testCreateWithNetworkZone)
	t.Run("create with activation group", testCreateWithActivationGroup)
	t.Run("create with trusted certificates", testCreateWithTrustedCertificates)
	t.Run("create with proxy settings", testCreateWithProxySettings)
}

func testCreateWithProxySettings(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		podSpec := setupForPodSpec(t, func(instance *v1alpha1.DynaKube) {
			instance.Spec.KubernetesMonitoringSpec.Enabled = true
			instance.Spec.Proxy = &v1alpha1.DynaKubeProxy{
				Value: "https://my-proxy",
			}
		})
		assert.NotNil(t, podSpec)

		container := podSpec.Containers[0]
		assert.Contains(t, container.Args, `PROXY="${ACTIVE_GATE_PROXY}"`)

		var proxyEnvVar *v1.EnvVar
		for _, env := range container.Env {
			if env.Name == "ACTIVE_GATE_PROXY" {
				proxyEnvVar = &env
			}
		}

		assert.NotNil(t, proxyEnvVar)
		// Check for nil so linter does not complain
		if proxyEnvVar != nil {
			assert.Equal(t, "https://my-proxy", proxyEnvVar.Value)
		}
	})
	t.Run("with value source", func(t *testing.T) {
		podSpec := setupForPodSpec(t, func(instance *v1alpha1.DynaKube) {
			instance.Spec.KubernetesMonitoringSpec.Enabled = true
			instance.Spec.Proxy = &v1alpha1.DynaKubeProxy{
				ValueFrom: "proxy-config",
			}
		})
		assert.NotNil(t, podSpec)

		container := podSpec.Containers[0]
		assert.Contains(t, container.Args, `PROXY="${ACTIVE_GATE_PROXY}"`)

		var proxyEnvVar *v1.EnvVar
		for _, env := range container.Env {
			if env.Name == "ACTIVE_GATE_PROXY" {
				proxyEnvVar = &env
			}
		}

		assert.NotNil(t, proxyEnvVar)
		// Check for nil so linter does not complain
		if proxyEnvVar != nil {
			assert.Equal(t, "proxy-config", proxyEnvVar.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
			assert.Equal(t, "proxy", proxyEnvVar.ValueFrom.SecretKeyRef.Key)
		}
	})
}

func testCreateWithTrustedCertificates(t *testing.T) {
	podSpec := setupForPodSpec(t, func(instance *v1alpha1.DynaKube) {
		instance.Spec.KubernetesMonitoringSpec.Enabled = true
		instance.Spec.TrustedCAs = "ca-config-map"
	})
	assert.GreaterOrEqual(t, 1, len(podSpec.Volumes))

	var caVolume *v1.Volume
	for _, volume := range podSpec.Volumes {
		if volume.Name == "certs" {
			caVolume = &volume
		}
	}

	assert.NotNil(t, caVolume)
	// Check for nil so linter does not complain
	if caVolume != nil {
		volumeSource := caVolume.VolumeSource
		assert.NotNil(t, volumeSource)
		assert.NotNil(t, volumeSource.ConfigMap)
		assert.NotNil(t, volumeSource.ConfigMap.LocalObjectReference)
		assert.Equal(t, "ca-config-map", volumeSource.ConfigMap.LocalObjectReference.Name)
		assert.Equal(t, 1, len(volumeSource.ConfigMap.Items))
		assert.Equal(t, "certs", volumeSource.ConfigMap.Items[0].Key)
		assert.Equal(t, "certs.pem", volumeSource.ConfigMap.Items[0].Path)
	}
}

func testCreateWithActivationGroup(t *testing.T) {
	podSpec := setupForPodSpec(t, func(instance *v1alpha1.DynaKube) {
		instance.Spec.KubernetesMonitoringSpec.Enabled = true
		instance.Spec.KubernetesMonitoringSpec.Group = "my-group"
	})
	container := podSpec.Containers[0]
	assert.Contains(t, container.Args, `--group "my-group"`)
}

func testCreateCustomProperties(t *testing.T) {
	configMapValue := `
[section]
property=value
`
	r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
	assert.NotNil(t, r)
	assert.NoError(t, err)

	instance.Spec.KubernetesMonitoringSpec.Enabled = true
	instance.Spec.KubernetesMonitoringSpec.CustomProperties = &v1alpha1.DynaKubeValueSource{
		Value: configMapValue,
	}

	updateInstance(t, r, instance)

	var configMap v1.ConfigMap
	err = r.client.Get(context.TODO(), client.ObjectKey{Name: _const.CustomPropertiesConfigMapName, Namespace: _const.DynatraceNamespace}, &configMap)
	assert.NoError(t, err)
	assert.NotNil(t, configMap)

	configMapData, hasData := configMap.Data[_const.CustomPropertiesKey]
	assert.True(t, hasData)
	assert.Equal(t, configMapValue, configMapData)
}

func testCreateWithNetworkZone(t *testing.T) {
	podSpec := setupForPodSpec(t, func(instance *v1alpha1.DynaKube) {
		instance.Spec.KubernetesMonitoringSpec.Enabled = true
		instance.Spec.NetworkZone = "us-east-1"
	})
	container := podSpec.Containers[0]
	assert.Contains(t, container.Args, `--networkzone "us-east-1"`)
}

func updateInstance(t *testing.T, r *ReconcileActiveGate, instance *v1alpha1.DynaKube) {
	err := r.client.Update(context.TODO(), instance)
	assert.NoError(t, err)

	result, err := r.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      instance.Name,
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func setupForPodSpec(t *testing.T, applyToInstance func(instance *v1alpha1.DynaKube)) v1.PodSpec {
	r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
	assert.NotNil(t, r)
	assert.NoError(t, err)

	applyToInstance(instance)

	updateInstance(t, r, instance)

	var statefulSet appsv1.StatefulSet
	err = r.client.Get(context.TODO(), client.ObjectKey{Name: _const.ActivegateName, Namespace: instance.Namespace}, &statefulSet)
	assert.NoError(t, err)
	assert.NotNil(t, statefulSet)

	podSpec := statefulSet.Spec.Template.Spec
	assert.NotNil(t, podSpec)
	assert.GreaterOrEqual(t, 1, len(podSpec.Containers))

	return podSpec
}
