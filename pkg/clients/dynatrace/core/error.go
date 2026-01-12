package core

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ServerError represents an error returned from the server (e.g. authentication failure)
type ServerError struct {
	Message              string                `json:"message"`
	ConstraintViolations []ConstraintViolation `json:"constraintViolations,omitempty"`
	Code                 int                   `json:"code"`
}

// ConstraintViolation represents a constraint violation in server errors
type ConstraintViolation struct {
	Description       string `json:"description"`
	Location          string `json:"location"`
	Message           string `json:"message"`
	ParameterLocation string `json:"parameterLocation"`
	Path              string `json:"path"`
}

func (e *ServerError) Error() string {
	if len(e.Message) == 0 && e.Code == 0 {
		return "unknown server error"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "dynatrace server error %d: %s", e.Code, e.Message)

	for _, v := range e.ConstraintViolations {
		// Fprintf can cause up to 100x allocations
		sb.WriteString("\n\t- ")
		sb.WriteString(v.Path)
		sb.WriteString(": ")
		sb.WriteString(v.Message)
	}

	return sb.String()
}

// HTTPError represents an HTTP error that includes status code, response body, and parsed server errors
type HTTPError struct {
	Body         string        `json:"body"`
	Message      string        `json:"message"`
	ServerErrors []ServerError `json:"serverErrors,omitempty"`
	StatusCode   int           `json:"statusCode"`
}

func (e *HTTPError) Error() string {
	if len(e.ServerErrors) > 0 {
		var sb strings.Builder

		for i, serverErr := range e.ServerErrors {
			if i > 0 {
				sb.WriteString("; ")
			}

			sb.WriteString(serverErr.Error())
		}

		return fmt.Sprintf("HTTP %d: %s", e.StatusCode, sb.String())
	}

	return e.Message
}

// HasStatusCode checks if the given error is an HTTPError with the specified status code
func HasStatusCode(err error, statusCode int) bool {
	httpErr := new(HTTPError)

	return errors.As(err, &httpErr) && httpErr.StatusCode == statusCode
}

// IsNotFound checks if the given error represents an HTTP 404 Not Found error
func IsNotFound(err error) bool {
	return HasStatusCode(err, http.StatusNotFound)
}
