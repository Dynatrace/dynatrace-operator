package host

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const Mode = "host"

var log = logd.Get().WithName("csi-hostvolume")
