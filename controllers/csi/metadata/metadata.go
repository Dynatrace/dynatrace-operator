package metadata

type Tenant struct {
	TenantUUID    string
	LatestVersion string
	Dynakube      string
}

// NewTenant returns a new Tenant if all fields are set.
func NewTenant(uuid, latestVersion, dynakube string) *Tenant {
	if uuid == "" || latestVersion == "" || dynakube == "" {
		return nil
	}
	return &Tenant{uuid, latestVersion, dynakube}
}

type Volume struct {
	VolumeID   string
	PodName    string
	Version    string
	TenantUUID string
}

// NewVolume returns a new Volume if all fields are set.
func NewVolume(id, podUID, version, tenantUUID string) *Volume {
	if id == "" || podUID == "" || version == "" || tenantUUID == "" {
		return nil
	}
	return &Volume{id, podUID, version, tenantUUID}
}

type Access interface {
	Setup(path string) error

	InsertTenant(tenant *Tenant) error
	UpdateTenant(tenant *Tenant) error
	DeleteTenant(tenantUUID string) error
	GetTenant(dynakubeName string) (*Tenant, error)
	GetDynakubes() (map[string]string, error)

	InsertVolume(volume *Volume) error
	DeleteVolume(volumeID string) error
	GetVolume(volumeID string) (*Volume, error)
	GetPodNames() (map[string]string, error)
	GetUsedVersions(tenantUUID string) (map[string]bool, error)
}
