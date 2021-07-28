package storage

import (
	"database/sql"
)

func emptyMemoryDB() *SqliteAccess {
	dbPath = ":memory:"
	db := SqliteAccess{}
	_ = db.connect(sqliteDriverName, dbPath)
	return &db
}

func FakeMemoryDB() *SqliteAccess {
	dbPath = ":memory:"
	db := SqliteAccess{}
	db.Setup()
	_ = db.createTables()
	return &db
}

func checkIfTablesExist(db *SqliteAccess) bool {
	var volumesTable string
	row := db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", volumesTableName)
	err := row.Scan(&volumesTable)
	if err != nil {
		return false
	}
	var tentatsTable string
	row = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", tenantsTableName)
	err = row.Scan(&tentatsTable)
	if err != nil {
		return false
	}
	if tentatsTable != tenantsTableName || volumesTable != volumesTableName {
		return false
	}
	return true
}

type FakeFailDB struct{}

func (f *FakeFailDB) Setup() error                           { return sql.ErrTxDone }
func (f *FakeFailDB) InsertTenant(tenant *Tenant) error      { return sql.ErrTxDone }
func (f *FakeFailDB) UpdateTenant(tenant *Tenant) error      { return sql.ErrTxDone }
func (f *FakeFailDB) DeleteTenant(uuid string) error         { return sql.ErrTxDone }
func (f *FakeFailDB) GetTenant(uuid string) (*Tenant, error) { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetTenantViaDynakube(dynakube string) (*Tenant, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetDynakubes() (map[string]string, error)       { return nil, sql.ErrTxDone }
func (f *FakeFailDB) InsertVolumeInfo(volume *Volume) error          { return sql.ErrTxDone }
func (f *FakeFailDB) DeleteVolumeInfo(volumeID string) error         { return sql.ErrTxDone }
func (f *FakeFailDB) GetVolumeInfo(volumeID string) (*Volume, error) { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetPodNames() (map[string]string, error)        { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetUsedVersions(tenantUUID string) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}
