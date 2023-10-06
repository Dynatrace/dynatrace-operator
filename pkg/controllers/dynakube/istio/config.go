package istio

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

var (
	log = logger.Factory.GetLogger("dynakube-istio")
)

const (
	OperatorComponent   = "operator"
	OneAgentComponent   = "oneagent"
	ActiveGateComponent = "activegate"
	IstioGVRName        = "networking.istio.io"
	IstioGVRVersion     = "v1alpha3"
)

var (
	IstioGVR = fmt.Sprintf("%s/%s", IstioGVRName, IstioGVRVersion)
)
