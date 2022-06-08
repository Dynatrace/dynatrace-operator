package metadata

import (
	"database/sql"
)

var (
	testDynakube1 = Dynakube{
		TenantUUID:    "asc1",
		LatestVersion: "123",
		Name:          "dk1",
	}
	testDynakube2 = Dynakube{
		TenantUUID:    "asc2",
		LatestVersion: "223",
		Name:          "dk2",
	}
	testDynakube3 = Dynakube{
		TenantUUID:    "asc3",
		LatestVersion: "323",
		Name:          "dk3",
	}

	testVolume1 = Volume{
		VolumeID:   "vol-1",
		PodName:    "pod1",
		Version:    testDynakube1.LatestVersion,
		TenantUUID: testDynakube1.TenantUUID,
	}
	testVolume2 = Volume{
		VolumeID:   "vol-2",
		PodName:    "pod2",
		Version:    testDynakube2.LatestVersion,
		TenantUUID: testDynakube2.TenantUUID,
	}
	testVolume3 = Volume{
		VolumeID:   "vol-3",
		PodName:    "pod3",
		Version:    testDynakube3.LatestVersion,
		TenantUUID: testDynakube3.TenantUUID,
	}
)

func emptyMemoryDB() *SqliteAccess {
	db := SqliteAccess{}
	_ = db.connect(sqliteDriverName, ":memory:")
	return &db
}

func FakeMemoryDB() *SqliteAccess {
	db := SqliteAccess{}
	db.Setup(":memory:")
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
	row = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", dynakubesTableName)
	err = row.Scan(&tentatsTable)
	if err != nil {
		return false
	}
	if tentatsTable != dynakubesTableName || volumesTable != volumesTableName {
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
