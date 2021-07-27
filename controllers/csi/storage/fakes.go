package storage

import "database/sql"

func emptyMemoryDB() SqliteAccess {
	path := ":memory:"
	db := SqliteAccess{}
	_ = db.Connect(sqliteDriverName, path)
	return db
}

func FakeMemoryDB() *SqliteAccess {
	db := emptyMemoryDB()
	_ = db.createTables()
	return &db
}

type FakeFailDB struct{}

func (f *FakeFailDB) Setup() error                           { return sql.ErrTxDone }
func (f *FakeFailDB) InsertTenant(tenant *Tenant) error      { return sql.ErrTxDone }
func (f *FakeFailDB) UpdateTenant(tenant *Tenant) error      { return sql.ErrTxDone }
func (f *FakeFailDB) GetTenant(uuid string) (*Tenant, error) { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetTenantViaDynakube(dynakube string) (*Tenant, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) InsertVolumeInfo(volume *Volume) error          { return sql.ErrTxDone }
func (f *FakeFailDB) DeleteVolumeInfo(volumeID string) error         { return sql.ErrTxDone }
func (f *FakeFailDB) GetVolumeInfo(volumeID string) (*Volume, error) { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetPodNames() (map[string]string, error)        { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetUsedVersions(tenantUUID string) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}
