package nodes

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log                 = logd.Get().WithName("nodes")
	unschedulableTaints = []string{"ToBeDeletedByClusterAutoscaler"}
)
