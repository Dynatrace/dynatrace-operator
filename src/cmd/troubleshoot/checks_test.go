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
		Do: func(*troubleshootContext) error {
			return nil
		},
	}

	failingBasicCheck = &Check{
		Do: func(*troubleshootContext) error {
			return checkError
		},
	}

	passingCheckDependendOnPassingCheck = &Check{
		Do: func(*troubleshootContext) error {
			return nil
		},
		Prerequisites: []*Check{passingBasicCheck},
	}

	failingCheckDependendOnPassingCheck = &Check{
		Do: func(*troubleshootContext) error {
			return checkError
		},
		Prerequisites: []*Check{passingBasicCheck},
	}

	failingCheckDependendOnFailingCheck = &Check{
		Do: func(*troubleshootContext) error {
			return checkError
		},
		Prerequisites: []*Check{failingBasicCheck},
	}
)

func Test_runChecks(t *testing.T) {
	t.Run("no checks", func(t *testing.T) {
		checks := []*Check{}
		err := runChecks(tsContext, checks)
		require.NoError(t, err)
	})
	t.Run("a few passing checks", func(t *testing.T) {
		checks := []*Check{
			passingBasicCheck,
			passingCheckDependendOnPassingCheck,
		}
		err := runChecks(tsContext, checks)
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

		err := runChecks(tsContext, checks)
		require.Error(t, err)

		aggregatedError := err.(tserrors.AggregatedError)

		require.Len(t, aggregatedError.Errs, 2)
		require.ErrorIs(t, aggregatedError.Errs[0], checkError)
		require.ErrorIs(t, aggregatedError.Errs[1], checkError)
	})
	t.Run("check should not be run if prerequisite check failed", func(t *testing.T) {
		checks := []*Check{
			failingBasicCheck,
			failingCheckDependendOnFailingCheck, // should be skipped and error should not be reported
		}

		err := runChecks(tsContext, checks)
		require.Error(t, err)
	})
}
