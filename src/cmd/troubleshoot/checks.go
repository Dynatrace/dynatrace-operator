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
	checkResultMap map[*Check]Result
}

func NewChecksResults() ChecksResults {
	return ChecksResults{checkResultMap: map[*Check]Result{}}
}

func (checkResults ChecksResults) resetResults(keepChecks []*Check) map[*Check]Result {
	keptResults := map[*Check]Result{}

	for _, check := range keepChecks {
		keptResults[check] = checkResults.checkResultMap[check]
	}

	return keptResults
}

func (checkResults ChecksResults) set(check *Check, result Result) {
	checkResults.checkResultMap[check] = result
}

func (checkResults ChecksResults) failedPrerequisites(check *Check) []*Check {
	isFailed := func(check *Check) bool {
		return checkResults.checkResultMap[check] == FAILED
	}
	return functional.Filter(check.Prerequisites, isFailed)
}

func (checkResults ChecksResults) hasErrors() bool {
	for _, result := range checkResults.checkResultMap {
		if result == FAILED {
			return true
		}
	}
	return false
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

	getCheckName := func(check *Check) string {
		return check.Name
	}
	prerequisitesNames := strings.Join(functional.Map(failedPrerequisites, getCheckName), ",")
	logWarningf("Skipped '%s' check because prerequisites aren't met: [%s]", check.Name, prerequisitesNames)

	return true
}
