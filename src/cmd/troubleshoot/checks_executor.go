package troubleshoot

import (
	tserrors "github.com/Dynatrace/dynatrace-operator/src/cmd/troubleshoot/errors"
	"github.com/pkg/errors"
)

func runChecks(troubleshootCtx *troubleshootContext, checks []troubleshootFunc) error {
	errs := tserrors.NewAggregatedError()
	for _, check := range checks {
		if err := check(troubleshootCtx); err != nil {
			logErrorf(err.Error())
			errs.Add(err)
			if errors.Is(err, tserrors.CardinalProblemError) {
				break
			}
		}
	}

	if errs.Empty() {
		return nil
	}

	return errs
}
