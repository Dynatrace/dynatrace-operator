package metadata

import (
	"database/sql"
)

var (
	testTenant1 = Tenant{
		TenantUUID:    "asc1",
		LatestVersion: "123",
		Dynakube:      "dk1",
	}
	testTenant2 = Tenant{
		TenantUUID:    "asc2",
		LatestVersion: "223",
		Dynakube:      "dk2",
	}
	testTenant3 = Tenant{
		TenantUUID:    "asc3",
		LatestVersion: "323",
		Dynakube:      "dk3",
	}

	testVolume1 = Volume{
		VolumeID:   "vol-1",
		PodName:    "pod1",
		Version:    testTenant1.LatestVersion,
		TenantUUID: testTenant1.TenantUUID,
	}
	testVolume2 = Volume{
		VolumeID:   "vol-2",
		PodName:    "pod2",
		Version:    testTenant2.LatestVersion,
		TenantUUID: testTenant2.TenantUUID,
	}
	testVolume3 = Volume{
		VolumeID:   "vol-3",
		PodName:    "pod3",
		Version:    testTenant3.LatestVersion,
		TenantUUID: testTenant3.TenantUUID,
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

func (f *FakeFailDB) Setup(dbPath string) error                    { return sql.ErrTxDone }
func (f *FakeFailDB) InsertTenant(tenant *Tenant) error            { return sql.ErrTxDone }
func (f *FakeFailDB) UpdateTenant(tenant *Tenant) error            { return sql.ErrTxDone }
func (f *FakeFailDB) DeleteTenant(tenantUUID string) error         { return sql.ErrTxDone }
func (f *FakeFailDB) GetTenant(tenantUUID string) (*Tenant, error) { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetTenantViaDynakube(dynakubeName string) (*Tenant, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetDynakubes() (map[string]string, error)   { return nil, sql.ErrTxDone }
func (f *FakeFailDB) InsertVolume(volume *Volume) error          { return sql.ErrTxDone }
func (f *FakeFailDB) DeleteVolume(volumeID string) error         { return sql.ErrTxDone }
func (f *FakeFailDB) GetVolume(volumeID string) (*Volume, error) { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetPodNames() (map[string]string, error)    { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetUsedVersions(tenantUUID string) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}
