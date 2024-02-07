package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
)

const (
	InternalProxySecretVolumeName = "internal-proxy-secret-volume"
)

var (
	log = logger.Get().WithName("activegate-statefulset")
)
