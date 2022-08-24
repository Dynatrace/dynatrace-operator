package metadata

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccess(t *testing.T) {
	db, err := NewAccess(":memory:")
	require.NoError(t, err)
	assert.NotNil(t, db.(*SqliteAccess).conn)
}

func TestSetup(t *testing.T) {
	db := SqliteAccess{}
	err := db.Setup(":memory:")

	require.NoError(t, err)
	assert.True(t, checkIfTablesExist(&db))
}

func TestSetup_badPath(t *testing.T) {
	db := SqliteAccess{}
	err := db.Setup("/asd")
	require.Error(t, err)

	assert.False(t, checkIfTablesExist(&db))
}

func TestConnect(t *testing.T) {
	path := ":memory:"
	db := SqliteAccess{}
	err := db.connect(sqliteDriverName, path)
	require.NoError(t, err)
	assert.NotNil(t, db.conn)
}

func TestConnect_badDriver(t *testing.T) {
	db := SqliteAccess{}
	err := db.connect("die", "")
	require.Error(t, err)
	assert.Nil(t, db.conn)
}

func TestCreateTables(t *testing.T) {
	t.Run("volume table is created correctly", func(t *testing.T) {
		db := emptyMemoryDB()

		err := db.createTables()
		require.NoError(t, err)

		var volumeTableName string
		row := db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", volumesTableName)
		err = row.Scan(&volumeTableName)
		require.NoError(t, err)
		assert.Equal(t, volumeTableName, volumesTableName)

		rows, err := db.conn.Query("PRAGMA table_info(" + volumesTableName + ")")
		assert.NoError(t, err)
		assert.NotNil(t, rows)

		columns := []string{
			"ID",
			"PodName",
			"Version",
			"TenantUUID",
			"MountAttempts",
		}

		for _, column := range columns {
			assert.True(t, rows.Next())

			var id, name, columnType, notNull, primaryKey string
			var defaultValue = new(string)

			err = rows.Scan(&id, &name, &columnType, &notNull, &defaultValue, &primaryKey)

			assert.NoError(t, err)
			assert.Equal(t, column, name)

			if column == "MountAttempts" {
				assert.Equal(t, "0", *defaultValue)
				assert.Equal(t, "1", notNull)
			}
		}
	})
	t.Run("dynakube table is created correctly", func(t *testing.T) {
		db := emptyMemoryDB()

		err := db.createTables()
		require.NoError(t, err)

		var dkTable string
		row := db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", dynakubesTableName)
		err = row.Scan(&dkTable)
		require.NoError(t, err)
		assert.Equal(t, dkTable, dynakubesTableName)

		rows, err := db.conn.Query("PRAGMA table_info(" + dynakubesTableName + ")")
		assert.NoError(t, err)
		assert.NotNil(t, rows)

		columns := []string{
			"Name",
			"TenantUUID",
			"LatestVersion",
			"ImageDigest",
			"MaxFailedMountAttempts",
		}

		for _, column := range columns {
			assert.True(t, rows.Next())

			var id, name, columnType, notNull, primaryKey string
			var defaultValue = new(string)

			err = rows.Scan(&id, &name, &columnType, &notNull, &defaultValue, &primaryKey)

			assert.NoError(t, err)
			assert.Equal(t, column, name)

			if column == "MaxFailedMountAttempts" {
				maxFailedMountAttempts, err := strconv.Atoi(*defaultValue)
				assert.NoError(t, err)
				assert.Equal(t, defaultSqlMaxFailedMountAttempts, *defaultValue)
				assert.Equal(t, defaultMaxFailedMountAttempts, maxFailedMountAttempts)
				assert.Equal(t, "0", notNull)
			}
		}
	})
}

func TestInsertDynakube(t *testing.T) {
	testDynakube1 := createTestDynakube(1)

	db := FakeMemoryDB()

	err := db.InsertDynakube(&testDynakube1)
	require.NoError(t, err)

	var uuid, lv, name string
	var imageDigest string
	var maxMountAttempts int
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE TenantUUID = ?;", dynakubesTableName), testDynakube1.TenantUUID)
	err = row.Scan(&name, &uuid, &lv, &imageDigest, &maxMountAttempts)
	require.NoError(t, err)
	assert.Equal(t, testDynakube1.TenantUUID, uuid)
	assert.Equal(t, testDynakube1.LatestVersion, lv)
	assert.Equal(t, testDynakube1.Name, name)
	assert.Equal(t, testDynakube1.ImageDigest, imageDigest)
	assert.Equal(t, testDynakube1.MaxFailedMountAttempts, maxMountAttempts)
}

func TestGetDynakube_Empty(t *testing.T) {
	testDynakube1 := createTestDynakube(1)
	db := FakeMemoryDB()

	gt, err := db.GetDynakube(testDynakube1.TenantUUID)
	require.NoError(t, err)
	assert.Nil(t, gt)
}

func TestGetDynakube(t *testing.T) {
	t.Run("get dynakube", func(t *testing.T) {
		testDynakube1 := createTestDynakube(1)
		db := FakeMemoryDB()
		err := db.InsertDynakube(&testDynakube1)
		require.NoError(t, err)

		dynakube, err := db.GetDynakube(testDynakube1.Name)
		require.NoError(t, err)
		assert.Equal(t, testDynakube1, *dynakube)
	})
	t.Run("get dynakube with maxFailedMountAttempts set to null", func(t *testing.T) {
		db := FakeMemoryDB()

		require.NoError(t, db.createTables())
		require.NoError(t, db.executeStatement(insertDynakubeStatement, testName, testUUID, testVersion, testDigest, nil))

		dynakube, err := db.GetDynakube(testName)
		require.NoError(t, err)

		assert.Equal(t, testName, dynakube.Name)
		assert.Equal(t, testUUID, dynakube.TenantUUID)
		assert.Equal(t, testVersion, dynakube.LatestVersion)
		assert.Equal(t, testDigest, dynakube.ImageDigest)
		assert.Equal(t, 3, dynakube.MaxFailedMountAttempts)
	})
}

func TestUpdateDynakube(t *testing.T) {
	testDynakube1 := createTestDynakube(1)
	db := FakeMemoryDB()
	err := db.InsertDynakube(&testDynakube1)
	require.NoError(t, err)

	copyDynakube := testDynakube1
	copyDynakube.LatestVersion = "132.546"
	copyDynakube.ImageDigest = ""
	copyDynakube.MaxFailedMountAttempts = 10
	err = db.UpdateDynakube(&copyDynakube)
	require.NoError(t, err)

	var uuid, lv, name string
	var imageDigest string
	var maxFailedMountAttempts int

	row := db.conn.QueryRow(fmt.Sprintf("SELECT Name, TenantUUID, LatestVersion, ImageDigest, MaxFailedMountAttempts FROM %s WHERE Name = ?;", dynakubesTableName), copyDynakube.Name)
	err = row.Scan(&name, &uuid, &lv, &imageDigest, &maxFailedMountAttempts)

	require.NoError(t, err)
	assert.Equal(t, copyDynakube.TenantUUID, uuid)
	assert.Equal(t, copyDynakube.LatestVersion, lv)
	assert.Equal(t, copyDynakube.Name, name)
	assert.Equal(t, copyDynakube.MaxFailedMountAttempts, maxFailedMountAttempts)
	assert.Empty(t, imageDigest)
}

func TestGetTenantsToDynakubes(t *testing.T) {
	testDynakube1 := createTestDynakube(1)
	testDynakube2 := createTestDynakube(2)

	db := FakeMemoryDB()
	err := db.InsertDynakube(&testDynakube1)
	require.NoError(t, err)
	err = db.InsertDynakube(&testDynakube2)
	require.NoError(t, err)

	dynakubes, err := db.GetTenantsToDynakubes()
	require.NoError(t, err)
	assert.Equal(t, 2, len(dynakubes))
	assert.Equal(t, testDynakube1.TenantUUID, dynakubes[testDynakube1.Name])
	assert.Equal(t, testDynakube2.TenantUUID, dynakubes[testDynakube2.Name])
}

func TestGetAllDynakubes(t *testing.T) {
	t.Run("get multiple dynakubes", func(t *testing.T) {
		testDynakube1 := createTestDynakube(1)
		testDynakube2 := createTestDynakube(2)

		db := FakeMemoryDB()
		err := db.InsertDynakube(&testDynakube1)
		require.NoError(t, err)
		err = db.InsertDynakube(&testDynakube2)
		require.NoError(t, err)

		dynakubes, err := db.GetAllDynakubes()
		require.NoError(t, err)
		assert.Equal(t, 2, len(dynakubes))
	})
	t.Run("get dynakubes with null values", func(t *testing.T) {
		db := FakeMemoryDB()

		require.NoError(t, db.createTables())
		require.NoError(t, db.executeStatement(insertDynakubeStatement, testName+"-1", testUUID, testVersion, testDigest, nil))
		require.NoError(t, db.executeStatement(insertDynakubeStatement, testName+"-2", testUUID, testVersion, testDigest, nil))
		require.NoError(t, db.executeStatement(insertDynakubeStatement, testName+"-3", testUUID, testVersion, testDigest, 1))

		dynakubes, err := db.GetAllDynakubes()
		require.NoError(t, err)

		assert.Equal(t, testName+"-1", dynakubes[0].Name)
		assert.Equal(t, testUUID, dynakubes[0].TenantUUID)
		assert.Equal(t, testVersion, dynakubes[0].LatestVersion)
		assert.Equal(t, testDigest, dynakubes[0].ImageDigest)
		assert.Equal(t, 3, dynakubes[0].MaxFailedMountAttempts)

		assert.Equal(t, testName+"-2", dynakubes[1].Name)
		assert.Equal(t, testUUID, dynakubes[1].TenantUUID)
		assert.Equal(t, testVersion, dynakubes[1].LatestVersion)
		assert.Equal(t, testDigest, dynakubes[1].ImageDigest)
		assert.Equal(t, 3, dynakubes[1].MaxFailedMountAttempts)

		assert.Equal(t, testName+"-3", dynakubes[2].Name)
		assert.Equal(t, testUUID, dynakubes[2].TenantUUID)
		assert.Equal(t, testVersion, dynakubes[2].LatestVersion)
		assert.Equal(t, testDigest, dynakubes[2].ImageDigest)
		assert.Equal(t, 1, dynakubes[2].MaxFailedMountAttempts)
	})
}

func TestGetAllVolumes(t *testing.T) {
	testVolume1 := createTestVolume(1)
	testVolume2 := createTestVolume(2)

	db := FakeMemoryDB()
	err := db.InsertVolume(&testVolume1)
	require.NoError(t, err)
	err = db.InsertVolume(&testVolume2)
	require.NoError(t, err)

	volumes, err := db.GetAllVolumes()
	require.NoError(t, err)
	assert.Equal(t, 2, len(volumes))
	assert.Equal(t, testVolume1, *volumes[0])
	assert.Equal(t, testVolume2, *volumes[1])
}

func TestGetAllOsAgentVolumes(t *testing.T) {
	testDynakube1 := createTestDynakube(1)
	testDynakube2 := createTestDynakube(2)

	now := time.Now()
	osVolume1 := OsAgentVolume{
		VolumeID:     "vol-1",
		TenantUUID:   testDynakube1.TenantUUID,
		Mounted:      true,
		LastModified: &now,
	}
	osVolume2 := OsAgentVolume{
		VolumeID:     "vol-2",
		TenantUUID:   testDynakube2.TenantUUID,
		Mounted:      true,
		LastModified: &now,
	}
	db := FakeMemoryDB()
	err := db.InsertOsAgentVolume(&osVolume1)
	require.NoError(t, err)
	err = db.InsertOsAgentVolume(&osVolume2)
	require.NoError(t, err)

	osVolumes, err := db.GetAllOsAgentVolumes()
	require.NoError(t, err)
	assert.Equal(t, 2, len(osVolumes))
}

func TestDeleteDynakube(t *testing.T) {
	testDynakube1 := createTestDynakube(1)
	testDynakube2 := createTestDynakube(2)

	db := FakeMemoryDB()
	err := db.InsertDynakube(&testDynakube1)
	require.NoError(t, err)
	err = db.InsertDynakube(&testDynakube2)
	require.NoError(t, err)

	err = db.DeleteDynakube(testDynakube1.Name)
	require.NoError(t, err)
	dynakubes, err := db.GetTenantsToDynakubes()
	require.NoError(t, err)
	assert.Equal(t, len(dynakubes), 1)
	assert.Equal(t, testDynakube2.TenantUUID, dynakubes[testDynakube2.Name])
}

func TestGetVolume_Empty(t *testing.T) {
	testVolume1 := createTestVolume(1)
	db := FakeMemoryDB()

	vo, err := db.GetVolume(testVolume1.PodName)
	require.NoError(t, err)
	assert.Nil(t, vo)
}

func TestInsertVolume(t *testing.T) {
	testVolume1 := createTestVolume(1)
	db := FakeMemoryDB()

	err := db.InsertVolume(&testVolume1)
	require.NoError(t, err)
	row := db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE ID = ?;", volumesTableName), testVolume1.VolumeID)
	var id string
	var puid string
	var ver string
	var tuid string
	var mountAttempts int
	err = row.Scan(&id, &puid, &ver, &tuid, &mountAttempts)

	require.NoError(t, err)
	assert.Equal(t, testVolume1.VolumeID, id)
	assert.Equal(t, testVolume1.PodName, puid)
	assert.Equal(t, testVolume1.Version, ver)
	assert.Equal(t, testVolume1.TenantUUID, tuid)
	assert.Equal(t, testVolume1.MountAttempts, mountAttempts)

	newPodName := "something-else"
	testVolume1.PodName = newPodName
	err = db.InsertVolume(&testVolume1)
	require.NoError(t, err)
	row = db.conn.QueryRow(fmt.Sprintf("SELECT * FROM %s WHERE ID = ?;", volumesTableName), testVolume1.VolumeID)
	err = row.Scan(&id, &puid, &ver, &tuid, &mountAttempts)

	require.NoError(t, err)
	assert.Equal(t, testVolume1.VolumeID, id)
	assert.Equal(t, testVolume1.PodName, puid)
	assert.Equal(t, testVolume1.Version, ver)
	assert.Equal(t, testVolume1.TenantUUID, tuid)
	assert.Equal(t, testVolume1.MountAttempts, mountAttempts)
}

func TestInsertOsAgentVolume(t *testing.T) {
	testDynakube1 := createTestDynakube(1)
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
	require.NoError(t, err)
	assert.Equal(t, volumeID, volume.VolumeID)
	assert.Equal(t, tenantUUID, volume.TenantUUID)
	assert.Equal(t, mounted, volume.Mounted)
	assert.True(t, volume.LastModified.Equal(lastModified))
}

func TestGetOsAgentVolumeViaVolumeID(t *testing.T) {
	testDynakube1 := createTestDynakube(1)
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
	actual, err := db.GetOsAgentVolumeViaVolumeID(expected.VolumeID)
	require.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, expected.VolumeID, actual.VolumeID)
	assert.Equal(t, expected.TenantUUID, actual.TenantUUID)
	assert.Equal(t, expected.Mounted, actual.Mounted)
	assert.True(t, expected.LastModified.Equal(*actual.LastModified))
}

func TestGetOsAgentVolumeViaTennatUUID(t *testing.T) {
	testDynakube1 := createTestDynakube(1)
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
	actual, err := db.GetOsAgentVolumeViaTenantUUID(expected.TenantUUID)
	require.NoError(t, err)
	assert.Equal(t, expected.VolumeID, actual.VolumeID)
	assert.Equal(t, expected.TenantUUID, actual.TenantUUID)
	assert.Equal(t, expected.Mounted, actual.Mounted)
	assert.True(t, expected.LastModified.Equal(*actual.LastModified))
}

func TestUpdateOsAgentVolume(t *testing.T) {
	testDynakube1 := createTestDynakube(1)
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

	actual, err := db.GetOsAgentVolumeViaVolumeID(old.VolumeID)
	require.NoError(t, err)
	assert.Equal(t, old.VolumeID, actual.VolumeID)
	assert.Equal(t, old.TenantUUID, actual.TenantUUID)
	assert.NotEqual(t, old.Mounted, actual.Mounted)
	assert.True(t, old.LastModified.Equal(*actual.LastModified))
}

func TestGetVolume(t *testing.T) {
	testVolume1 := createTestVolume(1)
	db := FakeMemoryDB()
	err := db.InsertVolume(&testVolume1)
	require.NoError(t, err)

	volume, err := db.GetVolume(testVolume1.VolumeID)
	require.NoError(t, err)
	assert.Equal(t, testVolume1, *volume)
}

func TestUpdateVolume(t *testing.T) {
	testVolume1 := createTestVolume(1)
	db := FakeMemoryDB()
	err := db.InsertVolume(&testVolume1)

	require.NoError(t, err)

	testVolume1.PodName = "different pod name"
	testVolume1.Version = "new version"
	testVolume1.TenantUUID = "asdf-1234"
	testVolume1.MountAttempts = 10
	err = db.InsertVolume(&testVolume1)

	require.NoError(t, err)

	insertedVolume, err := db.GetVolume(testVolume1.VolumeID)

	assert.NoError(t, err)
	assert.Equal(t, testVolume1.VolumeID, insertedVolume.VolumeID)
	assert.Equal(t, testVolume1.PodName, insertedVolume.PodName)
	assert.Equal(t, testVolume1.Version, insertedVolume.Version)
	assert.Equal(t, testVolume1.TenantUUID, insertedVolume.TenantUUID)
	assert.Equal(t, testVolume1.MountAttempts, insertedVolume.MountAttempts)
}

func TestGetUsedVersions(t *testing.T) {
	testVolume1 := createTestVolume(1)
	db := FakeMemoryDB()
	err := db.InsertVolume(&testVolume1)
	testVolume11 := testVolume1
	testVolume11.VolumeID = "vol-11"
	testVolume11.Version = "321"
	require.NoError(t, err)
	err = db.InsertVolume(&testVolume11)
	require.NoError(t, err)

	versions, err := db.GetUsedVersions(testVolume1.TenantUUID)
	require.NoError(t, err)
	assert.Equal(t, len(versions), 2)
	assert.True(t, versions[testVolume1.Version])
	assert.True(t, versions[testVolume11.Version])
}

func TestGetAllUsedVersions(t *testing.T) {
	db := FakeMemoryDB()
	testVolume1 := createTestVolume(1)
	err := db.InsertVolume(&testVolume1)
	testVolume11 := testVolume1
	testVolume11.VolumeID = "vol-11"
	testVolume11.Version = "321"
	require.NoError(t, err)
	err = db.InsertVolume(&testVolume11)
	require.NoError(t, err)

	versions, err := db.GetAllUsedVersions()
	require.NoError(t, err)
	assert.Equal(t, len(versions), 2)
	assert.True(t, versions[testVolume1.Version])
	assert.True(t, versions[testVolume11.Version])
}

func TestGetUsedImageDigests(t *testing.T) {
	db := FakeMemoryDB()
	testDynakube1 := createTestDynakube(1)
	err := db.InsertDynakube(&testDynakube1)
	require.NoError(t, err)

	copyDynakube := testDynakube1
	copyDynakube.Name = "copy"
	err = db.InsertDynakube(&copyDynakube)
	require.NoError(t, err)

	testDynakube2 := createTestDynakube(2)
	err = db.InsertDynakube(&testDynakube2)
	require.NoError(t, err)

	digests, err := db.GetUsedImageDigests()
	require.NoError(t, err)
	assert.Equal(t, 2, len(digests))
	assert.True(t, digests[testDynakube1.ImageDigest])
	assert.True(t, digests[copyDynakube.ImageDigest])
	assert.True(t, digests[testDynakube2.ImageDigest])
}

func TestIsImageDigestUsed(t *testing.T) {
	db := FakeMemoryDB()

	isUsed, err := db.IsImageDigestUsed("test")
	require.NoError(t, err)
	require.False(t, isUsed)

	testDynakube1 := createTestDynakube(1)
	err = db.InsertDynakube(&testDynakube1)
	require.NoError(t, err)

	isUsed, err = db.IsImageDigestUsed(testDynakube1.ImageDigest)
	require.NoError(t, err)
	require.True(t, isUsed)
}

func TestGetPodNames(t *testing.T) {
	testVolume1 := createTestVolume(1)
	testVolume2 := createTestVolume(2)

	db := FakeMemoryDB()
	err := db.InsertVolume(&testVolume1)
	require.NoError(t, err)
	err = db.InsertVolume(&testVolume2)
	require.NoError(t, err)

	podNames, err := db.GetPodNames()
	require.NoError(t, err)
	assert.Equal(t, len(podNames), 2)
	assert.Equal(t, testVolume1.VolumeID, podNames[testVolume1.PodName])
	assert.Equal(t, testVolume2.VolumeID, podNames[testVolume2.PodName])
}

func TestDeleteVolume(t *testing.T) {
	testVolume1 := createTestVolume(1)
	testVolume2 := createTestVolume(2)

	db := FakeMemoryDB()
	err := db.InsertVolume(&testVolume1)
	require.NoError(t, err)
	err = db.InsertVolume(&testVolume2)
	require.NoError(t, err)

	err = db.DeleteVolume(testVolume2.VolumeID)
	require.NoError(t, err)
	podNames, err := db.GetPodNames()
	require.NoError(t, err)
	assert.Equal(t, len(podNames), 1)
	assert.Equal(t, testVolume1.VolumeID, podNames[testVolume1.PodName])
}
