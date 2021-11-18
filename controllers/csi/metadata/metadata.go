package metadata

// Stores the necessary info from the Dynakube that is needed to be used during volume mount/unmount.
type Dynakube struct {
	Name          string
	TenantUUID    string
	LatestVersion string
}

// NewDynakube returns a new metadata.Dynakube if all fields are set.
func NewDynakube(dynakubeName, tenantUUID, latestVersion string) *Dynakube {
	if tenantUUID == "" || latestVersion == "" || dynakubeName == "" {
		return nil
	}
	return &Dynakube{dynakubeName, tenantUUID, latestVersion}
}

type RuxitRevision struct {
	TenantUUID     string
	LatestRevision uint
}

func NewRuxitRevision(tenantUUID string, revision uint) *RuxitRevision {
	if tenantUUID == "" {
		return nil
	}
	return &RuxitRevision{tenantUUID, revision}
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

	InsertDynakube(dynakube *Dynakube) error
	UpdateDynakube(dynakube *Dynakube) error
	DeleteDynakube(dynakubeName string) error
	GetDynakube(dynakubeName string) (*Dynakube, error)
	GetDynakubes() (map[string]string, error)

	InsertRuxitRevision(ruxitRev *RuxitRevision) error
	UpdateRuxitRevision(ruxitRev *RuxitRevision) error
	GetRuxitRevision(tenantUUID string) (*RuxitRevision, error)

	InsertVolume(volume *Volume) error
	DeleteVolume(volumeID string) error
	GetVolume(volumeID string) (*Volume, error)
	GetPodNames() (map[string]string, error)
	GetUsedVersions(tenantUUID string) (map[string]bool, error)
}
