package dynatrace

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type serverErrorResponse struct {
	ErrorMessage ServerError `json:"error"`
}

type ConstraintViolation struct {
	Description       string `json:"description"`
	Location          string `json:"location"`
	Message           string `json:"message"`
	ParameterLocation string `json:"parameterLocation"`
	Path              string `json:"path"`
}

// ServerError represents an error returned from the server (e.g. authentication failure).
type ServerError struct {
	Message              string                `json:"message"`
	ConstraintViolations []ConstraintViolation `json:"constraintViolations,omitempty"`
	Code                 int                   `json:"code"`
}

// Error formats the server error code and message.
func (e ServerError) Error() string {
	if len(e.Message) == 0 && e.Code == 0 {
		return "unknown server error"
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("dynatrace server error %d: %s", int64(e.Code), e.Message))

	for _, constraintViolation := range e.ConstraintViolations {
		sb.WriteString(fmt.Sprintf("\n\t- %s: %s", constraintViolation.Path, constraintViolation.Message))
	}

	return sb.String()
}

func hasServerErrorCode(err error, status int) bool {
	var serverErr ServerError

	ok := errors.As(err, &serverErr)

	return ok && serverErr.Code == status
}
