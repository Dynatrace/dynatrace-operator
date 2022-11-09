package troubleshoot

import (
	"testing"

	tserrors "github.com/Dynatrace/dynatrace-operator/src/cmd/troubleshoot/errors"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var checkError = errors.New("check function failed")

func createCheckFunction(shouldSucceed bool, isCardinal bool) troubleshootFunc {
	return func(troubleshootCtx *troubleshootContext) error {
		if shouldSucceed {
			return nil
		}

		if isCardinal {
			return errors.Wrap(tserrors.CardinalProblemError, checkError.Error())
		}
		return checkError
	}
}

func Test_runChecks(t *testing.T) {
	context := &troubleshootContext{}

	nonCardinalPassingCheck := createCheckFunction(true, false)
	cardinalPassingCheck := createCheckFunction(true, true)
	nonCardinalFailingCheck := createCheckFunction(false, false)
	cardinalFailingCheck := createCheckFunction(false, true)

	t.Run("no checks", func(t *testing.T) {
		checks := []troubleshootFunc{}
		err := runChecks(context, checks)
		require.NoError(t, err)
	})
	t.Run("a few passing checks", func(t *testing.T) {
		checks := []troubleshootFunc{
			cardinalPassingCheck,
			nonCardinalPassingCheck,
		}
		err := runChecks(context, checks)
		require.NoError(t, err)
	})
	t.Run("a few passing, one failing checks", func(t *testing.T) {
		checks := []troubleshootFunc{
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
		checks := []troubleshootFunc{
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
