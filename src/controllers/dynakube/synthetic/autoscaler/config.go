package autoscaler

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const SynAutoscaler = "syn-autoscaler"

var (
	log = logger.Factory.GetLogger(SynAutoscaler)
)
