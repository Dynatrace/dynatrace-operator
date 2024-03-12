package metadata

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	EmptyStringFieldError = `Invalid field value: "" for field %s`
)

type TimeStampedModel struct {
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}

// TenantConfig holds info about a given configuration for a tenant.
type TenantConfig struct {
	TimeStampedModel
	UID                         string `gorm:"primaryKey"`
	Name                        string `gorm:"not null"`
	DownloadedCodeModuleVersion string
	ConfigDirPath               string `gorm:"not null"`
	TenantUUID                  string `gorm:"not null"`
	MaxFailedMountAttempts      int64  `gorm:"default:10"`
}

func (tenantConfig *TenantConfig) BeforeCreate(_ *gorm.DB) error {
	tenantConfig.UID = uuid.NewString()

	if err := tenantConfig.IsValid(); err != nil {
		return err
	}

	return nil
}

func (tenantConfig *TenantConfig) IsValid() error {
	if tenantConfig.UID == "" {
		return errors.Errorf(EmptyStringFieldError, `TenantConfig.UID`)
	}

	if tenantConfig.Name == "" {
		return errors.Errorf(EmptyStringFieldError, `TenantConfig.Name`)
	}

	if tenantConfig.ConfigDirPath == "" {
		return errors.Errorf(EmptyStringFieldError, `TenantConfig.ConfigDirPath`)
	}

	if tenantConfig.TenantUUID == "" {
		return errors.Errorf(EmptyStringFieldError, `TenantConfig.TenantUUID`)
	}

	return nil
}

// CodeModule holds what codemodules we have downloaded and available.
type CodeModule struct {
	Version  string `gorm:"primaryKey"`
	Location string `gorm:"not null"`
	TimeStampedModel
}

func (codeModule *CodeModule) BeforeCreate(_ *gorm.DB) error {
	if err := codeModule.IsValid(); err != nil {
		return err
	}

	return nil
}

func (codeModule *CodeModule) IsValid() error {
	if codeModule.Version == "" {
		return errors.Errorf(EmptyStringFieldError, `CodeModule.Version`)
	}

	if codeModule.Location == "" {
		return errors.Errorf(EmptyStringFieldError, `CodeModule.Location`)
	}

	return nil
}

// OSMount keeps track of our mounts to OS oneAgents, can be "remounted", which causes annoyances.
type OSMount struct {
	VolumeMeta VolumeMeta `gorm:"foreignKey:VolumeMetaID"`
	TimeStampedModel
	TenantConfigUID string       `gorm:"not null"`
	TenantUUID      string       `gorm:"primaryKey"`
	VolumeMetaID    string       `gorm:"not null"`
	Location        string       `gorm:"not null"`
	TenantConfig    TenantConfig `gorm:"foreignKey:TenantConfigUID"`
	MountAttempts   int64        `gorm:"not null"`
}

func (osMount *OSMount) BeforeCreate(_ *gorm.DB) error {
	if err := osMount.IsValid(); err != nil {
		return err
	}

	return nil
}

func (osMount *OSMount) IsValid() error {
	if err := osMount.VolumeMeta.IsValid(); err != nil {
		return err
	}

	if err := osMount.TenantConfig.IsValid(); err != nil {
		return err
	}

	if osMount.TenantUUID == "" {
		return errors.Errorf(EmptyStringFieldError, `OSMount.TenantUUID`)
	}

	if osMount.Location == "" {
		return errors.Errorf(EmptyStringFieldError, `OSMount.Location`)
	}

	return nil
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

func (appMount *AppMount) BeforeCreate(_ *gorm.DB) error {
	if err := appMount.IsValid(); err != nil {
		return err
	}

	return nil
}

func (appMount *AppMount) IsValid() error {
	if err := appMount.VolumeMeta.IsValid(); err != nil {
		return err
	}

	if err := appMount.CodeModule.IsValid(); err != nil {
		return err
	}

	if appMount.Location == "" {
		return errors.Errorf(EmptyStringFieldError, `AppMount.Location`)
	}

	return nil
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

func (volumeMeta *VolumeMeta) BeforeCreate(_ *gorm.DB) error {
	if err := volumeMeta.IsValid(); err != nil {
		return err
	}

	return nil
}

func (volumeMeta *VolumeMeta) IsValid() error {
	if volumeMeta.ID == "" {
		return errors.Errorf(EmptyStringFieldError, `VolumeMeta.ID`)
	}

	if volumeMeta.PodUid == "" {
		return errors.Errorf(EmptyStringFieldError, `VolumeMeta.PodUid`)
	}

	if volumeMeta.PodName == "" {
		return errors.Errorf(EmptyStringFieldError, `VolumeMeta.PodName`)
	}

	if volumeMeta.PodNamespace == "" {
		return errors.Errorf(EmptyStringFieldError, `VolumeMeta.PodNamespace`)
	}

	if volumeMeta.PodServiceAccount == "" {
		return errors.Errorf(EmptyStringFieldError, `VolumeMeta.PodServiceAccount`)
	}

	return nil
}
