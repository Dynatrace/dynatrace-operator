package storage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAccess(t *testing.T) {
	dbPath = ":memory:"
	db := NewAccess()

	assert.NotNil(t, db.conn)

	var podsTable string
	row := db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", podsTableName)
	row.Scan(&podsTable)
	assert.Equal(t, podsTable, podsTableName)

	var tentatsTable string
	row = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", tenantsTableName)
	row.Scan(&tentatsTable)
	assert.Equal(t, tentatsTable, tenantsTableName)
}

func TestNewAccess_badPath(t *testing.T) {
	dbPath = "/asd"
	db := NewAccess()

	var podsTable string
	row := db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", podsTableName)
	err := row.Scan(&podsTable)
	assert.Error(t, err)

	var tentatsTable string
	row = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", tenantsTableName)
	err = row.Scan(&tentatsTable)
	assert.Error(t, err)
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
	row := db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", podsTableName)
	row.Scan(&podsTable)
	assert.Equal(t, podsTable, podsTableName)

	var tentatsTable string
	row = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", tenantsTableName)
	row.Scan(&tentatsTable)
	assert.Equal(t, tentatsTable, tenantsTableName)

}

func TestInsertUpdateGetTenant(t *testing.T) {
	db := FakeMemoryDB()
	tenant := Tenant{
		UUID:          "123asd",
		LatestVersion: "123.456",
		Dynakube:      "dynakube-test",
	}

	// Insert
	err := db.InsertTenant(&tenant)
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
	gt, err := db.GetTenant(tenant.UUID)
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

func TestInsertGetDeletePod(t *testing.T) {
	db := FakeMemoryDB()
	podV1 := Pod{
		UID:        "123asd",
		VolumeID:   "1vol",
		Version:    "123.456",
		TenantUUID: "asl123",
	}
	podV2 := Pod{
		UID:        "23asd",
		VolumeID:   "2vol",
		Version:    "223.456",
		TenantUUID: "asl123",
	}

	// Insert
	err := db.InsertPodInfo(&podV1)
	assert.NoError(t, err)
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE UID = ?;", podsTableName), podV1.UID)
	var uid string
	var vid string
	var v string
	var tuid string
	err = row.Scan(&uid, &vid, &v, &tuid)
	assert.NoError(t, err)
	assert.Equal(t, uid, podV1.UID)
	assert.Equal(t, vid, podV1.VolumeID)
	assert.Equal(t, v, podV1.Version)
	assert.Equal(t, tuid, podV1.TenantUUID)

	// Get via volume id
	p, err := db.GetPodViaVolumeId(podV1.VolumeID)
	assert.NoError(t, err)
	assert.Equal(t, p.UID, podV1.UID)
	assert.Equal(t, p.VolumeID, podV1.VolumeID)
	assert.Equal(t, p.Version, podV1.Version)
	assert.Equal(t, p.TenantUUID, podV1.TenantUUID)

	// Get used versions
	db.InsertPodInfo(&podV2)
	versions, err := db.GetUsedVersions(podV1.TenantUUID)
	assert.NoError(t, err)
	assert.Equal(t, len(versions), 2)
	assert.True(t, versions[podV1.Version])
	assert.True(t, versions[podV2.Version])

	// Delete
	err = db.DeletePodInfo(&podV2)
	assert.NoError(t, err)
	versions, err = db.GetUsedVersions(podV1.TenantUUID)
	assert.NoError(t, err)
	assert.Equal(t, len(versions), 1)
	assert.True(t, versions[podV1.Version])
	assert.False(t, versions[podV2.Version])
}
