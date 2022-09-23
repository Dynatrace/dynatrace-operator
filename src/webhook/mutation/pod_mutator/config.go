package pod_mutator

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	injectEvent          = "Inject"
	updatePodEvent       = "UpdatePod"
	IncompatibleCRDEvent = "IncompatibleCRDPresent"
	missingDynakubeEvent = "MissingDynakube"

	defaultUser int64 = 1001
	defaultGroup int64 = 1001
	rootUser int64 = 0
)

var (
	log = logger.NewDTLogger().WithName("mutation-webhook.pod")
)
