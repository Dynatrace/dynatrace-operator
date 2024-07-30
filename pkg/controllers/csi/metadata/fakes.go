package metadata

import (
	"context"
	"database/sql"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func emptyMemoryDB() *GormConn {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		return nil
	}

	return &GormConn{ctx: context.Background(), db: db}
}

func FakeMemoryDB() *GormConn {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		return nil
	}

	gormConn := &GormConn{ctx: context.Background(), db: db}

	err = gormConn.InitGormSchema()
	if err != nil {
		log.Error(err, "Couldn't initialize GORM schema")

		return nil
	}

	return gormConn
}

type FakeFailDB struct{}

func (f *FakeFailDB) SchemaMigration() error {
	return sql.ErrTxDone
}

func (f *FakeFailDB) ReadTenantConfig(tenantConfig TenantConfig) (*TenantConfig, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadCodeModule(codeModule CodeModule) (*CodeModule, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadOSMount(osMount OSMount) (*OSMount, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadUnscopedOSMount(osMount OSMount) (*OSMount, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadVolumeMeta(volumeMeta VolumeMeta) (*VolumeMeta, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadAppMount(appMount AppMount) (*AppMount, error) {
	return nil, sql.ErrTxDone
}

func (f *FakeFailDB) ReadTenantConfigs() ([]TenantConfig, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadCodeModules() ([]CodeModule, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadOSMounts() ([]OSMount, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadAppMounts() ([]AppMount, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadVolumeMetas() ([]VolumeMeta, error) {
	return nil, sql.ErrTxDone
}

func (f *FakeFailDB) CreateTenantConfig(tenantConfig *TenantConfig) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) CreateCodeModule(codeModule *CodeModule) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) CreateOSMount(osMount *OSMount) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) CreateAppMount(appMount *AppMount) error {
	return sql.ErrTxDone
}

func (f *FakeFailDB) UpdateTenantConfig(tenantConfig *TenantConfig) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) UpdateOSMount(osMount *OSMount) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) UpdateAppMount(appMount *AppMount) error {
	return sql.ErrTxDone
}

func (f *FakeFailDB) DeleteTenantConfig(tenantConfig *TenantConfig, cascade bool) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) DeleteCodeModule(codeModule *CodeModule) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) DeleteOSMount(osMount *OSMount) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) DeleteAppMount(appMount *AppMount) error {
	return sql.ErrTxDone
}

func (f *FakeFailDB) IsCodeModuleOrphaned(codeModule *CodeModule) (bool, error) {
	return false, sql.ErrTxDone
}
func (f *FakeFailDB) RestoreOSMount(osMount *OSMount) (*OSMount, error) {
	return nil, sql.ErrTxDone
}
