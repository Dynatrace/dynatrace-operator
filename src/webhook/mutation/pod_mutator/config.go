package pod_mutator

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	injectEvent          = "Inject"
	updatePodEvent       = "UpdatePod"
	IncompatibleCRDEvent = "IncompatibleCRDPresent"
	missingDynakubeEvent = "MissingDynakube"
)

var (
	log = logger.NewDTLogger().WithName("mutation-webhook.pod")
)
