package version

import (
	"strings"
	"time"

	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
)

const (
	semverPrefix = "v"
	semverSep    = "."

	// relevantVersionLength defines how many segments of the semver we care about, for-example:
	// in case of 3, we only care about major.minor.patch and the rest is ignored.
	relevantSemverLength = 3

	// healthCheckVersionThreshold hold the semver after which point the binary-based health-check can be used.
	healthCheckVersionThreshold = semverPrefix + "1.276"

	defaultHealthConfigInterval    = 10 * time.Second
	defaultHealthConfigStartPeriod = 1200 * time.Second
	defaultHealthConfigTimeout     = 30 * time.Second
	defaultHealthConfigRetries     = 3
)

var (
	preThresholdHealthCheck = []string{"/bin/sh", "-c", "grep -q oneagentwatchdo /proc/[0-9]*/stat"}
	currentHealthCheck      = []string{"/usr/bin/watchdog-healthcheck64"}
)

// Constructor setting default values for docker image HealthConfig
func newHealthConfig(command []string) *containerv1.HealthConfig {
	return &containerv1.HealthConfig{
		Test:        command,
		Interval:    defaultHealthConfigInterval,
		StartPeriod: defaultHealthConfigStartPeriod,
		Timeout:     defaultHealthConfigTimeout,
		Retries:     defaultHealthConfigRetries,
	}
}

func getOneAgentHealthConfig(agentVersion string) (*containerv1.HealthConfig, error) {
	var testCommand []string

	if agentVersion != "" {
		agentSemver := agentVersionToSemver(agentVersion)
		if !semver.IsValid(agentSemver) {
			return nil, errors.Errorf("provided oneagent version %s is not a valid semver", agentVersion)
		}
		// threshold > agentSemver == 1
		// threshold < agentSemver == -1
		// threshold == agentSemver == 0
		switch semver.Compare(healthCheckVersionThreshold, agentSemver) {
		case 1:
			testCommand = preThresholdHealthCheck
		default:
			testCommand = currentHealthCheck
		}
	} else {
		testCommand = currentHealthCheck
	}

	return newHealthConfig(testCommand), nil
}

func agentVersionToSemver(agentVersion string) string {
	if agentVersion == "" {
		return ""
	}

	split := strings.Split(agentVersion, semverSep)

	var agentSemver string

	if len(split) > relevantSemverLength {
		agentSemver = strings.Join(split[:relevantSemverLength], semverSep)
	} else {
		agentSemver = strings.Join(split, semverSep)
	}

	if !strings.HasPrefix(agentSemver, semverPrefix) {
		agentSemver = semverPrefix + agentSemver
	}

	return semver.Canonical(agentSemver)
}
