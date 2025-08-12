package timeprovider

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testTimeout = 10 * time.Minute

func TestTimeoutReached(t *testing.T) {
	t.Run("returns true if the previous timestamp is too far in the past", func(t *testing.T) {
		now := now()
		outdated := metav1.NewTime(now.Add(time.Duration(-16) * time.Minute))
		assert.True(t, TimeoutReached(&outdated, now, testTimeout))
	})
	t.Run("returns false if the previous and current timestamp are the same", func(t *testing.T) {
		now := now()
		notOutdated := now
		assert.False(t, TimeoutReached(notOutdated, now, testTimeout))
	})
	t.Run("returns false if the previous timestamp is within the timeout period", func(t *testing.T) {
		now := now()
		notOutdated := metav1.NewTime(now.Add(time.Duration(-5) * time.Minute))
		assert.False(t, TimeoutReached(&notOutdated, now, testTimeout))
	})
	t.Run("returns true if the previous timestamp is nil", func(t *testing.T) {
		now := now()
		assert.True(t, TimeoutReached(nil, now, testTimeout))
	})
	t.Run("returns true if the current timestamp is nil", func(t *testing.T) {
		now := now()
		assert.True(t, TimeoutReached(now, nil, testTimeout))
	})
	t.Run("returns true if the previous and the current timestamp are nil", func(t *testing.T) {
		assert.True(t, TimeoutReached(nil, nil, testTimeout))
	})
}

func TestIsOutdated(t *testing.T) {
	provider := New().Freeze()
	now := provider.Now()

	t.Run("returns true if the previous timestamp is too far in the past", func(t *testing.T) {
		outdated := metav1.NewTime(now.Add(time.Duration(-16) * time.Minute))
		assert.True(t, provider.IsOutdated(&outdated, testTimeout))
	})
	t.Run("returns false if the previous and current timestamp are the same", func(t *testing.T) {
		assert.False(t, provider.IsOutdated(now, testTimeout))
	})
	t.Run("returns false if the previous timestamp is within the timeout period", func(t *testing.T) {
		notOutdated := metav1.NewTime(now.Add(time.Duration(-5) * time.Minute))
		assert.False(t, provider.IsOutdated(&notOutdated, testTimeout))
	})
	t.Run("returns true if the previous timestamp is nil", func(t *testing.T) {
		assert.True(t, provider.IsOutdated(nil, testTimeout))
	})
}

func TestTimeProvider(t *testing.T) {
	t.Run("timestamp is not set", func(t *testing.T) {
		provider := New()
		earlier := provider.Now()
		later := provider.Now()
		assert.False(t, earlier.Equal(later))
		assert.True(t, earlier.Before(later))
	})

	t.Run("timestamp is set", func(t *testing.T) {
		provider := New().Freeze()
		earlier := provider.Now()
		later := provider.Now()
		assert.True(t, earlier.Equal(later))
	})
}
