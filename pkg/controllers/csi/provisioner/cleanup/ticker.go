package cleanup

import (
	"os"
	"sync"
	"time"
)

const (
	defaultCleanupPeriod = "30m"
	cleanupEnv           = "CLEANUP_PERIOD"
)

var (
	cleanupPeriod time.Duration
	ticker        *time.Ticker
)

// checkTicker will initialize (if needed) and check the a ticker if enough time has passed since the last cleanup
func checkTicker() func() {
	setupCleanUpPeriod()

	if ticker == nil {
		log.Info("initial run of CSI filesystem cleanup")

		return func() {
			ticker = time.NewTicker(cleanupPeriod)
		}
	}

	select {
	case <-ticker.C:
		log.Info("running CSI filesystem cleanup")

		return func() {
			ticker.Reset(cleanupPeriod)
		}
	default:
		log.Info("skipping CSI filesystem cleanup, it only runs every given period", "period", cleanupPeriod.String())

		return nil
	}
}

func setupCleanUpPeriod() {
	sync.OnceFunc(func() {
		rawDuration := os.Getenv(cleanupEnv)

		duration, err := time.ParseDuration(rawDuration)
		if err != nil {
			if rawDuration != "" {
				log.Info("custom cleanup period could be parsed, falling back to default", "env", cleanupEnv, "value", rawDuration, "default", defaultCleanupPeriod)
			}

			duration, _ = time.ParseDuration(defaultCleanupPeriod)
		}

		cleanupPeriod = duration
	})()
}

// resetTickerAfterDelete is for the specific scenario of dynakube deletion
// its purpose is to reset the ticker safely, but not check it, so the cleanup will always run after a DynaKube deletion
// meant to be called via defer
func resetTickerAfterDelete() {
	setupCleanUpPeriod()

	if ticker == nil {
		log.Info("initial run of CSI filesystem cleanup")

		ticker = time.NewTicker(cleanupPeriod)
	} else {
		ticker.Reset(cleanupPeriod)
	}
}
