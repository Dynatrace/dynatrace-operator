package storage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAccess(t *testing.T) {
	dbPath = ":memory:"
	db := NewAccess().(*SqliteAccess)
	assert.NotNil(t, db.conn)
}

func TestSetup(t *testing.T) {
	dbPath = ":memory:"
	db := SqliteAccess{}
	err := db.Setup()

	assert.NoError(t, err)
	assert.True(t, checkIfTablesExist(&db))
}

func TestSetup_badPath(t *testing.T) {
	dbPath = "/asd"
	db := SqliteAccess{}
	err := db.Setup()

	assert.Error(t, err)

	assert.False(t, checkIfTablesExist(&db))
}

func TestConnect(t *testing.T) {
	path := ":memory:"
	db := SqliteAccess{}
	err := db.Connect(sqliteDriverName, path)
	assert.NoError(t, err)
	assert.NotNil(t, db.conn)
}

func TestConnect_badDriver(t *testing.T) {
	db := SqliteAccess{}
	err := db.Connect("die", "")
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

func TestInsertUpdateGetTenant(t *testing.T) {
	db := FakeMemoryDB()
	tenant := Tenant{
		UUID:          "123asd",
		LatestVersion: "123.456",
		Dynakube:      "dynakube-test",
	}

	// Get but empty
	gt, err := db.GetTenant(tenant.UUID)
	assert.NoError(t, err)
	assert.Nil(t, gt)

	// Insert
	err = db.InsertTenant(&tenant)
	assert.Nil(t, err)
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE UUID = ?;", tenantsTableName), tenant.UUID)
	var uuid string
	var lv string
	var dk string
	err = row.Scan(&uuid, &lv, &dk)
	assert.NoError(t, err)
	assert.Equal(t, uuid, tenant.UUID)
	assert.Equal(t, lv, tenant.LatestVersion)
	assert.Equal(t, dk, tenant.Dynakube)

	// Update
	tenant.LatestVersion = "132.546"
	err = db.UpdateTenant(&tenant)
	assert.NoError(t, err)
	row = db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE UUID = ?;", tenantsTableName), tenant.UUID)
	err = row.Scan(&uuid, &lv, &dk)
	assert.NoError(t, err)
	assert.Equal(t, uuid, tenant.UUID)
	assert.Equal(t, lv, tenant.LatestVersion)
	assert.Equal(t, dk, tenant.Dynakube)

	// Get
	gt, err = db.GetTenant(tenant.UUID)
	assert.NoError(t, err)
	assert.Equal(t, gt.UUID, tenant.UUID)
	assert.Equal(t, gt.LatestVersion, tenant.LatestVersion)
	assert.Equal(t, gt.Dynakube, tenant.Dynakube)

	// Get via Dynakube
	gt, err = db.GetTenantViaDynakube(tenant.Dynakube)
	assert.NoError(t, err)
	assert.Equal(t, gt.UUID, tenant.UUID)
	assert.Equal(t, gt.LatestVersion, tenant.LatestVersion)
	assert.Equal(t, gt.Dynakube, tenant.Dynakube)
}

func TestInsertGetDeleteVolume(t *testing.T) {
	db := FakeMemoryDB()
	volumeV1 := Volume{
		ID:         "123asd",
		PodName:    "pod1",
		Version:    "123.456",
		TenantUUID: "asl123",
	}
	volumeV2 := Volume{
		ID:         "23asd",
		PodName:    "pod2",
		Version:    "223.456",
		TenantUUID: "asl123",
	}

	// Get but empty
	vo, err := db.GetVolumeInfo(volumeV1.PodName)
	assert.NoError(t, err)
	assert.Nil(t, vo)

	// Insert
	err = db.InsertVolumeInfo(&volumeV1)
	assert.NoError(t, err)
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE ID = ?;", volumesTableName), volumeV1.ID)
	var id string
	var puid string
	var ver string
	var tuid string
	err = row.Scan(&id, &puid, &ver, &tuid)
	assert.NoError(t, err)
	assert.Equal(t, id, volumeV1.ID)
	assert.Equal(t, puid, volumeV1.PodName)
	assert.Equal(t, ver, volumeV1.Version)
	assert.Equal(t, tuid, volumeV1.TenantUUID)

	// Get via volume id
	vo, err = db.GetVolumeInfo(volumeV1.ID)
	assert.NoError(t, err)
	assert.Equal(t, vo.ID, volumeV1.ID)
	assert.Equal(t, vo.PodName, volumeV1.PodName)
	assert.Equal(t, vo.Version, volumeV1.Version)
	assert.Equal(t, vo.TenantUUID, volumeV1.TenantUUID)

	// Get used versions
	db.InsertVolumeInfo(&volumeV2)
	versions, err := db.GetUsedVersions(volumeV1.TenantUUID)
	assert.NoError(t, err)
	assert.Equal(t, len(versions), 2)
	assert.True(t, versions[volumeV1.Version])
	assert.True(t, versions[volumeV2.Version])

	// Get pod names
	podNames, err := db.GetPodNames()
	assert.NoError(t, err)
	assert.Equal(t, len(podNames), 2)
	assert.Equal(t, volumeV1.ID, podNames[volumeV1.PodName])
	assert.Equal(t, volumeV2.ID, podNames[volumeV2.PodName])

	// Delete
	err = db.DeleteVolumeInfo(volumeV2.ID)
	assert.NoError(t, err)
	versions, err = db.GetUsedVersions(volumeV1.TenantUUID)
	assert.NoError(t, err)
	assert.Equal(t, len(versions), 1)
	assert.True(t, versions[volumeV1.Version])
	assert.False(t, versions[volumeV2.Version])
}
