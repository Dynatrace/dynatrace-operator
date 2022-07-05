package pod_mutator

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	injectEvent          = "Inject"
	updatePodEvent       = "UpdatePod"
	missingDynakubeEvent = "MissingDynakube"
)

var (
	log = logger.NewDTLogger().WithName("mutation-webhook.pod")
)
