package tenantinfo

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	TenantTokenName            = "tenant-token"
	CommunicationEndpointsName = "communication-endpoints"
	TenantUuidName             = "tenant-uuid"
)

var (
	log = logger.NewDTLogger().WithName("dynakube.tenantinfo")
)
