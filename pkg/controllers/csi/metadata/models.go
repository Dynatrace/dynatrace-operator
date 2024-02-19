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

type TenantConfig struct {
	UID                        string `gorm:"primaryKey" json:"UID,omitempty"`
	Name                       string `gorm:"not null"`
	TenantUUID                 string `gorm:"not null"`
	DowloadedCodeModuleVersion string
	ConfigDirPath              string `gorm:"not null"`
	MaxFailedMountAttempts     int64  `gorm:"default:10"`
	TimeStampedModel
}

// CodeModule holds what codemodules we have downloaded and available
type CodeModule struct {
	Version  string `gorm:"primaryKey" json:"version,omitempty"`
	Location string `gorm:"not null"`
	TimeStampedModel
}

// OSMount keeps track of our mounts to OS oneAgents, can be "remounted", which causes annoyances
type OSMount struct {
	TenantUUID    string `gorm:"primaryKey" json:"tenantUUID,omitempty"`
	VolumeID      string
	Location      string `gorm:"not null"`
	MountAttempts int64  `gorm:"not null"`
	TimeStampedModel
}

// AppMount keeps track of our mounts to user applications, where we provide the codemodules
type AppMount struct {
	VolumeID          string `gorm:"primaryKey" json:"volumeID,omitempty"`
	CodeModuleVersion string
	Location          string `gorm:"not null"`
	MountAttempts     int64  `gorm:"not null"`
	TimeStampedModel
}

type Volumes struct {
	ID                string `gorm:"primaryKey" json:"id,omitempty"`
	PodUid            string `gorm:"not null"`
	PodName           string `gorm:"not null"`
	PodNamespace      string `gorm:"not null"`
	PodServiceAccount string `gorm:"not null"`
	TimeStampedModel
}
