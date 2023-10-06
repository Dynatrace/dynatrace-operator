package troubleshoot

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/stretchr/testify/assert"
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
		dynakube := buildTestDynakube()
		clt := fake.NewClient(&dynakube)

		dynakubes, err := getAllDynakubesInNamespace(context.Background(), getNullLogger(t), clt, testNamespace)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(dynakubes))
		assert.Equal(t, dynakube.Name, dynakubes[0].Name)
	})

	t.Run("getDynakube - only check one dynakube if set", func(t *testing.T) {
		dynakube := buildTestDynakube()
		clt := fake.NewClient(&dynakube)
		dynakubes, err := getDynakubes(context.Background(), getNullLogger(t), clt, testNamespace, testDynakube)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(dynakubes))
		assert.Equal(t, testDynakube, dynakubes[0].Name)
	})
}

func buildTestDynakube() dynatracev1beta1.DynaKube {
	return dynatracev1beta1.DynaKube{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakube,
			Namespace: testNamespace,
		},
	}
}
