package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestartReconciliationError(t *testing.T) {
	// Create error
	err := NewRestartReconciliationError("test error")

	targetErr := &RestartReconciliationError{}
	// Check error type
	assert.IsType(t, targetErr, err)

	// Check Is method
	assert.True(t, errors.Is(err, targetErr))

	require.True(t, errors.As(err, &targetErr))

	// Check error message
	assert.Equal(t, "test error", targetErr.Error())
}
