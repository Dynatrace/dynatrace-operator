package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
	"github.com/stretchr/testify/assert"
)

const (
	expectedBaseInitArgsLen            = 12
	expectedBaseInitArgsLenWithoutMEID = 11
)

func TestGetInitArgs(t *testing.T) {
	t.Run("get base init args", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Status.KubernetesClusterMEID = "test-me-id"
		dk.Name = "dk-name-test"
		args := getInitArgs(dk)

		assert.Len(t, args, expectedBaseInitArgsLen)

		for _, arg := range args {
			assert.NotEmpty(t, arg)
		}
	})

	t.Run("add user defined args to existing init args", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Name = "dk-name-test"
		dk.Status.KubernetesClusterMEID = "test-me-id"
		dk.Status.KubernetesClusterName = "test-cluster-name"
		dk.Spec.Templates.LogMonitoring = &logmonitoring.TemplateSpec{
			Args: []string{
				"customArg1",
				"customArg2",
			},
		}
		args := getInitArgs(dk)

		assert.Len(t, args, expectedBaseInitArgsLen+len(dk.LogMonitoring().Args))

		for _, customArg := range dk.LogMonitoring().Args {
			assert.Contains(t, args, customArg)
		}
	})

	t.Run("get base init args when no MEID or not all necessary scopes are set", func(t *testing.T) {
		dk := dynakube.DynaKube{}
		dk.Name = "dk-name-test"
		args := getInitArgs(dk)

		assert.Len(t, args, expectedBaseInitArgsLenWithoutMEID)

		for _, arg := range args {
			assert.NotEmpty(t, arg)
		}
	})
}
