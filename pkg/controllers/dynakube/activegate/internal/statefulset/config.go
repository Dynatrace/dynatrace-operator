package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logd"
)

const (
	InternalProxySecretVolumeName = "internal-proxy-secret-volume"
)

var (
	log = logd.Get().WithName("activegate-statefulset")
)
