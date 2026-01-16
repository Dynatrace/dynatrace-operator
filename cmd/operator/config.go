package operator

import "github.com/Dynatrace/dynatrace-operator/pkg/logd"

const (
	metricsBindAddress     = ":8080"
	healthProbeBindAddress = ":10080"

	leaderElectionID                  = "dynatrace-operator-lock"
	leaderElectionResourceLock        = "leases"
	leaderElectionEnvVarRenewDeadline = "LEADER_ELECTION_RENEW_DEADLINE"
	leaderElectionEnvVarRetryPeriod   = "LEADER_ELECTION_RETRY_PERIOD"
	leaderElectionEnvVarLeaseDuration = "LEADER_ELECTION_LEASE_DURATION"

	livezEndpointName    = "livez"
	livenessEndpointName = "/" + livezEndpointName

	defaultLeaseDuration = int64(30)
	defaultRenewDeadline = int64(20)
	defaultRetryPeriod   = int64(6)
)

var log = logd.Get().WithName("operator-command")
