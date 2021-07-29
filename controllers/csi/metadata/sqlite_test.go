package metadata

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

func TestInsertUpdateGetTenant(t *testing.T) {
	db := FakeMemoryDB()
	tenant1 := Tenant{
		TenantUUID:    "123asd",
		LatestVersion: "123.456",
		Dynakube:      "dynakube-test1",
	}
	tenant2 := Tenant{
		TenantUUID:    "223asd",
		LatestVersion: "223.456",
		Dynakube:      "dynakube-test2",
	}

	// Get but empty
	gt, err := db.GetTenant(tenant1.TenantUUID)
	assert.NoError(t, err)
	assert.Nil(t, gt)

	// Insert
	err = db.InsertTenant(&tenant1)
	assert.Nil(t, err)
	err = db.InsertTenant(&tenant2)
	assert.Nil(t, err)
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE UUID = ?;", tenantsTableName), tenant1.TenantUUID)
	var uuid string
	var lv string
	var dk string
	err = row.Scan(&uuid, &lv, &dk)
	assert.NoError(t, err)
	assert.Equal(t, uuid, tenant1.TenantUUID)
	assert.Equal(t, lv, tenant1.LatestVersion)
	assert.Equal(t, dk, tenant1.Dynakube)

	// Update
	tenant1.LatestVersion = "132.546"
	err = db.UpdateTenant(&tenant1)
	assert.NoError(t, err)
	row = db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE UUID = ?;", tenantsTableName), tenant1.TenantUUID)
	err = row.Scan(&uuid, &lv, &dk)
	assert.NoError(t, err)
	assert.Equal(t, uuid, tenant1.TenantUUID)
	assert.Equal(t, lv, tenant1.LatestVersion)
	assert.Equal(t, dk, tenant1.Dynakube)

	// Get
	gt, err = db.GetTenant(tenant1.TenantUUID)
	assert.NoError(t, err)
	assert.Equal(t, gt.TenantUUID, tenant1.TenantUUID)
	assert.Equal(t, gt.LatestVersion, tenant1.LatestVersion)
	assert.Equal(t, gt.Dynakube, tenant1.Dynakube)

	// Get via Dynakube
	gt, err = db.GetTenantViaDynakube(tenant1.Dynakube)
	assert.NoError(t, err)
	assert.Equal(t, gt.TenantUUID, tenant1.TenantUUID)
	assert.Equal(t, gt.LatestVersion, tenant1.LatestVersion)
	assert.Equal(t, gt.Dynakube, tenant1.Dynakube)

	// Get Dynakubes
	dynakubes, err := db.GetDynakubes()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(dynakubes))
	assert.Equal(t, tenant1.TenantUUID, dynakubes[tenant1.Dynakube])
	assert.Equal(t, tenant2.TenantUUID, dynakubes[tenant2.Dynakube])

	// Delete
	err = db.DeleteTenant(tenant1.TenantUUID)
	assert.NoError(t, err)
	dynakubes, err = db.GetDynakubes()
	assert.NoError(t, err)
	assert.Equal(t, len(dynakubes), 1)
	assert.Equal(t, tenant2.TenantUUID, dynakubes[tenant2.Dynakube])
}

func TestInsertGetDeleteVolume(t *testing.T) {
	db := FakeMemoryDB()
	volumeV1 := Volume{
		VolumeID:   "123asd",
		PodName:    "pod1",
		Version:    "123.456",
		TenantUUID: "asl123",
	}
	volumeV2 := Volume{
		VolumeID:   "23asd",
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
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE ID = ?;", volumesTableName), volumeV1.VolumeID)
	var id string
	var puid string
	var ver string
	var tuid string
	err = row.Scan(&id, &puid, &ver, &tuid)
	assert.NoError(t, err)
	assert.Equal(t, id, volumeV1.VolumeID)
	assert.Equal(t, puid, volumeV1.PodName)
	assert.Equal(t, ver, volumeV1.Version)
	assert.Equal(t, tuid, volumeV1.TenantUUID)

	// Get via volume id
	vo, err = db.GetVolumeInfo(volumeV1.VolumeID)
	assert.NoError(t, err)
	assert.Equal(t, vo.VolumeID, volumeV1.VolumeID)
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
	assert.Equal(t, volumeV1.VolumeID, podNames[volumeV1.PodName])
	assert.Equal(t, volumeV2.VolumeID, podNames[volumeV2.PodName])

	// Delete
	err = db.DeleteVolumeInfo(volumeV2.VolumeID)
	assert.NoError(t, err)
	versions, err = db.GetUsedVersions(volumeV1.TenantUUID)
	assert.NoError(t, err)
	assert.Equal(t, len(versions), 1)
	assert.True(t, versions[volumeV1.Version])
	assert.False(t, versions[volumeV2.Version])
}
