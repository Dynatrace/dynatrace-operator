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

// Return a new Tenant if all fields are set.
func NewTenant(uuid, latestVersion, dynakube string) *Tenant {
	if uuid == "" || latestVersion == "" || dynakube == "" {
		return nil
	}
	return &Tenant{uuid, latestVersion, dynakube}
}

type Volume struct {
	ID         string
	PodUID     string
	Version    string
	TenantUUID string
}

// Return a new Volume if all fields are set.
func NewVolume(id, podUID, version, tenantUUID string) *Volume {
	if id == "" || podUID == "" || version == "" || tenantUUID == "" {
		return nil
	}
	return &Volume{id, podUID, version, tenantUUID}
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
