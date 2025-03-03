package v1

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	defaultUser   int64 = 1001
	defaultGroup  int64 = 1001
	rootUserGroup int64 = 0

	// AnnotationFailurePolicy can be set on a Pod to control what the init container does on failures. When set to
	// "fail", the init container will exit with error code 1. Defaults to "silent".
	AnnotationFailurePolicy = "oneagent.dynatrace.com/failure-policy"
)

var (
	log = logd.Get().WithName("v1-pod-mutation")
)
