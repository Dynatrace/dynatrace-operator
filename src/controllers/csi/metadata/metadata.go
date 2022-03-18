package metadata

import "time"

// Stores the necessary info from the Dynakube that is needed to be used during volume mount/unmount.
type Dynakube struct {
	Name          string
	TenantUUID    string
	LatestVersion string
}

// NewDynakube returns a new metadata.Dynakube if all fields are set.
func NewDynakube(dynakubeName, tenantUUID, latestVersion string) *Dynakube {
	if tenantUUID == "" || dynakubeName == "" {
		return nil
	}
	return &Dynakube{dynakubeName, tenantUUID, latestVersion}
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

type OsAgentVolume struct {
	VolumeID     string
	TenantUUID   string
	Mounted      bool
	LastModified *time.Time
}

// NewOsAgentVolume returns a new volume if all fields are set.
func NewOsAgentVolume(volumeID, tenantUUID string, mounted bool, timeStamp *time.Time) *OsAgentVolume {
	if volumeID == "" || tenantUUID == "" || timeStamp == nil {
		return nil
	}
	return &OsAgentVolume{volumeID, tenantUUID, mounted, timeStamp}
}

type Access interface {
	Setup(path string) error

	InsertDynakube(dynakube *Dynakube) error
	UpdateDynakube(dynakube *Dynakube) error
	DeleteDynakube(dynakubeName string) error
	GetDynakube(dynakubeName string) (*Dynakube, error)
	GetDynakubes() (map[string]string, error)

	InsertOsAgentVolume(volume *OsAgentVolume) error
	GetOsAgentVolumeViaVolumeID(volumeID string) (*OsAgentVolume, error)
	GetOsAgentVolumeViaTenantUUID(volumeID string) (*OsAgentVolume, error)
	UpdateOsAgentVolume(volume *OsAgentVolume) error

	InsertVolume(volume *Volume) error
	DeleteVolume(volumeID string) error
	GetVolume(volumeID string) (*Volume, error)
	GetPodNames() (map[string]string, error)
	GetUsedVersions(tenantUUID string) (map[string]bool, error)
}
