package modifiers

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testDynakubeName  = "testDk"
	testNamespaceName = "testNs"
	testTenant        = "testTenant"
	testApiUrl        = "https://" + testTenant + ".xyz/api"
)

func createBuilderForTesting() builder.Builder {
	base := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testing",
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testing",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            consts.ActiveGateContainerName,
							SecurityContext: &corev1.SecurityContext{},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{},
								},
							},
						},
					},
				},
			},
		},
	}
	builder := builder.NewBuilder(base)
	return builder
}

func getBaseDynakube() dynatracev1beta1.DynaKube {
	return dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDynakubeName,
			Namespace:   testNamespaceName,
			Annotations: map[string]string{},
		},
		Spec: dynatracev1beta1.DynaKubeSpec{APIURL: testApiUrl},
	}
}

func enableKubeMonCapability(dynakube *dynatracev1beta1.DynaKube) {
	dynakube.Spec.ActiveGate.Capabilities = append(dynakube.Spec.ActiveGate.Capabilities, dynatracev1beta1.KubeMonCapability.DisplayName)
}

func isSubset[T any](t *testing.T, subset, superset []T) {
	for _, r := range subset {
		assert.Contains(t, superset, r)
	}
}

func enableAllModifiers(dynakube *dynatracev1beta1.DynaKube, capability capability.Capability) {
	setAutTokenUsage(dynakube, true)
	setCertUsage(dynakube, true)
	setCustomPropertyUsage(capability, true)
	setProxyUsage(dynakube, true)
	setRawImageUsage(dynakube, true)
	setReadOnlyUsage(dynakube, true)
	setKubernetesMonitoringUsage(dynakube, true)
	setServicePortUsage(dynakube, true)
}

func TestNoConflict(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		multiCapability := capability.NewMultiCapability(&dynakube)
		enableAllModifiers(&dynakube, multiCapability)
		mods := GenerateAllModifiers(dynakube, multiCapability)
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mods...).Build()

		require.NotEmpty(t, sts)
		for _, mod := range mods {
			volumesMod, ok := mod.(volumeModifier)
			if ok {
				isSubset(t, volumesMod.getVolumes(), sts.Spec.Template.Spec.Volumes)
			}
			mountMod, ok := mod.(volumeMountModifier)
			if ok {
				isSubset(t, mountMod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
			}
			envMod, ok := mod.(envModifier)
			if ok {
				envs, err := envMod.getEnvs()
				assert.NoError(t, err)
				isSubset(t, envs, sts.Spec.Template.Spec.Containers[0].Env)
			}
			initMod, ok := mod.(initContainerModifier)
			if ok {
				isSubset(t, initMod.getInitContainers(), sts.Spec.Template.Spec.InitContainers)
			}
		}
	})
}
