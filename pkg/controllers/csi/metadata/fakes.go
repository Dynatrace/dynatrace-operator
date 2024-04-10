package metadata

import (
	"context"
	"database/sql"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func emptyMemoryDB() *DBConn {
	db := DBConn{}

	return &db
}

func FakeMemoryDB() *DBConn {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return nil
	}

	dbConn := &DBConn{db: db}

	err = dbConn.SchemaMigration(context.Background())
	if err != nil {
		return nil
	}

	return dbConn
}

type FakeFailDB struct{}

func (f *FakeFailDB) Setup(_ context.Context, _ string) error { return sql.ErrTxDone }

func (f *FakeFailDB) SchemaMigration(ctx context.Context) error {
	return sql.ErrTxDone
}

func (f *FakeFailDB) CreateTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) UpdateTenantConfig(ctx context.Context, tenantConfig *TenantConfig) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) DeleteTenantConfig(ctx context.Context, tenantConfig *TenantConfig, cascade bool) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) ReadTenantConfig(ctx context.Context, tenantConfig TenantConfig) (*TenantConfig, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadTenantConfigs(ctx context.Context) ([]TenantConfig, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) CreateCodeModule(ctx context.Context, codeModule *CodeModule) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) DeleteCodeModule(ctx context.Context, codeModule *CodeModule) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) ReadCodeModule(ctx context.Context, codeModule CodeModule) (*CodeModule, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadCodeModules(ctx context.Context) ([]CodeModule, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) IsCodeModuleOrphaned(ctx context.Context, codeModule *CodeModule) (bool, error) {
	return false, sql.ErrTxDone
}
func (f *FakeFailDB) CreateOSMount(ctx context.Context, osMount *OSMount) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) UpdateOSMount(ctx context.Context, osMount *OSMount) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) DeleteOSMount(ctx context.Context, osMount *OSMount) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) ReadOSMount(ctx context.Context, osMount OSMount) (*OSMount, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadOSMounts(ctx context.Context) ([]OSMount, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) CreateAppMount(ctx context.Context, appMount *AppMount) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) UpdateAppMount(ctx context.Context, appMount *AppMount) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) DeleteAppMount(ctx context.Context, appMount *AppMount) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) ReadAppMount(ctx context.Context, appMount AppMount) (*AppMount, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadAppMounts(ctx context.Context) ([]AppMount, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadVolumeMeta(ctx context.Context, volumeMeta VolumeMeta) (*VolumeMeta, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) ReadVolumeMetas(ctx context.Context) ([]VolumeMeta, error) {
	return nil, sql.ErrTxDone
}
