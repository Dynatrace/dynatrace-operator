package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/src/util/logger"
)

const (
	InternalProxySecretVolumeName = "internal-proxy-secret-volume"
)

var (
	log = logger.Factory.GetLogger("activegate-statefulset")
)
