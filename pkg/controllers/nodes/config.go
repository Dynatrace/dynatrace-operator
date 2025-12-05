package nodes

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	Log                 = logd.Get().WithName("nodes")
	unschedulableTaints = []string{"ToBeDeletedByClusterAutoscaler"}
)
