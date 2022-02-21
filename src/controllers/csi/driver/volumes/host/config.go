package hostvolumes

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const Mode = "host"

var log = logger.NewDTLogger().WithName("csi-driver.host")
