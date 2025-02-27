package troubleshoot

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTroubleshootCommandBuilder(t *testing.T) {
	t.Run("build command", func(t *testing.T) {
		builder := NewTroubleshootCommandBuilder()
		csiCommand := builder.Build()

		assert.NotNil(t, csiCommand)
		assert.Equal(t, use, csiCommand.Use)
		assert.NotNil(t, csiCommand.RunE)
	})

	t.Run("getAllDynakubesInNamespace", func(t *testing.T) {
		dk := buildTestDynakube()
		clt := fake.NewClient(&dk)

		dynakubes, err := getAllDynakubesInNamespace(context.Background(), getNullLogger(t), clt, testNamespace)
		require.NoError(t, err)
		assert.Len(t, dynakubes, 1)
		assert.Equal(t, dk.Name, dynakubes[0].Name)
	})

	t.Run("getDynakube - only check one dynakube if set", func(t *testing.T) {
		dk := buildTestDynakube()
		clt := fake.NewClient(&dk)
		dynakubes, err := getDynakubes(context.Background(), getNullLogger(t), clt, testNamespace, testDynakube)
		require.NoError(t, err)
		assert.Len(t, dynakubes, 1)
		assert.Equal(t, testDynakube, dynakubes[0].Name)
	})
}

func buildTestDynakube() dynakube.DynaKube {
	return dynakube.DynaKube{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakube,
			Namespace: testNamespace,
		},
	}
}
