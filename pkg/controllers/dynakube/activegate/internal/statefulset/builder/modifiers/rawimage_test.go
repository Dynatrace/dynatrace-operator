package modifiers

import (
	"strconv"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setRawImageUsage(dynakube *dynatracev1beta1.DynaKube, isUsed bool) {
	dynakube.Annotations[dynatracev1beta1.AnnotationFeatureActiveGateRawImage] = strconv.FormatBool(isUsed)
}

func TestRawImageEnabled(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setRawImageUsage(&dynakube, true)

		mod := NewRawImageModifier(dynakube, prioritymap.NewMap())

		assert.True(t, mod.Enabled())
	})

	t.Run("false", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setRawImageUsage(&dynakube, false)

		mod := NewRawImageModifier(dynakube, prioritymap.NewMap())

		assert.False(t, mod.Enabled())
	})
}

func TestRawImageModify(t *testing.T) {
	t.Run("successfully modified", func(t *testing.T) {
		dynakube := getBaseDynakube()
		enableKubeMonCapability(&dynakube)
		setRawImageUsage(&dynakube, true)
		mod := NewRawImageModifier(dynakube, prioritymap.NewMap())
		builder := createBuilderForTesting()

		sts, _ := builder.AddModifier(mod).Build()

		require.NotEmpty(t, sts)
		isSubset(t, mod.getVolumes(), sts.Spec.Template.Spec.Volumes)
		isSubset(t, mod.getVolumeMounts(), sts.Spec.Template.Spec.Containers[0].VolumeMounts)
		isSubset(t, mod.getEnvs(), sts.Spec.Template.Spec.Containers[0].Env)
	})
}
