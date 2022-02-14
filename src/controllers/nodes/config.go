package nodes

import (
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	cacheName                   = "dynatrace-node-cache"
	cacheLifetime               = 10 * time.Minute
	lastUpdatedCacheAnnotiation = "DTOperatorLastUpdated"
)

var (
	log                 = logger.NewDTLogger().WithName("nodes-controller")
	unschedulableTaints = []string{"ToBeDeletedByClusterAutoscaler"}
)
