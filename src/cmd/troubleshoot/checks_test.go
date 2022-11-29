package troubleshoot

import (
	"testing"

	tserrors "github.com/Dynatrace/dynatrace-operator/src/cmd/troubleshoot/errors"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var (
	checkError = errors.New("check function failed")

	tsContext = &troubleshootContext{}

	passingBasicCheck = &Check{
		Name: "passingBasicCheck",
		Do: func(*troubleshootContext) error {
			return nil
		},
	}

	failingBasicCheck = &Check{
		Name: "failingBasicCheck",
		Do: func(*troubleshootContext) error {
			return checkError
		},
	}

	passingCheckDependendOnPassingCheck = &Check{
		Name: "passingCheckDependendOnPassingCheck",
		Do: func(*troubleshootContext) error {
			return nil
		},
		Prerequisites: []*Check{passingBasicCheck},
	}

	failingCheckDependendOnPassingCheck = &Check{
		Name: "failingCheckDependendOnPassingCheck",
		Do: func(*troubleshootContext) error {
			return checkError
		},
		Prerequisites: []*Check{passingBasicCheck},
	}

	failingCheckDependendOnFailingCheck = &Check{
		Name: "failingCheckDependendOnFailingCheck",
		Do: func(*troubleshootContext) error {
			return checkError
		},
		Prerequisites: []*Check{failingBasicCheck},
	}
)

func Test_runChecks(t *testing.T) {
	t.Run("no checks", func(t *testing.T) {
		checks := []*Check{}
		results := NewChecksResults()
		err := runChecks(results, tsContext, checks)
		require.NoError(t, err)
	})
	t.Run("a few passing checks", func(t *testing.T) {
		checks := []*Check{
			passingBasicCheck,
			passingCheckDependendOnPassingCheck,
		}
		results := NewChecksResults()
		err := runChecks(results, tsContext, checks)
		require.NoError(t, err)
	})
	t.Run("passing and failing checks", func(t *testing.T) {
		checks := []*Check{
			passingBasicCheck,
			passingCheckDependendOnPassingCheck,
			failingCheckDependendOnPassingCheck,
			failingBasicCheck,
			failingCheckDependendOnFailingCheck, // should be skipped and error should not be reported
		}
		results := NewChecksResults()
		resetLogger()
		err := runChecks(results, tsContext, checks)
		require.Error(t, err)

		aggregatedError := tserrors.AggregatedError{}
		isAggredatedError := errors.As(err, &aggregatedError)

		require.True(t, isAggredatedError)
		require.Len(t, aggregatedError.Errs, 2)
		require.ErrorIs(t, aggregatedError.Errs[0], checkError)
		require.ErrorIs(t, aggregatedError.Errs[1], checkError)
	})
	t.Run("check should not be run if prerequisite check failed", func(t *testing.T) {
		checks := []*Check{
			failingBasicCheck,
			failingCheckDependendOnFailingCheck, // should be skipped and error should not be reported
		}
		results := NewChecksResults()
		err := runChecks(results, tsContext, checks)
		require.Error(t, err)
	})
}
