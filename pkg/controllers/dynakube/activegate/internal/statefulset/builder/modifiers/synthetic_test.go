package modifiers

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestSyntheticContainer(t *testing.T) {
	assertion := assert.New(t)

	dynaKube := getBaseDynakube()
	dynaKube.ObjectMeta.Annotations[dynatracev1beta1.AnnotationFeatureSyntheticNodeType] = dynatracev1beta1.SyntheticNodeXs

	modifier := newSyntheticModifier(
		dynaKube,
		capability.NewSyntheticCapability(&dynaKube),
		prioritymap.New(),
	)
	container := modifier.buildContainer()

	t.Run("by liveness probe", func(t *testing.T) {
		assertion.NotEmpty(container.LivenessProbe, "declared liveness probe")
		assertion.Equal(
			container.LivenessProbe.Exec.Command,
			livenessCmd,
			"declared command for liveness probe")
	})

	t.Run("by volumes", func(t *testing.T) {
		assertion.Subset(
			container.VolumeMounts,
			modifier.getVolumeMounts(),
			"declared mount paths")
	})

	t.Run("by requirements", func(t *testing.T) {
		expectedRequestCpu := resource.NewScaledQuantity(1000, resource.Milli).String()
		assertion.Equalf(
			expectedRequestCpu,
			container.Resources.Requests.Cpu().String(),
			"declared for %v node resource request CPU: %v",
			dynatracev1beta1.SyntheticNodeXs,
			expectedRequestCpu)
	})

	t.Run("by environment variables", func(t *testing.T) {
		assertion.Subset(
			container.Env,
			modifier.getEnvs(),
			"declared environment variables")
	})
}
