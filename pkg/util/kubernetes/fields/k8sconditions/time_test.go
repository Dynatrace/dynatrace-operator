// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8sconditions

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsOutdated(t *testing.T) {
	testingConditionType := "testing "

	t.Run("empty condition => outdated", func(t *testing.T) {
		dk := &dynakube.DynaKube{}

		assert.True(t, IsOutdated(dk, testingConditionType))
	})

	t.Run("False condition => outdated", func(t *testing.T) {
		dk := &dynakube.DynaKube{}

		SetDynatraceAPIError(dk.Conditions(), testingConditionType, errors.New("boom"))

		assert.True(t, IsOutdated(dk, testingConditionType))
	})

	t.Run("False condition with OptionalScopeMissing reason => outdated only after threshold", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			dk := &dynakube.DynaKube{}

			SetOptionalScopeMissing(dk.Conditions(), testingConditionType, "scope missing")

			assert.False(t, IsOutdated(dk, testingConditionType))

			synctest.Sleep(dk.APIRequestThreshold())

			assert.True(t, IsOutdated(dk, testingConditionType))
		})
	})

	t.Run("True condition + current timestamp => NOT outdated", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{},
		}

		SetSecretCreated(dk.Conditions(), testingConditionType, "")

		assert.False(t, IsOutdated(dk, testingConditionType))
	})

	t.Run("old timestamp => outdated", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			dk := &dynakube.DynaKube{}

			SetSecretCreated(dk.Conditions(), testingConditionType, "")

			// set time to just before the API request threshold
			apiRequestThreshold := dk.APIRequestThreshold()
			synctest.Sleep(apiRequestThreshold - time.Nanosecond)

			assert.False(t, IsOutdated(dk, testingConditionType))

			// set time just at the API request threshold
			synctest.Sleep(time.Nanosecond)

			assert.True(t, IsOutdated(dk, testingConditionType))
		})
	})

	t.Run("custom APIRequestThreshold is respected", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			thresholdInMinutes := 5
			dk := &dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					DynatraceAPIRequestThreshold: new(uint16(thresholdInMinutes)),
				},
			}

			SetSecretCreated(dk.Conditions(), testingConditionType, "")

			synctest.Sleep(time.Duration(thresholdInMinutes)*time.Minute - time.Nanosecond)

			assert.False(t, IsOutdated(dk, testingConditionType))

			synctest.Sleep(time.Nanosecond)

			assert.True(t, IsOutdated(dk, testingConditionType))
		})
	})

	t.Run("zero LastTransitionTime => outdated", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				Conditions: []metav1.Condition{
					{
						Type:   testingConditionType,
						Status: metav1.ConditionTrue,
					},
				},
			},
		}

		assert.True(t, IsOutdated(dk, testingConditionType))
	})
}
