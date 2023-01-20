package common

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	Log = logger.Factory.GetLogger(DynaMetrics)
)
