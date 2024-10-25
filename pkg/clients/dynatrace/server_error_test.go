package dynatrace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerErrorString(t *testing.T) {
	t.Run("no constraint violation", func(t *testing.T) {
		serverError := ServerError{
			Message:              "an error",
			ConstraintViolations: nil,
			Code:                 1,
		}

		assert.Equal(t, "dynatrace server error 1: an error", serverError.Error())
	})

	t.Run("single constraint violation", func(t *testing.T) {
		serverError := ServerError{
			Message: "an error",
			ConstraintViolations: []ConstraintViolation{
				{
					Message: "message",
					Path:    "path",
				},
			},
			Code: 1,
		}

		assert.Equal(t, "dynatrace server error 1: an error\n\t- path: message", serverError.Error())
	})

	t.Run("two constraint violations", func(t *testing.T) {
		serverError := ServerError{
			Message: "an error",
			ConstraintViolations: []ConstraintViolation{
				{
					Message: "message",
					Path:    "path",
				},
				{
					Message: "message2",
					Path:    "path2",
				},
			},
			Code: 1,
		}

		assert.Equal(t, "dynatrace server error 1: an error\n\t- path: message\n\t- path2: message2", serverError.Error())
	})
}
