package connectioninfo

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	TenantTokenName            = "tenant-token"
	CommunicationEndpointsName = "communication-endpoints"
	TenantUuidName             = "tenant-uuid"

	TokenBasePath         = "/var/lib/dynatrace/secrets/tokens" //nolint:gosec
	TenantTokenMountPoint = TokenBasePath + "/tenant-token"

	TenantSecretVolumeName = "connection-info-secret"
)

var (
	log = logger.Factory.GetLogger("dynakube-connectioninfo")
)
