package nodes

import (
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/logd"
)

const (
	cacheName                  = "dynatrace-node-cache"
	cacheLifetime              = 10 * time.Minute
	lastUpdatedCacheAnnotation = "DTOperatorLastUpdated"
)

var (
	log                 = logd.Get().WithName("nodes")
	unschedulableTaints = []string{"ToBeDeletedByClusterAutoscaler"}
)
