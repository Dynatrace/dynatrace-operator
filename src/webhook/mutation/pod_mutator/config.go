package pod_mutator

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	injectEvent          = "Inject"
	updatePodEvent       = "UpdatePod"
	missingDynakubeEvent = "MissingDynakube"

	injectionConfigVolumeName = "injection-config"
)

var (
	log = logger.NewDTLogger().WithName("pod.mutation-webhook")
)
