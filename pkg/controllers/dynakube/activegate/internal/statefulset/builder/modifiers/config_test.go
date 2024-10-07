package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testDynakubeName  = "testDk"
	testNamespaceName = "testNs"
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

func getBaseDynakube() dynakube.DynaKube {
	return dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testDynakubeName,
			Namespace:   testNamespaceName,
			Annotations: map[string]string{},
		},
		Spec: dynakube.DynaKubeSpec{},
	}
}

func enableKubeMonCapability(dk *dynakube.DynaKube) {
	dk.Spec.ActiveGate.Capabilities = append(dk.Spec.ActiveGate.Capabilities, activegate.KubeMonCapability.DisplayName)
}

func isSubset[T any](t *testing.T, subset, superset []T) {
	for _, r := range subset {
		assert.Contains(t, superset, r)
	}
}

func enableAllModifiers(dk *dynakube.DynaKube, capability capability.Capability) {
	setCertUsage(dk, true)
	setCustomPropertyUsage(capability, true)
	setProxyUsage(dk, true)
	setKubernetesMonitoringUsage(dk, true)
	setServicePortUsage(dk, true)
}

func TestNoConflict(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dk := getBaseDynakube()
		enableKubeMonCapability(&dk)
		multiCapability := capability.NewMultiCapability(&dk)
		enableAllModifiers(&dk, multiCapability)
		mods := GenerateAllModifiers(dk, multiCapability, prioritymap.New())
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mods...).Build()

		require.NotEmpty(t, sts)

		for _, mod := range mods {
			if !mod.Enabled() {
				continue
			}

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
				isSubset(t, envMod.getEnvs(), sts.Spec.Template.Spec.Containers[0].Env)
			}

			initMod, ok := mod.(initContainerModifier)
			if ok {
				isSubset(t, initMod.getInitContainers(), sts.Spec.Template.Spec.InitContainers)
			}
		}
	})
}
