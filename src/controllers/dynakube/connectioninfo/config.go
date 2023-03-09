package connectioninfo

import (
	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	TenantTokenName            = "tenant-token"
	CommunicationEndpointsName = "communication-endpoints"
	CommunicationHosts         = "communication-hosts"
	TenantUUIDName             = "tenant-uuid"

	TokenBasePath         = "/var/lib/dynatrace/secrets/tokens"
	TenantTokenMountPoint = TokenBasePath + "/tenant-token"

	TenantSecretVolumeName = "connection-info-secret"

	EnvDtServer = "DT_SERVER"
	EnvDtTenant = "DT_TENANT"
)

var (
	log = logger.Factory.GetLogger("dynakube-connectioninfo")
)
