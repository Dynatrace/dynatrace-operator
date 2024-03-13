package init

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var log = logd.Get().WithName("csi-init")
