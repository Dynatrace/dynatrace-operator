package standalone

import "github.com/Dynatrace/dynatrace-operator/src/logger"

var (
	log = logger.NewDTLogger().WithName("standalone-init")
)
