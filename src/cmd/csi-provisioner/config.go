package csi_server

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

var (
	_ = logger.NewDTLogger().WithName("csi-launcher")
)
