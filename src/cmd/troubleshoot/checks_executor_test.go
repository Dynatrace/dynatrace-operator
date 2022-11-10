package troubleshoot

import (
	"testing"

	tserrors "github.com/Dynatrace/dynatrace-operator/src/cmd/troubleshoot/errors"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var checkError = errors.New("check function failed")

func newCheck(shouldSucceed bool, isCardinal bool) Check {
	return Check{
		Do: func(troubleshootCtx *troubleshootContext) error {
			if shouldSucceed {
				return nil
			}

			if isCardinal {
				return errors.Wrap(tserrors.CardinalProblemError, checkError.Error())
			}
			return checkError
		},
		Name: "check",
	}
}

func Test_runChecks(t *testing.T) {
	context := &troubleshootContext{}

	nonCardinalPassingCheck := newCheck(true, false)
	cardinalPassingCheck := newCheck(true, true)
	nonCardinalFailingCheck := newCheck(false, false)
	cardinalFailingCheck := newCheck(false, true)

	t.Run("no checks", func(t *testing.T) {
		checks := []Check{}
		err := runChecks(context, checks)
		require.NoError(t, err)
	})
	t.Run("a few passing checks", func(t *testing.T) {
		checks := []Check{
			cardinalPassingCheck,
			nonCardinalPassingCheck,
		}
		err := runChecks(context, checks)
		require.NoError(t, err)
	})
	t.Run("a few passing, one failing checks", func(t *testing.T) {
		checks := []Check{
			cardinalPassingCheck,
			nonCardinalPassingCheck,
			nonCardinalFailingCheck,
		}
		err := runChecks(context, checks)
		require.Error(t, err)
		require.Contains(t, err.(tserrors.AggregatedError).Errs, checkError)
		require.NotContains(t, err.(tserrors.AggregatedError).Errs, tserrors.CardinalProblemError)
	})
	t.Run("stop on failing cardinal check", func(t *testing.T) {
		checks := []Check{
			cardinalPassingCheck,
			nonCardinalFailingCheck,
			cardinalFailingCheck, // checks execution should stop on this check
			nonCardinalPassingCheck,
			nonCardinalFailingCheck,
		}

		err := runChecks(context, checks)
		require.Error(t, err)

		aggregatedError := err.(tserrors.AggregatedError)

		require.Len(t, aggregatedError.Errs, 2)
		require.ErrorIs(t, aggregatedError.Errs[0], checkError)
		require.ErrorIs(t, aggregatedError.Errs[1], tserrors.CardinalProblemError)
	})
}
