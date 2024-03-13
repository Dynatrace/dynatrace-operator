package pod_mutator

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logd"
)

const (
	injectEvent          = "Inject"
	updatePodEvent       = "UpdatePod"
	IncompatibleCRDEvent = "IncompatibleCRDPresent"
	missingDynakubeEvent = "MissingDynakube"

	defaultUser   int64 = 1001
	defaultGroup  int64 = 1001
	rootUserGroup int64 = 0
)

var (
	log = logd.Get().WithName("pod-mutation")
)
