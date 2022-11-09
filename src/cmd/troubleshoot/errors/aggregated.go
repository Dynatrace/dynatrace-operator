package errors

import (
	"strings"
)

type AggregatedError struct {
	Errs []error
}

func (errors *AggregatedError) Add(err error) {
	errors.Errs = append(errors.Errs, err)
}

func NewAggregatedError() AggregatedError {
	return AggregatedError{Errs: []error{}}
}

func (e AggregatedError) Error() string {
	sb := strings.Builder{}
	for _, err := range e.Errs {
		sb.WriteString(err.Error() + "\n")
	}
	return sb.String()
}

func (e AggregatedError) Empty() bool {
	return len(e.Errs) == 0
}

var _ error = (*AggregatedError)(nil)
