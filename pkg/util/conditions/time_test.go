package conditions

import (
	"testing"
	"time"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestIsOutdated(t *testing.T) {
	testingConditionType := "testing "

	t.Run("empty condition => outdated", func(t *testing.T) {
		tp := timeprovider.New()
		dk := &dynatracev1beta2.DynaKube{}

		assert.True(t, IsOutdated(tp, *dk.Conditions(), dk.ApiRequestThreshold(), testingConditionType))
	})

	t.Run("False condition => outdated", func(t *testing.T) {
		tp := timeprovider.New()
		dk := &dynatracev1beta2.DynaKube{}

		SetDynatraceApiError(dk.Conditions(), testingConditionType, errors.New("boom"))

		assert.True(t, IsOutdated(tp, *dk.Conditions(), dk.ApiRequestThreshold(), testingConditionType))
	})

	t.Run("True condition + current timestamp => NOT outdated", func(t *testing.T) {
		tp := timeprovider.New()
		dk := &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				DynatraceApiRequestThreshold: dynatracev1beta2.DefaultMinRequestThresholdMinutes,
			},
		}

		SetSecretCreated(dk.Conditions(), testingConditionType, "")

		assert.False(t, IsOutdated(tp, *dk.Conditions(), dk.ApiRequestThreshold(), testingConditionType))
	})

	t.Run("old timestamp => outdated", func(t *testing.T) {
		tp := timeprovider.New()
		dk := &dynatracev1beta2.DynaKube{}

		SetSecretCreated(dk.Conditions(), testingConditionType, "")
		tp.Set(tp.Now().Add(time.Minute * 60))

		assert.True(t, IsOutdated(tp, *dk.Conditions(), dk.ApiRequestThreshold(), testingConditionType))
	})
}
