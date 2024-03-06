package metadata

import (
	"time"

	"gorm.io/gorm"
)

type TimeStampedModel struct {
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}

// TenantConfig holds info about a given configuration for a tenant.
type TenantConfig struct {
	TimeStampedModel
	UID                         string `gorm:"primaryKey" json:"UID,omitempty"`
	Name                        string `gorm:"not null"`
	DownloadedCodeModuleVersion string
	ConfigDirPath               string  `gorm:"not null"`
	TenantUUID                  string  `gorm:"not null"`
	OSMount                     OSMount `gorm:"foreignKey:TenantUUID;references:TenantUUID"`
	MaxFailedMountAttempts      int64   `gorm:"default:10"`
}

// CodeModule holds what codemodules we have downloaded and available.
type CodeModule struct {
	Version  string `gorm:"primaryKey"`
	Location string `gorm:"not null"`
	TimeStampedModel
}

// OSMount keeps track of our mounts to OS oneAgents, can be "remounted", which causes annoyances.
type OSMount struct {
	VolumeMeta VolumeMeta
	TimeStampedModel
	TenantUUID    string `gorm:"primaryKey" json:"tenantUUID,omitempty"`
	VolumeMetaID  string `gorm:"not null"`
	Location      string `gorm:"not null"`
	MountAttempts int64  `gorm:"not null"`
}

// AppMount keeps track of our mounts to user applications, where we provide the codemodules.
type AppMount struct {
	VolumeMeta VolumeMeta
	CodeModule CodeModule `gorm:"foreignKey:CodeModuleVersion"`
	TimeStampedModel
	VolumeMetaID      string `gorm:"primaryKey"`
	CodeModuleVersion string
	Location          string `gorm:"not null"`
	MountAttempts     int64  `gorm:"not null"`
}

// VolumeMeta keeps metadata we get from kubernetes about the volume.
type VolumeMeta struct {
	ID                string `gorm:"primaryKey"`
	PodUid            string `gorm:"not null"`
	PodName           string `gorm:"not null"`
	PodNamespace      string `gorm:"not null"`
	PodServiceAccount string `gorm:"not null"`
	TimeStampedModel
}
