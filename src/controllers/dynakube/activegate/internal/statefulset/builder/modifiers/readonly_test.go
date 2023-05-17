package modifiers

import (
	"strconv"
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setReadOnlyUsage(dynakube *dynatracev1.DynaKube, isUsed bool) {
	dynakube.Annotations[dynatracev1.AnnotationFeatureActiveGateReadOnlyFilesystem] = strconv.FormatBool(isUsed)
}

func TestReadOnlyEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setReadOnlyUsage(&dynakube, true)

		mod := NewReadOnlyModifier(dynakube)

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setReadOnlyUsage(&dynakube, false)

		mod := NewReadOnlyModifier(dynakube)

		assert.False(t, mod.Enabled())
	})
}

func TestReadOnlyModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setReadOnlyUsage(&dynakube, true)
		mod := NewReadOnlyModifier(dynakube)
		builder := createBuilderForTesting()
		expectedVolumes := mod.getVolumes()
		expectedVolumeMounts := mod.getVolumeMounts()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, expectedVolumes, sts.Spec.Template.Spec.Volumes)
		isSubset(t, expectedVolumeMounts, sts.Spec.Template.Spec.Containers[0].VolumeMounts)
	})
}
