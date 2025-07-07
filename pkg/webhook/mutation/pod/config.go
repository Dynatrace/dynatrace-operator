package pod

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("pod-mutation")
)

const (
	NoBootstrapperConfigReason = "NoBootstrapperConfig"
	NoMutationNeededReason     = "NoMutationNeeded"
)
