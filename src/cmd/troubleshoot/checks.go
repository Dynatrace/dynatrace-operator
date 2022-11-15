package troubleshoot

import (
	"strings"

	tserrors "github.com/Dynatrace/dynatrace-operator/src/cmd/troubleshoot/errors"
	"github.com/Dynatrace/dynatrace-operator/src/functional"
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
	Name          string
}

var results = map[*Check]Result{}

func runChecks(troubleshootCtx *troubleshootContext, checks []*Check) error {
	errs := tserrors.NewAggregatedError()
	for _, check := range checks {
		if shouldSkip(check) {
			prereqs := strings.Join(functional.Map(check.Prerequisites, func(c *Check) string { return c.Name }), ",")
			logWarningf("Skipped '%s' check because prerequisites aren't met: [%s]", check.Name, prereqs)
			results[check] = SKIPPED
			continue
		}

		err := check.Do(troubleshootCtx)
		if err != nil {
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
