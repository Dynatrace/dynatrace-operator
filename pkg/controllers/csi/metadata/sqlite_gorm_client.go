package metadata

import (
	"context"
	"strings"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GormAccess interface {
	SchemaMigration() error

	ReadTenantConfig(tenantConfig TenantConfig) (*TenantConfig, error)
	ReadCodeModule(codeModule CodeModule) (*CodeModule, error)
	ReadOSMount(osMount OSMount) (*OSMount, error)
	ReadAppMount(appMount AppMount) (*AppMount, error)

	ReadTenantConfigs() ([]TenantConfig, error)
	ReadCodeModules() ([]CodeModule, error)
	ReadOSMounts() ([]OSMount, error)
	ReadAppMounts() ([]AppMount, error)
	ReadVolumeMetas() ([]VolumeMeta, error)

	CreateTenantConfig(tenantConfig *TenantConfig) error
	CreateCodeModule(codeModule *CodeModule) error
	CreateOSMount(osMount *OSMount) error
	CreateAppMount(appMount *AppMount) error

	UpdateTenantConfig(tenantConfig *TenantConfig) error
	UpdateOSMount(osMount *OSMount) error
	UpdateAppMount(appMount *AppMount) error

	DeleteTenantConfig(tenantConfig *TenantConfig, cascade bool) error
	DeleteCodeModule(codeModule *CodeModule) error
	DeleteOSMount(osMount *OSMount) error
	DeleteAppMount(appMount *AppMount) error

	IsCodeModuleOrphaned(codeModule *CodeModule) (bool, error)
}

type GormConn struct {
	ctx context.Context
	db  *gorm.DB
}

var _ GormAccess = &GormConn{}

// NewAccess creates a new gorm db connection to the database.
func NewAccess(ctx context.Context, path string) (*GormConn, error) {
	// we need to explicitly enable foreign_keys for sqlite to have sqlite enforce this constraint
	if strings.Contains(path, "?") {
		path += "&_foreign_keys=on"
	} else {
		path += "?_foreign_keys=on"
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default})

	if err != nil {
		return &GormConn{}, err
	}

	return &GormConn{ctx: ctx, db: db}, nil
}

// SchemaMigration runs gormigrate migrations to create tables
func (conn *GormConn) SchemaMigration() error {
	err := conn.InitGormSchema()
	if err != nil {
		return err
	}

	return gormigrate.New(conn.db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID:      "202403041200",
			Migrate: dataMigration,
			Rollback: func(tx *gorm.DB) error {
				return nil
			},
		},
	}).Migrate()
}

func (conn *GormConn) InitGormSchema() error {
	m := gormigrate.New(conn.db, gormigrate.DefaultOptions, []*gormigrate.Migration{})
	m.InitSchema(func(tx *gorm.DB) error {
		err := tx.AutoMigrate(
			&TenantConfig{},
			&CodeModule{},
			&OSMount{},
			&AppMount{},
			&VolumeMeta{},
		)
		if err != nil {
			return err
		}
		// all other constraints, indexes, etc...
		return nil
	})

	_ = m.Migrate()

	return nil
}

func (conn *GormConn) ReadTenantConfig(tenantConfig TenantConfig) (*TenantConfig, error) {
	var record *TenantConfig

	if (tenantConfig == TenantConfig{}) {
		return nil, errors.New("Can't query for empty TenantConfig")
	}

	result := conn.db.WithContext(conn.ctx).Find(&record, tenantConfig)
	if result.Error != nil {
		return nil, result.Error
	}

	if (*record == TenantConfig{}) {
		return nil, gorm.ErrRecordNotFound
	}

	return record, nil
}

func (conn *GormConn) ReadCodeModule(codeModule CodeModule) (*CodeModule, error) {
	var record *CodeModule

	if (codeModule == CodeModule{}) {
		return nil, errors.New("Can't query for empty CodeModule")
	}

	result := conn.db.WithContext(conn.ctx).Find(&record, codeModule)
	if result.Error != nil {
		return nil, result.Error
	}

	if (*record == CodeModule{}) {
		return nil, gorm.ErrRecordNotFound
	}

	return record, nil
}

func (conn *GormConn) ReadOSMount(osMount OSMount) (*OSMount, error) {
	var record *OSMount

	if (osMount == OSMount{}) {
		return nil, errors.New("Can't query for empty OSMount")
	}

	result := conn.db.WithContext(conn.ctx).Preload("VolumeMeta").Find(&record, osMount)
	if result.Error != nil {
		return nil, result.Error
	}

	if (*record == OSMount{}) {
		return nil, gorm.ErrRecordNotFound
	}

	return record, nil
}

func (conn *GormConn) ReadAppMount(appMount AppMount) (*AppMount, error) {
	var record *AppMount

	if (appMount == AppMount{}) {
		return nil, errors.New("Can't query for empty AppMount")
	}

	result := conn.db.WithContext(conn.ctx).Preload("VolumeMeta").Preload("CodeModule").Find(&record, appMount)
	if result.Error != nil {
		return nil, result.Error
	}

	if (*record == AppMount{}) {
		return nil, gorm.ErrRecordNotFound
	}

	return record, nil
}

func (conn *GormConn) ReadTenantConfigs() ([]TenantConfig, error) {
	var tenantConfigs []TenantConfig

	result := conn.db.WithContext(conn.ctx).Find(&tenantConfigs)
	if result.Error != nil {
		return nil, result.Error
	}

	return tenantConfigs, nil
}

func (conn *GormConn) ReadCodeModules() ([]CodeModule, error) {
	var codeModules []CodeModule

	result := conn.db.WithContext(conn.ctx).Find(&codeModules)
	if result.Error != nil {
		return nil, result.Error
	}

	return codeModules, nil
}

func (conn *GormConn) ReadOSMounts() ([]OSMount, error) {
	var osMounts []OSMount

	result := conn.db.WithContext(conn.ctx).Preload("VolumeMeta").Find(&osMounts)
	if result.Error != nil {
		return nil, result.Error
	}

	return osMounts, nil
}

func (conn *GormConn) ReadAppMounts() ([]AppMount, error) {
	var appMounts []AppMount

	result := conn.db.WithContext(conn.ctx).Preload("VolumeMeta").Preload("CodeModule").Find(&appMounts)
	if result.Error != nil {
		return nil, result.Error
	}

	return appMounts, nil
}

func (conn *GormConn) ReadVolumeMetas() ([]VolumeMeta, error) {
	var volumeMetas []VolumeMeta

	result := conn.db.WithContext(conn.ctx).Find(&volumeMetas)
	if result.Error != nil {
		return nil, result.Error
	}

	return volumeMetas, nil
}

func (conn *GormConn) CreateTenantConfig(tenantConfig *TenantConfig) error {
	return conn.db.WithContext(conn.ctx).Create(tenantConfig).Error
}

func (conn *GormConn) CreateCodeModule(codeModule *CodeModule) error {
	return conn.db.WithContext(conn.ctx).Create(codeModule).Error
}

func (conn *GormConn) CreateOSMount(osMount *OSMount) error {
	return conn.db.WithContext(conn.ctx).Create(osMount).Error
}
func (conn *GormConn) CreateAppMount(appMount *AppMount) error {
	return conn.db.WithContext(conn.ctx).Create(appMount).Error
}

func (conn *GormConn) UpdateTenantConfig(tenantConfig *TenantConfig) error {
	if (tenantConfig == nil || *tenantConfig == TenantConfig{}) {
		return errors.New("Can't save an empty TenantConfig")
	}

	return conn.db.WithContext(conn.ctx).Save(tenantConfig).Error
}

func (conn *GormConn) UpdateOSMount(osMount *OSMount) error {
	if (osMount == nil || *osMount == OSMount{}) {
		return errors.New("Can't save an empty TenantConfig")
	}

	return conn.db.WithContext(conn.ctx).Updates(osMount).Error
}
func (conn *GormConn) UpdateAppMount(appMount *AppMount) error {
	if (appMount == nil || *appMount == AppMount{}) {
		return errors.New("Can't save an empty AppMount")
	}

	return conn.db.WithContext(conn.ctx).Updates(appMount).Error
}

func (conn *GormConn) DeleteTenantConfig(tenantConfig *TenantConfig, cascade bool) error {
	if (tenantConfig == nil || *tenantConfig == TenantConfig{}) {
		return nil
	}

	tenantConfig, err := conn.ReadTenantConfig(*tenantConfig)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	err = conn.db.WithContext(conn.ctx).Delete(&TenantConfig{}, &tenantConfig).Error
	if err != nil {
		return err
	}

	if cascade {
		orphaned, err := conn.IsCodeModuleOrphaned(&CodeModule{Version: tenantConfig.DownloadedCodeModuleVersion})
		if err != nil {
			return err
		}

		if orphaned {
			err = conn.DeleteCodeModule(&CodeModule{Version: tenantConfig.DownloadedCodeModuleVersion})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (conn *GormConn) DeleteCodeModule(codeModule *CodeModule) error {
	if (codeModule == nil || *codeModule == CodeModule{}) {
		return errors.New("Can't delete an empty CodeModule")
	}

	return conn.db.WithContext(conn.ctx).Delete(&CodeModule{}, codeModule).Error
}

func (conn *GormConn) DeleteOSMount(osMount *OSMount) error {
	if (osMount == nil || *osMount == OSMount{}) {
		return errors.New("Can't delete an empty OSMount")
	}

	return conn.db.WithContext(conn.ctx).Delete(&OSMount{}, osMount).Error
}

func (conn *GormConn) DeleteAppMount(appMount *AppMount) error {
	if (appMount == nil || *appMount == AppMount{}) {
		return errors.New("Can't delete an empty AppMount")
	}

	return conn.db.WithContext(conn.ctx).Delete(&AppMount{}, appMount).Error
}

func (conn *GormConn) IsCodeModuleOrphaned(codeModule *CodeModule) (bool, error) {
	var tenantConfigResults []TenantConfig

	var appMountResults []AppMount

	if (codeModule == nil || *codeModule == CodeModule{}) {
		return false, nil
	}

	err := conn.db.WithContext(conn.ctx).Find(&tenantConfigResults, &TenantConfig{DownloadedCodeModuleVersion: codeModule.Version}).Error
	if err != nil {
		return false, err
	}

	err = conn.db.WithContext(conn.ctx).Find(&appMountResults, &AppMount{CodeModuleVersion: codeModule.Version}).Error
	if err != nil {
		return false, err
	}

	if len(tenantConfigResults) == 0 && len(appMountResults) == 0 {
		return true, nil
	}

	return false, nil
}

type AccessOverview struct {
	VolumeMetas   []VolumeMeta   `json:"volumeMetas"`
	AppMounts     []AppMount     `json:"appMounts"`
	TenantConfigs []TenantConfig `json:"tenantConfigs"`
	CodeModules   []CodeModule   `json:"codeModules"`
	OSMounts      []OSMount      `json:"osMounts"`
}

func NewAccessOverview(access GormAccess) (*AccessOverview, error) {
	volumeMetas, err := access.ReadVolumeMetas()
	if err != nil {
		return nil, err
	}

	appMounts, err := access.ReadAppMounts()
	if err != nil {
		return nil, err
	}

	tenantConfigs, err := access.ReadTenantConfigs()
	if err != nil {
		return nil, err
	}

	codeModules, err := access.ReadCodeModules()
	if err != nil {
		return nil, err
	}

	osMounts, err := access.ReadOSMounts()
	if err != nil {
		return nil, err
	}

	return &AccessOverview{
		VolumeMetas:   volumeMetas,
		AppMounts:     appMounts,
		TenantConfigs: tenantConfigs,
		CodeModules:   codeModules,
		OSMounts:      osMounts,
	}, nil
}

func LogAccessOverview(access GormAccess) {
	overview, err := NewAccessOverview(access)
	if err != nil {
		log.Error(err, "Failed to get an overview of the stored csi metadata")
	}

	log.Info("The current overview of the csi metadata", "overview", overview)
}
