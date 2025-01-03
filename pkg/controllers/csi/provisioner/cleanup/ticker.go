package cleanup

import (
	"os"
	"sync"
	"time"
)

const (
	defaultCleanupPeriod = "5m"
	cleanupEnv           = "CLEANUP_PERIOD"
)

var (
	cleanupPeriod time.Duration
	ticker        *time.Ticker
)

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
