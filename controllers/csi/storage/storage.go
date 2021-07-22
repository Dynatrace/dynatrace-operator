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

type Pod struct {
	UID        string
	VolumeID   string
	Version    string
	TenantUUID string
}

type Access interface {
	InsertTenant(tenant *Tenant) error
	UpdateTenant(tenant *Tenant) error
	GetTenant(uuid string) (*Tenant, error)
	GetTenantViaDynakube(dynakube string) (*Tenant, error)

	InsertPodInfo(pod *Pod) error
	DeletePodInfo(pod *Pod) error
	GetPodViaVolumeId(volumeID string) (*Pod, error)
	GetUsedVersions(tenantUUID string) (map[string]bool, error)
}

var dbPath = filepath.Join(dtcsi.DataPath, "csi.db")
