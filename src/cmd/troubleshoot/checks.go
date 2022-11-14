package troubleshoot

import (
	tserrors "github.com/Dynatrace/dynatrace-operator/src/cmd/troubleshoot/errors"
)

type Result int

const (
	PASSED = iota + 1
	FAILED
	SKIPPED
)

type troubleshootFunc func(troubleshootCtx *troubleshootContext) error

type Check struct {
	Do            troubleshootFunc
	Prerequisites []*Check
}

var results = map[*Check]Result{}

func runChecks(troubleshootCtx *troubleshootContext, checks []*Check) error {
	errs := tserrors.NewAggregatedError()
	for _, check := range checks {
		if shouldSkip(check) {
			results[check] = SKIPPED
			continue
		}

		if err := check.Do(troubleshootCtx); err != nil {
			logErrorf(err.Error())
			errs.Add(err)
			results[check] = FAILED
		} else {
			results[check] = PASSED
		}
	}

	if errs.Empty() {
		return nil
	}

	return errs
}

func shouldSkip(check *Check) bool {
	for _, p := range check.Prerequisites {
		if results[p] != PASSED {
			return true
		}
	}
	return false
}
