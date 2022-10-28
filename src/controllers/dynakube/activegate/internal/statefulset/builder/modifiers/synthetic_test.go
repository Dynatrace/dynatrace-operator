package modifiers

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
)

func TestSyntheticContainers(t *testing.T) {
	assertion := assert.New(t)

	t.Run(
		"synthetic-container",
		func(t *testing.T) {
			dynakube := getBaseDynakube()
			syn := newSyntheticModifier(dynakube)
			container := syn.buildContainer()

			assertion.NotEmpty(container.LivenessProbe, "declared liveness probe")
			assertion.Equal(container.LivenessProbe.Exec.Command, livenessCmd, "declared command for liveness probe")

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
		})
}
