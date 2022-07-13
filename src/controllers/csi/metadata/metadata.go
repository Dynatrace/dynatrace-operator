package metadata

import (
	"time"

	"github.com/go-logr/logr"
)

// Stores the necessary info from the Dynakube that is needed to be used during volume mount/unmount.
type Dynakube struct {
	Name          string `json:"name"`
	TenantUUID    string `json:"tenantUUID"`
	LatestVersion string `json:"latestVersion"`
	ImageDigest   string `json:"imageDigest"`
}

// NewDynakube returns a new metadata.Dynakube if all fields are set.
func NewDynakube(dynakubeName, tenantUUID, latestVersion string, imageDigest string) *Dynakube {
	if tenantUUID == "" || dynakubeName == "" {
		return nil
	}
	return &Dynakube{dynakubeName, tenantUUID, latestVersion, imageDigest}
}

type Volume struct {
	VolumeID   string `json:"volumeID"`
	PodName    string `json:"podName"`
	Version    string `json:"version"`
	TenantUUID string `json:"tenantUUID"`
}

// NewVolume returns a new Volume if all fields are set.
func NewVolume(id, podUID, version, tenantUUID string) *Volume {
	if id == "" || podUID == "" || version == "" || tenantUUID == "" {
		return nil
	}
	return &Volume{id, podUID, version, tenantUUID}
}

type OsAgentVolume struct {
	VolumeID     string     `json:"volumeID"`
	TenantUUID   string     `json:"tenantUUID"`
	Mounted      bool       `json:"mounted"`
	LastModified *time.Time `json:"lastModified"`
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
	GetTenantsToDynakubes() (map[string]string, error)
	GetAllDynakubes() ([]*Dynakube, error)

	InsertOsAgentVolume(volume *OsAgentVolume) error
	GetOsAgentVolumeViaVolumeID(volumeID string) (*OsAgentVolume, error)
	GetOsAgentVolumeViaTenantUUID(volumeID string) (*OsAgentVolume, error)
	UpdateOsAgentVolume(volume *OsAgentVolume) error
	GetAllOsAgentVolumes() ([]*OsAgentVolume, error)

	InsertVolume(volume *Volume) error
	DeleteVolume(volumeID string) error
	GetVolume(volumeID string) (*Volume, error)
	GetAllVolumes() ([]*Volume, error)
	GetPodNames() (map[string]string, error)
	GetUsedVersions(tenantUUID string) (map[string]bool, error)
	GetAllUsedVersions() (map[string]bool, error)
	GetUsedImageDigests() (map[string]bool, error)
	GetDynakubeNamesForImageDigest(imageDigest string) ([]string, error)
}

type AccessOverview struct {
	Volumes        []*Volume        `json:"volumes"`
	Dynakubes      []*Dynakube      `json:"dynakubes"`
	OsAgentVolumes []*OsAgentVolume `json:"osAgentVolumes"`
}

func NewAccessOverview(access Access) (*AccessOverview, error) {
	volumes, err := access.GetAllVolumes()
	if err != nil {
		return nil, err
	}
	dynakubes, err := access.GetAllDynakubes()
	if err != nil {
		return nil, err
	}
	osVolumes, err := access.GetAllOsAgentVolumes()
	if err != nil {
		return nil, err
	}
	return &AccessOverview{
		Volumes:        volumes,
		Dynakubes:      dynakubes,
		OsAgentVolumes: osVolumes,
	}, nil
}

func LogAccessOverview(log logr.Logger, access Access) {
	overview, err := NewAccessOverview(access)
	if err != nil {
		log.Error(err, "Failed to get an overview of the stored csi metadata")
	}
	log.Info("The current overview of the csi metadata", "overview", overview)
}
