package istio

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	log = logger.Factory.GetLogger("dynakube-istio")
)

const (
	OperatorComponent = "operator"
	OneAgentComponent = "oneagent"
)
