package core

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerError(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		serverErr := &ServerError{}
		assert.EqualError(t, serverErr, "unknown server error")
	})

	t.Run("simple", func(t *testing.T) {
		serverErr := &ServerError{Code: 404, Message: "not found"}
		assert.EqualError(t, serverErr, "dynatrace server error 404: not found")
	})

	t.Run("single constraint", func(t *testing.T) {
		serverErr := &ServerError{
			Code:    422,
			Message: "invalid",
			ConstraintViolations: []ConstraintViolation{
				{
					Description:       "foo",
					Location:          "bar",
					Message:           "test message",
					ParameterLocation: "baz",
					Path:              "test path",
				},
			},
		}
		assert.EqualError(t, serverErr, "dynatrace server error 422: invalid\n\t- test path: test message")
	})

	t.Run("multiple constraints", func(t *testing.T) {
		serverErr := &ServerError{
			Code:    422,
			Message: "invalid",
			ConstraintViolations: []ConstraintViolation{
				{Message: "message1", Path: "path1"},
				{Message: "message2", Path: "path2"},
			},
		}
		assert.EqualError(t, serverErr, "dynatrace server error 422: invalid\n\t- path1: message1\n\t- path2: message2")
	})
}

func TestHTTPError(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		httpErr := &HTTPError{
			StatusCode: 404,
			Message:    "not found",
		}
		assert.EqualError(t, httpErr, "not found")
	})

	t.Run("single server error", func(t *testing.T) {
		httpErr := &HTTPError{
			StatusCode: 401,
			ServerErrors: []ServerError{
				{Code: 401, Message: "unauthorized"},
			},
		}
		assert.EqualError(t, httpErr, "HTTP 401: dynatrace server error 401: unauthorized")
	})

	t.Run("multiple server errors", func(t *testing.T) {
		httpErr := &HTTPError{
			StatusCode: 400,
			ServerErrors: []ServerError{
				{Code: 400, Message: "bad1"},
				{Code: 400, Message: "bad2"},
			},
		}
		assert.EqualError(t, httpErr, "HTTP 400: dynatrace server error 400: bad1; dynatrace server error 400: bad2")
	})
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name   string
		in     error
		expect bool
	}{
		{"nil", nil, false},
		{"no http error", errors.New("BOOM"), false},
		{"wrong status code", &HTTPError{StatusCode: 401}, false},
		{"matching status code", &HTTPError{StatusCode: 404}, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expect, IsNotFound(test.in))
		})
	}
}
