package modifiers

import (
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestSyntheticContainer(t *testing.T) {
	assertion := assert.New(t)

	dynaKube := getBaseDynakube()
	dynaKube.ObjectMeta.Annotations[dynatracev1.AnnotationFeatureSyntheticNodeType] = dynatracev1.SyntheticNodeXs

	modifier := newSyntheticModifier(
		dynaKube,
		capability.NewSyntheticCapability(&dynaKube),
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
		assertion.Equal(
			container.Resources.Requests.Cpu().String(),
			expectedRequestCpu,
			"declared for %v node resource request CPU: %v",
			dynatracev1.SyntheticNodeXs,
			expectedRequestCpu)
	})

	t.Run("by environment variables", func(t *testing.T) {
		assertion.Subset(
			container.Env,
			modifier.getEnvs(),
			"declared environment variables")
	})
}
