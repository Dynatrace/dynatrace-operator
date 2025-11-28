package injection

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("pod-mutation-injection")
)

const (
	NoBootstrapperConfigReason = "NoBootstrapperConfig"
	NoMutationNeededReason     = "NoMutationNeeded"

	RootUser  int64 = 0
	RootGroup int64 = 0
)
