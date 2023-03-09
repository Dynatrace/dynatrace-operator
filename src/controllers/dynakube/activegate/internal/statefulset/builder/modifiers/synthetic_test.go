package modifiers

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
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
	)
	container := modifier.buildContainer()

	toAssertProbe := func(t *testing.T) {
		assertion.NotEmpty(container.LivenessProbe, "declared liveness probe")
		assertion.Equal(
			container.LivenessProbe.Exec.Command,
			livenessCmd,
			"declared command for liveness probe")
	}
	t.Run("by-probe", toAssertProbe)

	toAssertVolumes := func(t *testing.T) {
		assertion.Subset(
			container.VolumeMounts,
			modifier.getVolumeMounts(),
			"declared mount paths")
	}
	t.Run("by-volumes", toAssertVolumes)

	toAssertRequirements := func(t *testing.T) {
		expectedRequestCpu := resource.NewScaledQuantity(1000, resource.Milli).String()
		assertion.Equal(
			container.Resources.Requests.Cpu().String(),
			expectedRequestCpu,
			"declared for %v node resource request CPU: %v",
			dynatracev1beta1.SyntheticNodeXs,
			expectedRequestCpu)
	}
	t.Run("by-requirements", toAssertRequirements)

	toAssertEnv := func(t *testing.T) {
		assertion.Subset(
			container.Env,
			modifier.getEnvs(),
			"declared environment variables")
	}
	t.Run("by-environment-variables", toAssertEnv)
}
