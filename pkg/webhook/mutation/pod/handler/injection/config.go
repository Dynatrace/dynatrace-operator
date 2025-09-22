package injection

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("pod-mutation-injection")
)

const (
	K8sNodeNameEnv = "K8S_NODE_NAME"
	K8sPodNameEnv  = "K8S_PODNAME"
	K8sPodUIDEnv   = "K8S_PODUID"

	NoBootstrapperConfigReason = "NoBootstrapperConfig"
	NoMutationNeededReason     = "NoMutationNeeded"

	RootUser  int64 = 0
	RootGroup int64 = 0
)
