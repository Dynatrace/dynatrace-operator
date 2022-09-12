package modifiers

import (
	"strconv"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setAutTokenUsage(dynakube *dynatracev1beta1.DynaKube, isUsed bool) {
	dynakube.Annotations[dynatracev1beta1.AnnotationFeatureActiveGateAuthToken] = strconv.FormatBool(isUsed)
}

func TestAuthTokenEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setAutTokenUsage(&dynakube, true)

		mod := NewAuthTokenModifier(dynakube)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setAutTokenUsage(&dynakube, false)

		mod := NewAuthTokenModifier(dynakube)

		assert.False(t, mod.Enabled())
	})
}

func TestAuthTokenModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setAutTokenUsage(&dynakube, true)
		mod := NewAuthTokenModifier(dynakube)
		builder := createBuilderForTesting()

		sts := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
