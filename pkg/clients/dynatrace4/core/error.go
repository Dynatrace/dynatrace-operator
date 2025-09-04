package core

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ServerError represents an error returned from the server (e.g. authentication failure).
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

// Error formats the server error code and message.
func (e ServerError) Error() string {
	if len(e.Message) == 0 && e.Code == 0 {
		return "unknown server error"
	}

	result := fmt.Sprintf("dynatrace server error %d: %s", e.Code, e.Message)

	for _, constraintViolation := range e.ConstraintViolations {
		result += fmt.Sprintf("\n\t- %s: %s", constraintViolation.Path, constraintViolation.Message)
	}

	return result
}

// handleErrorResponse processes error responses from the API
func (rb *requestBuilder) handleErrorResponse(resp *http.Response, body []byte) error {
	httpErr := &HTTPError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Message:    fmt.Sprintf("HTTP request (%s) failed %d", rb.path, resp.StatusCode),
	}

	// Try to parse as a single server error first
	var singleError struct {
		Error ServerError `json:"error"`
	}

	if err := json.Unmarshal(body, &singleError); err == nil && singleError.Error.Code != 0 {
		httpErr.SingleError = &singleError.Error
		return httpErr
	}

	// Try to parse as array of server errors (common in settings API)
	var errorArray []struct {
		ErrorMessage ServerError `json:"error"`
	}

	if err := json.Unmarshal(body, &errorArray); err == nil && len(errorArray) > 0 {
		httpErr.ServerErrors = make([]ServerError, len(errorArray))
		for i, errItem := range errorArray {
			httpErr.ServerErrors[i] = errItem.ErrorMessage
		}
		return httpErr
	}

	// No parseable server errors found, return generic HTTP error
	return httpErr
}

// HTTPError represents an HTTP error that includes status code, response body, and parsed server errors
type HTTPError struct {
	StatusCode   int           `json:"statusCode"`
	Body         string        `json:"body"`
	Message      string        `json:"message"`
	ServerErrors []ServerError `json:"serverErrors,omitempty"`
	SingleError  *ServerError  `json:"singleError,omitempty"`
}

func (e *HTTPError) Error() string {
	if len(e.ServerErrors) > 0 {
		// Multiple server errors
		var combinedMsg string
		for i, serverErr := range e.ServerErrors {
			if i > 0 {
				combinedMsg += "; "
			}
			combinedMsg += serverErr.Error()
		}
		return fmt.Sprintf("HTTP %d: %s", e.StatusCode, combinedMsg)
	}

	if e.SingleError != nil {
		// Single server error
		return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.SingleError.Error())
	}

	// Generic HTTP error
	return e.Message
}

// // HasServerErrors returns true if the error contains parsed server errors
// func (e *HTTPError) HasServerErrors() bool {
// 	return len(e.ServerErrors) > 0 || e.SingleError != nil
// }

// // GetServerErrors returns all server errors (single error is included in the slice)
// func (e *HTTPError) GetServerErrors() []ServerError {
// 	if e.SingleError != nil {
// 		return []ServerError{*e.SingleError}
// 	}
// 	return e.ServerErrors
// }
