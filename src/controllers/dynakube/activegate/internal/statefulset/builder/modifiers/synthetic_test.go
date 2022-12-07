package modifiers

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestSyntheticContainers(t *testing.T) {
	assertion := assert.New(t)

	t.Run("synthetic-container", func(t *testing.T) {
		dynakube := getBaseDynakube()
		dynakube.Annotations[dynatracev1beta1.AnnotationFeatureSyntheticNodeType] = dynatracev1beta1.SyntheticNodeXs

		syn := newSyntheticModifier(dynakube)
		container := syn.buildContainer()

		assertion.NotEmpty(container.LivenessProbe, "declared liveness probe")
		assertion.Equal(
			container.LivenessProbe.Exec.Command,
			livenessCmd,
			"declared command for liveness probe")

		for _, mnt := range syn.getVolumeMounts() {
			assertion.Truef(
				kubeobjects.MountPathIsIn(container.VolumeMounts, mnt.MountPath),
				"declared mount path: %s",
				mnt.MountPath)
		}

		for _, env := range syn.getEnvs() {
			assertion.Truef(
				kubeobjects.EnvVarIsIn(container.Env, env.Name),
				"declared environment variable: %s",
				env.Name)
		}

		expectedRequestCpu := resource.NewScaledQuantity(1000, resource.Milli).String()
		assertion.Equal(
			container.Resources.Requests.Cpu().String(),
			expectedRequestCpu,
			"declared for %v node resource request CPU: %v",
			dynatracev1beta1.SyntheticNodeXs,
			expectedRequestCpu)
	})
}
