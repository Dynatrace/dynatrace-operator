package metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAccess(t *testing.T) {
	db, err := NewAccess(":memory:")
	assert.Nil(t, err)
	assert.NotNil(t, db.(*SqliteAccess).conn)
}

func TestSetup(t *testing.T) {
	db := SqliteAccess{}
	err := db.Setup(":memory:")

	assert.NoError(t, err)
	assert.True(t, checkIfTablesExist(&db))
}

func TestSetup_badPath(t *testing.T) {
	db := SqliteAccess{}
	err := db.Setup("/asd")

	assert.Error(t, err)

	assert.False(t, checkIfTablesExist(&db))
}

func TestConnect(t *testing.T) {
	path := ":memory:"
	db := SqliteAccess{}
	err := db.connect(sqliteDriverName, path)
	assert.NoError(t, err)
	assert.NotNil(t, db.conn)
}

func TestConnect_badDriver(t *testing.T) {
	db := SqliteAccess{}
	err := db.connect("die", "")
	assert.Error(t, err)
	assert.Nil(t, db.conn)
}

func TestCreateTables(t *testing.T) {
	db := emptyMemoryDB()

	err := db.createTables()

	assert.Nil(t, err)

	var podsTable string
	row := db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", volumesTableName)
	row.Scan(&podsTable)
	assert.Equal(t, podsTable, volumesTableName)

	var tenantsTable string
	row = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", tenantsTableName)
	row.Scan(&tenantsTable)
	assert.Equal(t, tenantsTable, tenantsTableName)

}

func TestInsertTenant(t *testing.T) {
	db := FakeMemoryDB()

	// Insert
	err := db.InsertTenant(&testTenant1)
	assert.Nil(t, err)
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE UUID = ?;", tenantsTableName), testTenant1.TenantUUID)
	var uuid string
	var lv string
	var dk string
	err = row.Scan(&uuid, &lv, &dk)
	assert.NoError(t, err)
	assert.Equal(t, uuid, testTenant1.TenantUUID)
	assert.Equal(t, lv, testTenant1.LatestVersion)
	assert.Equal(t, dk, testTenant1.Dynakube)
}

func TestGetTenant_Empty(t *testing.T) {
	db := FakeMemoryDB()

	gt, err := db.GetTenant(testTenant1.TenantUUID)
	assert.NoError(t, err)
	assert.Nil(t, gt)
}

func TestGetTenant(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertTenant(&testTenant1)
	assert.Nil(t, err)

	tenant, err := db.GetTenant(testTenant1.TenantUUID)
	assert.NoError(t, err)
	assert.Equal(t, testTenant1, *tenant)
}

func TestGetTenantViaDynakube(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertTenant(&testTenant1)
	assert.Nil(t, err)

	tenant, err := db.GetTenantViaDynakube(testTenant1.Dynakube)
	assert.NoError(t, err)
	assert.Equal(t, testTenant1, *tenant)
}

func TestUpdateTenant(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertTenant(&testTenant1)
	assert.Nil(t, err)

	testTenant1.LatestVersion = "132.546"
	err = db.UpdateTenant(&testTenant1)
	var uuid string
	var lv string
	var dk string
	assert.NoError(t, err)
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE UUID = ?;", tenantsTableName), testTenant1.TenantUUID)
	err = row.Scan(&uuid, &lv, &dk)
	assert.NoError(t, err)
	assert.Equal(t, uuid, testTenant1.TenantUUID)
	assert.Equal(t, lv, "132.546")
	assert.Equal(t, dk, testTenant1.Dynakube)
}

func TestGetDynakubes(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertTenant(&testTenant1)
	assert.Nil(t, err)
	err = db.InsertTenant(&testTenant2)
	assert.Nil(t, err)

	dynakubes, err := db.GetDynakubes()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(dynakubes))
	assert.Equal(t, testTenant1.TenantUUID, dynakubes[testTenant1.Dynakube])
	assert.Equal(t, testTenant2.TenantUUID, dynakubes[testTenant2.Dynakube])
}

func TestDeleteTenant(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertTenant(&testTenant1)
	assert.Nil(t, err)
	err = db.InsertTenant(&testTenant2)
	assert.Nil(t, err)

	err = db.DeleteTenant(testTenant1.TenantUUID)
	assert.NoError(t, err)
	dynakubes, err := db.GetDynakubes()
	assert.NoError(t, err)
	assert.Equal(t, len(dynakubes), 1)
	assert.Equal(t, testTenant2.TenantUUID, dynakubes[testTenant2.Dynakube])
}

func TestGetVolume_Empty(t *testing.T) {
	db := FakeMemoryDB()

	vo, err := db.GetVolume(testVolume1.PodName)
	assert.NoError(t, err)
	assert.Nil(t, vo)
}

func TestGetInsert(t *testing.T) {
	db := FakeMemoryDB()

	err := db.InsertVolume(&testVolume1)
	assert.NoError(t, err)
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE ID = ?;", volumesTableName), testVolume1.VolumeID)
	var id string
	var puid string
	var ver string
	var tuid string
	err = row.Scan(&id, &puid, &ver, &tuid)
	assert.NoError(t, err)
	assert.Equal(t, id, testVolume1.VolumeID)
	assert.Equal(t, puid, testVolume1.PodName)
	assert.Equal(t, ver, testVolume1.Version)
	assert.Equal(t, tuid, testVolume1.TenantUUID)

}

func TestGetVolume(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertVolume(&testVolume1)
	assert.NoError(t, err)

	volume, err := db.GetVolume(testVolume1.VolumeID)
	assert.NoError(t, err)
	assert.Equal(t, testVolume1, *volume)
}

func TestGetUsedVersions(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertVolume(&testVolume1)
	testVolume11 := testVolume1
	testVolume11.VolumeID = "vol-11"
	testVolume11.Version = "321"
	assert.NoError(t, err)
	err = db.InsertVolume(&testVolume11)
	assert.NoError(t, err)

	versions, err := db.GetUsedVersions(testVolume1.TenantUUID)
	assert.NoError(t, err)
	assert.Equal(t, len(versions), 2)
	assert.True(t, versions[testVolume1.Version])
	assert.True(t, versions[testVolume11.Version])
}

func TestGetPodNames(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertVolume(&testVolume1)
	assert.NoError(t, err)
	err = db.InsertVolume(&testVolume2)
	assert.NoError(t, err)

	podNames, err := db.GetPodNames()
	assert.NoError(t, err)
	assert.Equal(t, len(podNames), 2)
	assert.Equal(t, testVolume1.VolumeID, podNames[testVolume1.PodName])
	assert.Equal(t, testVolume2.VolumeID, podNames[testVolume2.PodName])
}

func TestDeleteVolume(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertVolume(&testVolume1)
	assert.NoError(t, err)
	err = db.InsertVolume(&testVolume2)
	assert.NoError(t, err)

	err = db.DeleteVolume(testVolume2.VolumeID)
	assert.NoError(t, err)
	podNames, err := db.GetPodNames()
	assert.NoError(t, err)
	assert.Equal(t, len(podNames), 1)
	assert.Equal(t, testVolume1.VolumeID, podNames[testVolume1.PodName])
}
