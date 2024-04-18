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
	SchemaMigration(ctx context.Context) error

	CreateTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error
	UpdateTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error
	DeleteTenantConfig(ctx context.Context, tenantConfig *TenantConfig, cascade bool) error
	ReadTenantConfig(ctx context.Context, tenantConfig TenantConfig) (*TenantConfig, error)
	ReadTenantConfigs(ctx context.Context) ([]TenantConfig, error)

	CreateCodeModule(ctx context.Context, codeModule *CodeModule) error
	DeleteCodeModule(ctx context.Context, codeModule *CodeModule) error
	ReadCodeModule(ctx context.Context, codeModule CodeModule) (*CodeModule, error)
	ReadCodeModules(ctx context.Context) ([]CodeModule, error)
	IsCodeModuleOrphaned(ctx context.Context, codeModule *CodeModule) (bool, error)

	CreateOSMount(ctx context.Context, osMount *OSMount) error
	UpdateOSMount(ctx context.Context, osMount *OSMount) error
	DeleteOSMount(ctx context.Context, osMount *OSMount) error
	ReadOSMount(ctx context.Context, osMount OSMount) (*OSMount, error)
	ReadOSMounts(ctx context.Context) ([]OSMount, error)

	CreateAppMount(ctx context.Context, appMount *AppMount) error
	UpdateAppMount(ctx context.Context, appMount *AppMount) error
	DeleteAppMount(ctx context.Context, appMount *AppMount) error
	ReadAppMount(ctx context.Context, appMount AppMount) (*AppMount, error)
	ReadAppMounts(ctx context.Context) ([]AppMount, error)

	ReadVolumeMetas(ctx context.Context) ([]VolumeMeta, error)
}

type GormConn struct {
	db *gorm.DB
}

var _ GormAccess = &GormConn{}

// NewAccess creates a new gorm db connection to the database.
func NewAccess(path string) (*GormConn, error) {
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

	return &GormConn{db: db}, nil
}

// SchemaMigration runs gormigrate migrations to create tables
func (conn *GormConn) SchemaMigration(_ context.Context) error {
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

func (conn *GormConn) CreateTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error {
	return conn.db.WithContext(ctx).Create(tenantConfig).Error
}

func (conn *GormConn) UpdateTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error {
	if (tenantConfig == nil || *tenantConfig == TenantConfig{}) {
		return errors.New("Can't save an empty TenantConfig")
	}

	return conn.db.WithContext(ctx).Save(tenantConfig).Error
}

func (conn *GormConn) DeleteTenantConfig(ctx context.Context, tenantConfig *TenantConfig, cascade bool) error {
	if (tenantConfig == nil || *tenantConfig == TenantConfig{}) {
		return nil
	}

	tenantConfig, err := conn.ReadTenantConfig(ctx, *tenantConfig)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	err = conn.db.WithContext(ctx).Delete(&tenantConfig).Error
	if err != nil {
		return err
	}

	if cascade {
		orphaned, err := conn.IsCodeModuleOrphaned(ctx, &CodeModule{Version: tenantConfig.DownloadedCodeModuleVersion})
		if err != nil {
			return err
		}

		if orphaned {
			err = conn.DeleteCodeModule(ctx, &CodeModule{Version: tenantConfig.DownloadedCodeModuleVersion})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (conn *GormConn) ReadTenantConfig(ctx context.Context, tenantConfig TenantConfig) (*TenantConfig, error) {
	var record *TenantConfig

	if (tenantConfig == TenantConfig{}) {
		return nil, errors.New("Can't query for empty TenantConfig")
	}

	result := conn.db.WithContext(ctx).Find(&record, tenantConfig)
	if result.Error != nil {
		return nil, result.Error
	}

	if (*record == TenantConfig{}) {
		return nil, gorm.ErrRecordNotFound
	}

	return record, nil
}

func (conn *GormConn) ReadTenantConfigs(ctx context.Context) ([]TenantConfig, error) {
	var tenantConfigs []TenantConfig

	result := conn.db.WithContext(ctx).Find(&tenantConfigs)
	if result.Error != nil {
		return nil, result.Error
	}

	return tenantConfigs, nil
}

func (conn *GormConn) CreateCodeModule(ctx context.Context, codeModule *CodeModule) error {
	return conn.db.WithContext(ctx).Create(codeModule).Error
}

func (conn *GormConn) DeleteCodeModule(ctx context.Context, codeModule *CodeModule) error {
	if (codeModule == nil || *codeModule == CodeModule{}) {
		return errors.New("Can't delete an empty CodeModule")
	}

	return conn.db.WithContext(ctx).Delete(&CodeModule{}, codeModule).Error
}

func (conn *GormConn) ReadCodeModule(ctx context.Context, codeModule CodeModule) (*CodeModule, error) {
	var record *CodeModule

	if (codeModule == CodeModule{}) {
		return nil, errors.New("Can't query for empty CodeModule")
	}

	err := conn.db.WithContext(ctx).Find(&record, codeModule).Error
	if err != nil {
		return nil, err
	}

	if (*record == CodeModule{}) {
		return nil, gorm.ErrRecordNotFound
	}

	return record, nil
}

func (conn *GormConn) ReadCodeModules(ctx context.Context) ([]CodeModule, error) {
	var codeModules []CodeModule

	result := conn.db.WithContext(ctx).Find(&codeModules)
	if result.Error != nil {
		return nil, result.Error
	}

	return codeModules, nil
}

func (conn *GormConn) IsCodeModuleOrphaned(ctx context.Context, codeModule *CodeModule) (bool, error) {
	var tenantConfigResults []TenantConfig

	var appMountResults []AppMount

	if (codeModule == nil || *codeModule == CodeModule{}) {
		return false, nil
	}

	err := conn.db.WithContext(ctx).Find(&tenantConfigResults, &TenantConfig{DownloadedCodeModuleVersion: codeModule.Version}).Error
	if err != nil {
		return false, err
	}

	err = conn.db.WithContext(ctx).Find(&appMountResults, &AppMount{CodeModuleVersion: codeModule.Version}).Error
	if err != nil {
		return false, err
	}

	if len(tenantConfigResults) == 0 && len(appMountResults) == 0 {
		return true, nil
	}

	return false, nil
}

func (conn *GormConn) CreateOSMount(ctx context.Context, osMount *OSMount) error {
	return conn.db.WithContext(ctx).Create(osMount).Error
}

func (conn *GormConn) UpdateOSMount(ctx context.Context, osMount *OSMount) error {
	if (osMount == nil || *osMount == OSMount{}) {
		return errors.New("Can't save an empty TenantConfig")
	}

	err := conn.restoreOSMount(ctx, osMount)
	if err != nil {
		return err
	}

	return conn.db.WithContext(ctx).Updates(osMount).Error
}

func (conn *GormConn) restoreOSMount(ctx context.Context, osMount *OSMount) error {
	result := conn.db.WithContext(ctx).Preload("VolumeMeta").Unscoped().Find(&osMount, osMount)
	if result.Error != nil {
		return result.Error
	}

	if osMount == nil {
		return errors.New("Cannot restore nil OSMount")
	}

	osMount.DeletedAt.Valid = false

	return conn.db.WithContext(ctx).Unscoped().Updates(osMount).Error
}

func (conn *GormConn) ReadOSMount(ctx context.Context, osMount OSMount) (*OSMount, error) {
	var record *OSMount

	if (osMount == OSMount{}) {
		return nil, errors.New("Can't query for empty OSMount")
	}

	result := conn.db.WithContext(ctx).Preload("VolumeMeta").Find(&record, osMount)
	if result.Error != nil {
		return nil, result.Error
	}

	if (*record == OSMount{}) {
		return nil, gorm.ErrRecordNotFound
	}

	return record, nil
}

func (conn *GormConn) ReadOSMounts(ctx context.Context) ([]OSMount, error) {
	var osMounts []OSMount

	result := conn.db.WithContext(ctx).Preload("VolumeMeta").Find(&osMounts)
	if result.Error != nil {
		return nil, result.Error
	}

	return osMounts, nil
}

func (conn *GormConn) DeleteOSMount(ctx context.Context, osMount *OSMount) error {
	if (osMount == nil || *osMount == OSMount{}) {
		return errors.New("Can't delete an empty OSMount")
	}

	return conn.db.WithContext(ctx).Delete(&OSMount{}, osMount).Error
}

func (conn *GormConn) CreateAppMount(ctx context.Context, appMount *AppMount) error {
	result := conn.db.WithContext(ctx).Create(appMount)

	return result.Error
}

func (conn *GormConn) UpdateAppMount(ctx context.Context, appMount *AppMount) error {
	if (appMount == nil || *appMount == AppMount{}) {
		return errors.New("Can't save an empty AppMount")
	}

	return conn.db.WithContext(ctx).Updates(appMount).Error
}

func (conn *GormConn) ReadAppMount(ctx context.Context, appMount AppMount) (*AppMount, error) {
	var record *AppMount

	if (appMount == AppMount{}) {
		return nil, errors.New("Can't query for empty AppMount")
	}

	result := conn.db.WithContext(ctx).Preload("VolumeMeta").Preload("CodeModule").Find(&record, appMount)
	if result.Error != nil {
		return nil, result.Error
	}

	if (*record == AppMount{}) {
		return nil, gorm.ErrRecordNotFound
	}

	return record, nil
}

func (conn *GormConn) ReadAppMounts(ctx context.Context) ([]AppMount, error) {
	var appMounts []AppMount

	result := conn.db.WithContext(ctx).Preload("VolumeMeta").Find(&appMounts)
	if result.Error != nil {
		return nil, result.Error
	}

	return appMounts, nil
}

func (conn *GormConn) DeleteAppMount(ctx context.Context, appMount *AppMount) error {
	if (appMount == nil || *appMount == AppMount{}) {
		return errors.New("Can't delete an empty AppMount")
	}

	return conn.db.WithContext(ctx).Delete(&AppMount{}, appMount).Error
}

func (conn *GormConn) ReadVolumeMetas(ctx context.Context) ([]VolumeMeta, error) {
	var volumeMetas []VolumeMeta

	result := conn.db.WithContext(ctx).Find(&volumeMetas)
	if result.Error != nil {
		return nil, result.Error
	}

	return volumeMetas, nil
}

type AccessOverview struct {
	VolumeMetas   []VolumeMeta   `json:"volumeMetas"`
	AppMounts     []AppMount     `json:"appMounts"`
	TenantConfigs []TenantConfig `json:"tenantConfigs"`
	CodeModules   []CodeModule   `json:"codeModules"`
	OSMounts      []OSMount      `json:"osMounts"`
}

func NewAccessOverview(access GormAccess) (*AccessOverview, error) {
	ctx := context.Background()

	volumeMetas, err := access.ReadVolumeMetas(ctx)
	if err != nil {
		return nil, err
	}

	appMounts, err := access.ReadAppMounts(ctx)
	if err != nil {
		return nil, err
	}

	tenantConfigs, err := access.ReadTenantConfigs(ctx)
	if err != nil {
		return nil, err
	}

	codeModules, err := access.ReadCodeModules(ctx)
	if err != nil {
		return nil, err
	}

	osMounts, err := access.ReadOSMounts(ctx)
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
