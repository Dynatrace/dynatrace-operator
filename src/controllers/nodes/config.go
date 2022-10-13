package nodes

import (
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	cacheName                  = "dynatrace-node-cache"
	cacheLifetime              = 10 * time.Minute
	lastUpdatedCacheAnnotation = "DTOperatorLastUpdated"
)

var (
	log                 = logger.Factory.GetLogger("nodes")
	unschedulableTaints = []string{"ToBeDeletedByClusterAutoscaler"}
)
