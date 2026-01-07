package crdcleanup

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var log = logd.Get().WithName("crdcleanup")

const (
	healthProbeBindAddress = ":10080"

	livezEndpointName    = "livez"
	livenessEndpointName = "/" + livezEndpointName
)
