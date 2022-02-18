package metadata

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	var dkTable string
	row = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", dynakubesTableName)
	row.Scan(&dkTable)
	assert.Equal(t, dkTable, dynakubesTableName)
}

func TestInsertDynakube(t *testing.T) {
	db := FakeMemoryDB()

	err := db.InsertDynakube(&testDynakube1)
	assert.Nil(t, err)
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE TenantUUID = ?;", dynakubesTableName), testDynakube1.TenantUUID)
	var uuid string
	var lv string
	var name string
	err = row.Scan(&name, &uuid, &lv)
	assert.NoError(t, err)
	assert.Equal(t, uuid, testDynakube1.TenantUUID)
	assert.Equal(t, lv, testDynakube1.LatestVersion)
	assert.Equal(t, name, testDynakube1.Name)
}

func TestGetDynakube_Empty(t *testing.T) {
	db := FakeMemoryDB()

	gt, err := db.GetDynakube(testDynakube1.TenantUUID)
	assert.NoError(t, err)
	assert.Nil(t, gt)
}

func TestGetDynakube(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertDynakube(&testDynakube1)
	assert.Nil(t, err)

	dynakube, err := db.GetDynakube(testDynakube1.Name)
	assert.NoError(t, err)
	assert.Equal(t, testDynakube1, *dynakube)
}

func TestUpdateDynakube(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertDynakube(&testDynakube1)
	assert.Nil(t, err)

	testDynakube1.LatestVersion = "132.546"
	err = db.UpdateDynakube(&testDynakube1)
	var uuid string
	var lv string
	var name string
	assert.NoError(t, err)
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE Name = ?;", dynakubesTableName), testDynakube1.Name)
	err = row.Scan(&name, &uuid, &lv)
	assert.NoError(t, err)
	assert.Equal(t, uuid, testDynakube1.TenantUUID)
	assert.Equal(t, lv, "132.546")
	assert.Equal(t, name, testDynakube1.Name)
}

func TestGetDynakubes(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertDynakube(&testDynakube1)
	assert.Nil(t, err)
	err = db.InsertDynakube(&testDynakube2)
	assert.Nil(t, err)

	dynakubes, err := db.GetDynakubes()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(dynakubes))
	assert.Equal(t, testDynakube1.TenantUUID, dynakubes[testDynakube1.Name])
	assert.Equal(t, testDynakube2.TenantUUID, dynakubes[testDynakube2.Name])
}

func TestDeleteDynakube(t *testing.T) {
	db := FakeMemoryDB()
	err := db.InsertDynakube(&testDynakube1)
	assert.Nil(t, err)
	err = db.InsertDynakube(&testDynakube2)
	assert.Nil(t, err)

	err = db.DeleteDynakube(testDynakube1.Name)
	assert.NoError(t, err)
	dynakubes, err := db.GetDynakubes()
	assert.NoError(t, err)
	assert.Equal(t, len(dynakubes), 1)
	assert.Equal(t, testDynakube2.TenantUUID, dynakubes[testDynakube2.Name])
}

func TestGetVolume_Empty(t *testing.T) {
	db := FakeMemoryDB()

	vo, err := db.GetVolume(testVolume1.PodName)
	assert.NoError(t, err)
	assert.Nil(t, vo)
}

func TestInsertVolume(t *testing.T) {
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

func TestInsertOsAgentVolume(t *testing.T) {
	db := FakeMemoryDB()

	now := time.Now()
	volume := OsAgentVolume{
		VolumeID:     "vol-4",
		TenantUUID:   testDynakube1.TenantUUID,
		Mounted:      true,
		LastModified: &now,
	}

	err := db.InsertOsAgentVolume(&volume)
	require.NoError(t, err)
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE TenantUUID = ?;", osAgentVolumesTableName), volume.TenantUUID)
	var volumeID string
	var tenantUUID string
	var mounted bool
	var lastModified time.Time
	err = row.Scan(&tenantUUID, &volumeID, &mounted, &lastModified)
	assert.NoError(t, err)
	assert.Equal(t, volumeID, volume.VolumeID)
	assert.Equal(t, tenantUUID, volume.TenantUUID)
	assert.Equal(t, mounted, volume.Mounted)
	assert.True(t, volume.LastModified.Equal(lastModified))
}

func TestGetOsAgentVolume(t *testing.T) {
	db := FakeMemoryDB()

	now := time.Now()
	expected := OsAgentVolume{
		VolumeID:     "vol-4",
		TenantUUID:   testDynakube1.TenantUUID,
		Mounted:      true,
		LastModified: &now,
	}

	err := db.InsertOsAgentVolume(&expected)
	require.NoError(t, err)
	actual, err := db.GetOsAgentVolume(expected.VolumeID)
	require.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, expected.VolumeID, actual.VolumeID)
	assert.Equal(t, expected.TenantUUID, actual.TenantUUID)
	assert.Equal(t, expected.Mounted, actual.Mounted)
	assert.True(t, expected.LastModified.Equal(*actual.LastModified))
}

func TestUpdateOsAgentVolume(t *testing.T) {
	db := FakeMemoryDB()

	now := time.Now()
	old := OsAgentVolume{
		VolumeID:     "vol-4",
		TenantUUID:   testDynakube1.TenantUUID,
		Mounted:      true,
		LastModified: &now,
	}

	err := db.InsertOsAgentVolume(&old)
	require.NoError(t, err)
	new := old
	new.Mounted = false
	err = db.UpdateOsAgentVolume(&new)
	require.NoError(t, err)

	actual, err := db.GetOsAgentVolume(old.VolumeID)
	require.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, old.VolumeID, actual.VolumeID)
	assert.Equal(t, old.TenantUUID, actual.TenantUUID)
	assert.NotEqual(t, old.Mounted, actual.Mounted)
	assert.True(t, old.LastModified.Equal(*actual.LastModified))
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
