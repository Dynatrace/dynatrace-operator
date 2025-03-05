package oneagent

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("oneagent-pod-v2-mutation")
)

const (
	NoBootstrapperConfigReason = "NoBootstrapperConfig"
)
