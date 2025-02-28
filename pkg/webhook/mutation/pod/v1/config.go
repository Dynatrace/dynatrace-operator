package pod

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	defaultUser   int64 = 1001
	defaultGroup  int64 = 1001
	rootUserGroup int64 = 0
)

var (
	log = logd.Get().WithName("v1-pod-mutation")
)
