package cleanup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTicker(t *testing.T) {
	customCleanUpPeriod := "22m"
	t.Setenv(cleanupEnv, customCleanUpPeriod)

	parsedDuration, _ := time.ParseDuration(customCleanUpPeriod)

	// Initially cleanup period  is not set
	require.Empty(t, cleanupPeriod)

	// Initially ticker is not set
	require.Nil(t, ticker)

	// Works with nil ticker
	resetTickerAfterDelete()

	// ticker is now set
	require.NotNil(t, ticker)
	// cleanup period is now set and respects env
	require.Equal(t, parsedDuration, cleanupPeriod)

	// Works with not-nil ticker
	resetTickerAfterDelete()

	ticker.Stop()
	ticker = nil

	// Works with nil ticker
	resetFunc := checkTicker()
	require.NotNil(t, resetFunc)
	require.Nil(t, ticker)

	resetFunc()
	require.NotNil(t, ticker)

	// Works with not-nil ticker
	resetFunc = checkTicker()
	require.Nil(t, resetFunc)

	ticker.Stop()
}
