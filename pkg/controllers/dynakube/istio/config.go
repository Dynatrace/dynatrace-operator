package istio

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("dynakube-istio")
)

const (
	OperatorComponent   = "operator"
	CodeModuleComponent = "OneAgent"
	ActiveGateComponent = "ActiveGate"
	IstioGVRName        = "networking.istio.io"
	IstioGVRVersion     = "v1beta1"
)

var (
	IstioGVR = fmt.Sprintf("%s/%s", IstioGVRName, IstioGVRVersion)
)
