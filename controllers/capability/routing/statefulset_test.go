package routing

import (
	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
	testValue     = "test-value"
	testUID       = "test-uid"
)

func TestNewStatefulSetBuilder(t *testing.T) {
	stsBuilder := newStatefulSetProperties(&v1alpha1.DynaKube{}, &v1alpha1.CapabilityProperties{}, testUID, testValue)
	assert.NotNil(t, stsBuilder)
	assert.NotNil(t, stsBuilder.DynaKube)
	assert.NotNil(t, stsBuilder.CapabilityProperties)
	assert.NotNil(t, stsBuilder.CustomPropertiesHash)
	assert.NotEmpty(t, stsBuilder.CustomPropertiesHash)
	assert.NotEmpty(t, stsBuilder.KubeSystemUID)
}

func TestStatefulSetBuilder_Build(t *testing.T) {
	instance := buildTestInstance()
	capabilityProperties := &instance.Spec.RoutingSpec.CapabilityProperties

	t.Run(`is not nil`, func(t *testing.T) {
		sts, err := createStatefulSet(newStatefulSetProperties(instance, &v1alpha1.CapabilityProperties{}, "", ""))
		assert.NoError(t, err)
		assert.NotNil(t, sts)
	})
	t.Run(`name is instance name plus correct suffix`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance, &v1alpha1.CapabilityProperties{}, "", ""))
		assert.Equal(t, instance.Name+StatefulSetSuffix, sts.Name)
	})
	t.Run(`namespace is instance namespace`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance, &v1alpha1.CapabilityProperties{}, "", ""))
		assert.Equal(t, instance.Namespace, sts.Namespace)
	})
	t.Run(`has labels`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance, &v1alpha1.CapabilityProperties{}, "", ""))
		assert.Equal(t, map[string]string{
			capability.KeyDynatrace:  capability.ValueActiveGate,
			capability.KeyActiveGate: instance.Name,
		}, sts.Labels)
	})
	t.Run(`has replicas`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance, capabilityProperties, "", ""))
		assert.Equal(t, instance.Spec.RoutingSpec.Replicas, sts.Spec.Replicas)
	})
	t.Run(`has pod management policy`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance, &v1alpha1.CapabilityProperties{}, "", ""))
		assert.Equal(t, appsv1.ParallelPodManagement, sts.Spec.PodManagementPolicy)
	})
	t.Run(`has selector`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance, &v1alpha1.CapabilityProperties{}, "", ""))
		assert.Equal(t, metav1.LabelSelector{
			MatchLabels: capability.BuildLabelsFromInstance(instance),
		}, *sts.Spec.Selector)
	})
	t.Run(`has non empty template`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance, &v1alpha1.CapabilityProperties{}, "", ""))
		assert.NotEqual(t, corev1.PodTemplateSpec{}, sts.Spec.Template)
	})
	t.Run(`template has labels`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance, capabilityProperties, "", ""))
		assert.Equal(t, capability.BuildLabels(instance, capabilityProperties), sts.Spec.Template.Labels)
	})
	t.Run(`template has annotations`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance, capabilityProperties, "", testValue))
		assert.Equal(t, map[string]string{
			annotationImageHash:       instance.Status.ActiveGate.ImageHash,
			annotationImageVersion:    instance.Status.ActiveGate.ImageVersion,
			annotationCustomPropsHash: testValue,
		}, sts.Spec.Template.Annotations)
	})
	t.Run(`template has non empty spec`, func(t *testing.T) {
		sts, _ := createStatefulSet(newStatefulSetProperties(instance, capabilityProperties, "", ""))
		assert.NotEqual(t, corev1.PodSpec{}, sts.Spec.Template.Spec)
	})
}

func buildTestInstance() *v1alpha1.DynaKube {
	replicas := int32(3)

	return &v1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: v1alpha1.DynaKubeSpec{
			RoutingSpec: v1alpha1.RoutingSpec{
				v1alpha1.CapabilityProperties{
					Replicas: &replicas,
				}},
		},
	}
}
