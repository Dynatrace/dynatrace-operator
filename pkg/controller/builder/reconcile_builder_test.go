package builder

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReconcileAfter(t *testing.T) {
	duration := 1 * time.Minute
	result := ReconcileAfter(duration)
	assert.NotNil(t, result)
	assert.Equal(t, duration, result.RequeueAfter)
	assert.False(t, result.Requeue)

	duration = 5 * time.Second
	result = ReconcileAfter(duration)
	assert.NotNil(t, result)
	assert.Equal(t, duration, result.RequeueAfter)
	assert.False(t, result.Requeue)
}

func TestReconcileAfterFiveMinutes(t *testing.T) {
	result := ReconcileAfterFiveMinutes()
	assert.NotNil(t, result)
	assert.Equal(t, 5*time.Minute, result.RequeueAfter)
	assert.False(t, result.Requeue)
}
