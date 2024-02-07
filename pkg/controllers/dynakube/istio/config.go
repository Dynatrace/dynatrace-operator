package istio

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

var (
	log = logger.Get().WithName("dynakube-istio")
)

const (
	OperatorComponent   = "operator"
	OneAgentComponent   = "oneagent"
	ActiveGateComponent = "activegate"
	IstioGVRName        = "networking.istio.io"
	IstioGVRVersion     = "v1beta1"
)

var (
	IstioGVR = fmt.Sprintf("%s/%s", IstioGVRName, IstioGVRVersion)
)
