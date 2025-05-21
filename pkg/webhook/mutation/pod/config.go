package pod

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("pod-mutation")
)

import "github.com/Dynatrace/dynatrace-operator/pkg/logd"

// var (
// 	log = logd.Get().WithName("v2-pod-mutation")
// )

const (
	NoBootstrapperConfigReason = "NoBootstrapperConfig"
	NoCodeModulesImageReason   = "NoCodeModulesImage"
	NoMutationNeededReason     = "NoMutationNeeded"
)
