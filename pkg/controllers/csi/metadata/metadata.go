package metadata

import (
	"context"
	"time"
)

// Dynakube stores the necessary info from the Dynakube that is needed to be used during volume mount/unmount.
type Dynakube struct {
	Name                   string `json:"name"`
	TenantUUID             string `json:"tenantUUID"`
	LatestVersion          string `json:"latestVersion"`
	ImageDigest            string `json:"imageDigest"`
	MaxFailedMountAttempts int    `json:"maxFailedMountAttempts"`
}

// NewDynakube returns a new metadata.Dynakube if all fields are set.
func NewDynakube(dynakubeName, tenantUUID, latestVersion, imageDigest string, maxFailedMountAttempts int) *Dynakube {
	if tenantUUID == "" || dynakubeName == "" {
		return nil
	}

	return &Dynakube{
		Name:                   dynakubeName,
		TenantUUID:             tenantUUID,
		LatestVersion:          latestVersion,
		ImageDigest:            imageDigest,
		MaxFailedMountAttempts: maxFailedMountAttempts,
	}
}

type Volume struct {
	VolumeID      string `json:"volumeID" gorm:"column:ID"`
	PodName       string `json:"podName"`
	Version       string `json:"version"`
	TenantUUID    string `json:"tenantUUID"`
	MountAttempts int    `json:"mountAttempts"`
}

// NewVolume returns a new Volume if all fields (except version) are set.
func NewVolume(id, podName, version, tenantUUID string, mountAttempts int) *Volume {
	if id == "" || podName == "" || tenantUUID == "" {
		return nil
	}

	if mountAttempts < 0 {
		mountAttempts = 0
	}

	return &Volume{
		VolumeID:      id,
		PodName:       podName,
		Version:       version,
		TenantUUID:    tenantUUID,
		MountAttempts: mountAttempts,
	}
}

type OsAgentVolume struct {
	LastModified *time.Time `json:"lastModified"`
	VolumeID     string     `json:"volumeID"`
	TenantUUID   string     `json:"tenantUUID"`
	Mounted      bool       `json:"mounted"`
}

// NewOsAgentVolume returns a new volume if all fields are set.
func NewOsAgentVolume(volumeID, tenantUUID string, mounted bool, timeStamp *time.Time) *OsAgentVolume {
	if volumeID == "" || tenantUUID == "" || timeStamp == nil {
		return nil
	}

	return &OsAgentVolume{VolumeID: volumeID, TenantUUID: tenantUUID, Mounted: mounted, LastModified: timeStamp}
}

type Access interface {
	Setup(ctx context.Context, path string) error

	InsertDynakube(ctx context.Context, dynakube *Dynakube) error
	UpdateDynakube(ctx context.Context, dynakube *Dynakube) error
	DeleteDynakube(ctx context.Context, dynakubeName string) error
	GetDynakube(ctx context.Context, dynakubeName string) (*Dynakube, error)
	GetTenantsToDynakubes(ctx context.Context) (map[string]string, error)
	GetAllDynakubes(ctx context.Context) ([]*Dynakube, error)
	GetAllAppMounts(ctx context.Context) []*Volume

	InsertOsAgentVolume(ctx context.Context, volume *OsAgentVolume) error
	GetOsAgentVolumeViaVolumeID(ctx context.Context, volumeID string) (*OsAgentVolume, error)
	GetOsAgentVolumeViaTenantUUID(ctx context.Context, volumeID string) (*OsAgentVolume, error)
	UpdateOsAgentVolume(ctx context.Context, volume *OsAgentVolume) error
	GetAllOsAgentVolumes(ctx context.Context) ([]*OsAgentVolume, error)

	InsertVolume(ctx context.Context, volume *Volume) error
	DeleteVolume(ctx context.Context, volumeID string) error
	GetVolume(ctx context.Context, volumeID string) (*Volume, error)
	GetAllVolumes(ctx context.Context) ([]*Volume, error)
	GetPodNames(ctx context.Context) (map[string]string, error)
	GetUsedVersions(ctx context.Context, tenantUUID string) (map[string]bool, error)
	GetAllUsedVersions(ctx context.Context) (map[string]bool, error)
	GetLatestVersions(ctx context.Context) (map[string]bool, error)
	GetUsedImageDigests(ctx context.Context) (map[string]bool, error)
	IsImageDigestUsed(ctx context.Context, imageDigest string) (bool, error)
}

type AccessOverview struct {
	Volumes        []*Volume        `json:"volumes"`
	Dynakubes      []*Dynakube      `json:"dynakubes"`
	OsAgentVolumes []*OsAgentVolume `json:"osAgentVolumes"`
}

func NewAccessOverview(access Access) (*AccessOverview, error) {
	ctx := context.Background()

	volumes, err := access.GetAllVolumes(ctx)
	if err != nil {
		return nil, err
	}

	dynakubes, err := access.GetAllDynakubes(ctx)
	if err != nil {
		return nil, err
	}

	osVolumes, err := access.GetAllOsAgentVolumes(ctx)
	if err != nil {
		return nil, err
	}

	return &AccessOverview{
		Volumes:        volumes,
		Dynakubes:      dynakubes,
		OsAgentVolumes: osVolumes,
	}, nil
}

func LogAccessOverview(access Access) {
	overview, err := NewAccessOverview(access)
	if err != nil {
		log.Error(err, "Failed to get an overview of the stored csi metadata")
	}

	log.Info("The current overview of the csi metadata", "overview", overview)
}
