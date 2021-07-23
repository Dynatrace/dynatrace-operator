package storage

import (
	"path/filepath"

	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
)

type Tenant struct {
	UUID          string
	LatestVersion string
	Dynakube      string
}

type Volume struct {
	ID         string
	PodUID     string
	Version    string
	TenantUUID string
}

type Access interface {
	InsertTenant(tenant *Tenant) error
	UpdateTenant(tenant *Tenant) error
	GetTenant(uuid string) (*Tenant, error)
	GetTenantViaDynakube(dynakube string) (*Tenant, error)

	InsertVolumeInfo(volume *Volume) error
	DeleteVolumeInfo(volumeID string) error
	GetVolumeInfo(volumeID string) (*Volume, error)
	GetUsedVersions(tenantUUID string) (map[string]bool, error)
}

var dbPath = filepath.Join(dtcsi.DataPath, "csi.db")
