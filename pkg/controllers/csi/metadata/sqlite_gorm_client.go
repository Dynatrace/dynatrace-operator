package metadata

import (
	"context"
	"strings"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBAccess interface {
	SchemaMigration(ctx context.Context) error

	CreateTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error
	UpdateTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error
	DeleteTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error
	ReadTenantConfigByTenantUUID(ctx context.Context, uid string) (*TenantConfig, error)

	CreateCodeModule(ctx context.Context, codeModule *CodeModule) error
	ReadCodeModuleByVersion(ctx context.Context, version string) (*CodeModule, error)
	DeleteCodeModule(ctx context.Context, codeModule *CodeModule) error

	CreateOSMount(ctx context.Context, osMount *OSMount) error
	UpdateOSMount(ctx context.Context, osMount *OSMount) error
	DeleteOSMount(ctx context.Context, osMount *OSMount) error
	ReadOSMountByTenantUUID(ctx context.Context, tenantUUID string) (*OSMount, error)

	CreateAppMount(ctx context.Context, appMount *AppMount) error
	UpdateAppMount(ctx context.Context, appMount *AppMount) error
	DeleteAppMount(ctx context.Context, appMount *AppMount) error
	ReadAppMountByVolumeMetaID(ctx context.Context, volumeMetaID string) (*AppMount, error)
	ReadAppMounts(ctx context.Context) ([]AppMount, error)
}

type DBConn struct {
	db *gorm.DB
}

var _ DBAccess = (*DBConn)(nil)

// NewDBAccess creates a new gorm db connection to the database.
func NewDBAccess(path string) (DBConn, error) {
	// we should explicitly enable foreign_keys for sqlite
	if strings.Contains(path, "?") {
		path += "&_foreign_keys=on"
	} else {
		path += "?_foreign_keys=on"
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default})

	if err != nil {
		return DBConn{}, err
	}

	return DBConn{db: db}, nil
}

// SchemaMigration runs gormigrate migrations to create tables
func (conn *DBConn) SchemaMigration(_ context.Context) error {
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

func (conn *DBConn) CreateTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error {
	return conn.db.WithContext(ctx).Create(tenantConfig).Error
}

func (conn *DBConn) UpdateTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error {
	return conn.db.WithContext(ctx).Updates(tenantConfig).Error
}

func (conn *DBConn) DeleteTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error {
	return conn.db.WithContext(ctx).Delete(tenantConfig).Error
}

func (conn *DBConn) CreateCodeModule(ctx context.Context, codeModule *CodeModule) error {
	return conn.db.WithContext(ctx).Create(codeModule).Error
}

func (conn *DBConn) ReadCodeModuleByVersion(ctx context.Context, version string) (*CodeModule, error) {
	var codeModule CodeModule

	result := conn.db.WithContext(ctx).First(&codeModule, "version = ?", version)
	if result.Error != nil {
		return nil, result.Error
	}

	return &codeModule, nil
}

func (conn *DBConn) DeleteCodeModule(ctx context.Context, codeModule *CodeModule) error {
	return conn.db.WithContext(ctx).Delete(codeModule).Error
}

func (conn *DBConn) ReadTenantConfigByTenantUUID(ctx context.Context, uid string) (*TenantConfig, error) {
	var tenantConfig TenantConfig

	result := conn.db.WithContext(ctx).First(&tenantConfig, "tenant_uuid = ?", uid)
	if result.Error != nil {
		return nil, result.Error
	}

	return &tenantConfig, nil
}

func (conn *DBConn) CreateOSMount(ctx context.Context, osMount *OSMount) error {
	return conn.db.WithContext(ctx).Create(osMount).Error
}

func (conn *DBConn) UpdateOSMount(ctx context.Context, osMount *OSMount) error {
	return conn.db.WithContext(ctx).Updates(osMount).Error
}

func (conn *DBConn) ReadOSMountByTenantUUID(ctx context.Context, tenantUUID string) (*OSMount, error) {
	var osMount OSMount

	result := conn.db.WithContext(ctx).Preload("VolumeMeta").First(&osMount, "tenant_uuid = ?", tenantUUID)
	if result.Error != nil {
		return nil, result.Error
	}

	return &osMount, nil
}

func (conn *DBConn) DeleteOSMount(ctx context.Context, osMount *OSMount) error {
	if osMount.VolumeMeta != (VolumeMeta{}) {
		conn.db.WithContext(ctx).Delete(&osMount.VolumeMeta)
	}

	return conn.db.WithContext(ctx).Delete(osMount).Error
}

func (conn *DBConn) CreateAppMount(ctx context.Context, appMount *AppMount) error {
	return conn.db.WithContext(ctx).Create(appMount).Error
}

func (conn *DBConn) UpdateAppMount(ctx context.Context, appMount *AppMount) error {
	return conn.db.WithContext(ctx).Updates(appMount).Error
}

func (conn *DBConn) ReadAppMountByVolumeMetaID(ctx context.Context, volumeMetaID string) (*AppMount, error) {
	var appMount AppMount

	result := conn.db.WithContext(ctx).Preload("VolumeMeta").First(&appMount, "volume_meta_id = ?", volumeMetaID)
	if result.Error != nil {
		return nil, result.Error
	}

	return &appMount, nil
}

func (conn *DBConn) ReadAppMounts(ctx context.Context) ([]AppMount, error) {
	var appMounts []AppMount

	result := conn.db.WithContext(ctx).Preload("VolumeMeta").Find(&appMounts)
	if result.Error != nil {
		return nil, result.Error
	}

	return appMounts, nil
}

func (conn *DBConn) DeleteAppMount(ctx context.Context, appMount *AppMount) error {
	if appMount.VolumeMeta != (VolumeMeta{}) {
		conn.db.WithContext(ctx).Delete(&appMount.VolumeMeta)
	}

	return conn.db.WithContext(ctx).Delete(appMount).Error
}
