package k8sconditions

import (
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestIsOutdated(t *testing.T) {
	testingConditionType := "testing "

	t.Run("empty condition => outdated", func(t *testing.T) {
		tp := timeprovider.New()
		dk := &dynakube.DynaKube{}

		assert.True(t, IsOutdated(tp, dk, testingConditionType))
	})

	t.Run("False condition => outdated", func(t *testing.T) {
		tp := timeprovider.New()
		dk := &dynakube.DynaKube{}

		SetDynatraceAPIError(dk.Conditions(), testingConditionType, errors.New("boom"))

		assert.True(t, IsOutdated(tp, dk, testingConditionType))
	})

	t.Run("True condition + current timestamp => NOT outdated", func(t *testing.T) {
		tp := timeprovider.New()
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{},
		}

		SetSecretCreated(dk.Conditions(), testingConditionType, "")

		assert.False(t, IsOutdated(tp, dk, testingConditionType))
	})

	t.Run("old timestamp => outdated", func(t *testing.T) {
		tp := timeprovider.New()
		dk := &dynakube.DynaKube{}

		SetSecretCreated(dk.Conditions(), testingConditionType, "")
		tp.Set(tp.Now().Add(time.Minute * 60))

		assert.True(t, IsOutdated(tp, dk, testingConditionType))
	})
}
