package nodes

import (
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
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
