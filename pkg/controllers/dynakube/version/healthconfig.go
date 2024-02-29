package version

import (
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dtversion"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	"golang.org/x/mod/semver"
)

const (
	semverPrefix = "v"

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

	if agentVersion != "" && agentVersion != string(status.CustomImageVersionSource) {
		agentSemver, err := dtversion.ToSemver(agentVersion)
		if err != nil {
			return nil, err
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
