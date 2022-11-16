package troubleshoot

import (
	"strings"

	tserrors "github.com/Dynatrace/dynatrace-operator/src/cmd/troubleshoot/errors"
	"github.com/Dynatrace/dynatrace-operator/src/functional"
)

type Result int

const (
	PASSED Result = iota + 1
	FAILED
	SKIPPED
)

type troubleshootFunc func(troubleshootCtx *troubleshootContext) error

type Check struct {
	Do            troubleshootFunc
	Prerequisites []*Check
	Name          string
}

type ChecksResults struct {
	check2Result map[*Check]Result
}

func NewChecksResults() ChecksResults {
	return ChecksResults{check2Result: map[*Check]Result{}}
}

func (cr ChecksResults) set(check *Check, result Result) {
	cr.check2Result[check] = result
}

func (cr ChecksResults) failedPrerequisites(check *Check) []*Check {
	isFailed := func(c *Check) bool {
		return cr.check2Result[c] == FAILED
	}
	return functional.Filter(check.Prerequisites, isFailed)
}

func runChecks(results ChecksResults, troubleshootCtx *troubleshootContext, checks []*Check) error {
	errs := tserrors.NewAggregatedError()

	for _, check := range checks {
		if shouldSkip(results, check) {
			results.set(check, SKIPPED)
			continue
		}

		err := check.Do(troubleshootCtx)
		if err != nil {
			logErrorf(err.Error())
			errs.Add(err)
			results.set(check, FAILED)
		} else {
			results.set(check, PASSED)
		}
	}

	if errs.Empty() {
		return nil
	}

	return errs
}

func shouldSkip(results ChecksResults, check *Check) bool {
	failedPrerequisites := results.failedPrerequisites(check)

	if len(failedPrerequisites) == 0 {
		return false
	}

	getCheckName := func(c *Check) string {
		return c.Name
	}
	prereqsNames := strings.Join(functional.Map(failedPrerequisites, getCheckName), ",")
	logWarningf("Skipped '%s' check because prerequisites aren't met: [%s]", check.Name, prereqsNames)

	return true
}
