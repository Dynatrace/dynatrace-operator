package metadata

import (
	"database/sql"
)

func emptyMemoryDB() *SqliteAccess {
	db := SqliteAccess{}
	_ = db.connect(sqliteDriverName, ":memory:")
	return &db
}

func FakeMemoryDB() *SqliteAccess {
	db := SqliteAccess{}
	db.Setup(":memory:")
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
	var tenantsTable string
	row = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", dynakubesTableName)
	err = row.Scan(&tenantsTable)
	if err != nil {
		return false
	}
	if tenantsTable != dynakubesTableName || volumesTable != volumesTableName {
		return false
	}
	return true
}

type FakeFailDB struct{}

func (f *FakeFailDB) Setup(dbPath string) error                          { return sql.ErrTxDone }
func (f *FakeFailDB) InsertDynakube(tenant *Dynakube) error              { return sql.ErrTxDone }
func (f *FakeFailDB) UpdateDynakube(tenant *Dynakube) error              { return sql.ErrTxDone }
func (f *FakeFailDB) DeleteDynakube(dynakubeName string) error           { return sql.ErrTxDone }
func (f *FakeFailDB) GetDynakube(dynakubeName string) (*Dynakube, error) { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetTenantsToDynakubes() (map[string]string, error)  { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetAllDynakubes() ([]*Dynakube, error)              { return nil, sql.ErrTxDone }

func (f *FakeFailDB) InsertOsAgentVolume(volume *OsAgentVolume) error { return sql.ErrTxDone }
func (f *FakeFailDB) GetOsAgentVolumeViaVolumeID(volumeID string) (*OsAgentVolume, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetOsAgentVolumeViaTenantUUID(volumeID string) (*OsAgentVolume, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) UpdateOsAgentVolume(volume *OsAgentVolume) error { return sql.ErrTxDone }
func (f *FakeFailDB) GetAllOsAgentVolumes() ([]*OsAgentVolume, error) { return nil, sql.ErrTxDone }

func (f *FakeFailDB) InsertVolume(volume *Volume) error          { return sql.ErrTxDone }
func (f *FakeFailDB) DeleteVolume(volumeID string) error         { return sql.ErrTxDone }
func (f *FakeFailDB) GetVolume(volumeID string) (*Volume, error) { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetAllVolumes() ([]*Volume, error)          { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetPodNames() (map[string]string, error)    { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetUsedVersions(tenantUUID string) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetAllUsedVersions() (map[string]bool, error) { return nil, sql.ErrTxDone }

func (f *FakeFailDB) GetUsedImageDigests() (map[string]bool, error) { return nil, sql.ErrTxDone }
