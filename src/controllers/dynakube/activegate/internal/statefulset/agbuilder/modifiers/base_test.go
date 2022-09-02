package modifiers

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testKubeUID       = "123test"
	testConfigHash    = "testHash"
	testDynakubeName  = "testDk"
	testNamespaceName = "testNs"
)

func createBuilderForTesting() agbuilder.Builder {
	builder := agbuilder.Builder{}
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
							Name: consts.ActiveGateContainerName,
						},
					},
				},
			},
		},
	}
	builder.SetBase(base)
	return builder
}

func TestGetBaseObjectMeta(t *testing.T) {
	dynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.KubeMonCapability.DisplayName,
				},
			},
		},
	}
	t.Run("creating object meta", func(t *testing.T) {
		cap := capability.NewMultiCapability(&dynakube)
		mod := NewBaseModifier(testKubeUID, testConfigHash, dynakube, cap).(BaseModifier)

		objectMeta := mod.getBaseObjectMeta()

		require.NotEmpty(t, objectMeta)
		assert.Contains(t, objectMeta.Name, dynakube.Name)
		assert.Contains(t, objectMeta.Name, cap.ShortName())
		assert.NotNil(t, objectMeta.Annotations)
	})
}
