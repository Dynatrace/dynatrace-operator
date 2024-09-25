package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
)

const (
	expectedBaseInitArgsLen = 11
)

func TestGetInitArgs(t *testing.T) {
	t.Run("get base init args", func(t *testing.T) {
		dk := dynakube.DynaKube{}
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
		dk.Spec.Templates.LogModule.Args = []string{
			"customArg1",
			"customArg2",
		}
		args := getInitArgs(dk)

		assert.Len(t, args, expectedBaseInitArgsLen+len(dk.Spec.Templates.LogModule.Args))

		for _, customArg := range dk.Spec.Templates.LogModule.Args {
			assert.Contains(t, args, customArg)
		}
	})
}
