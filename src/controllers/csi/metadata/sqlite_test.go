package metadata

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseLocked(t *testing.T) {
	dbPath := "test.sqlite"
	t.Run("1 writer 1 reader, same table", func(t *testing.T) {
		defer os.Remove(dbPath)
		var wait sync.WaitGroup

		db1 := createDB(t, dbPath)
		db2 := createDB(t, dbPath)

		startRoutine(&wait, writeALotOfDynakubesToDB, t, db1, 0, 3000)
		startRoutine(&wait, readALotOfDynakubesFromDB, t, db2, 5, 500)

		wait.Wait()
	})
	t.Run("2 writers different tables", func(t *testing.T) {
		defer os.Remove(dbPath)
		var wait sync.WaitGroup

		db1 := createDB(t, dbPath)
		db2 := createDB(t, dbPath)

		startRoutine(&wait, writeALotOfDynakubesToDB, t, db1, 0, 3000)
		startRoutine(&wait, writeALotOfVolumesToDB, t, db2, 0, 3000)

		wait.Wait()
	})
	t.Run("2 writers same table", func(t *testing.T) {
		defer os.Remove(dbPath)
		var wait sync.WaitGroup

		db1 := createDB(t, dbPath)
		db2 := createDB(t, dbPath)

		wait.Add(1)
		go func() {
			defer wait.Done()
			writeALotOfDynakubesToDB(t, db1, 0, 3000)
		}()

		wait.Add(1)
		go func() {
			defer wait.Done()
			writeALotOfDynakubesToDB(t, db2, 3001, 5000)
		}()

		wait.Wait()
	})

	t.Run("5 writers same table", func(t *testing.T) {
		defer os.Remove(dbPath)
		var wait sync.WaitGroup

		startRoutineXTimes(5, &wait, writeALotOfVolumesToDB, t, dbPath, 0, 5000)

		wait.Wait()
	})

	t.Run("10 writers same table", func(t *testing.T) {
		defer os.Remove(dbPath)
		var wait sync.WaitGroup

		startRoutineXTimes(10, &wait, writeALotOfVolumesToDB, t, dbPath, 0, 8000)

		wait.Wait()
	})

	t.Run("5-5 writers different table", func(t *testing.T) {
		defer os.Remove(dbPath)
		var wait sync.WaitGroup

		go startRoutineXTimes(5, &wait, writeALotOfVolumesToDB, t, dbPath, 0, 4000)
		go startRoutineXTimes(5, &wait, writeALotOfDynakubesToDB, t, dbPath, 0, 4000)

		wait.Wait()
	})

	t.Run("2-2 2-2 readers-writers 2 table", func(t *testing.T) {
		defer os.Remove(dbPath)
		var wait sync.WaitGroup

		go startRoutineXTimes(2, &wait, writeALotOfVolumesToDB, t, dbPath, 0, 8000)
		go startRoutineXTimes(2, &wait, writeALotOfDynakubesToDB, t, dbPath, 0, 8000)
		go startRoutineXTimes(2, &wait, readALotOfVolumesFromDB, t, dbPath, 0, 8000)
		go startRoutineXTimes(2, &wait, readALotOfDynakubesFromDB, t, dbPath, 0, 8000)

		wait.Wait()
	})

	t.Run("x writers same table", func(t *testing.T) {
		defer os.Remove(dbPath)
		var wait sync.WaitGroup

		x := 60
		end := x * 10

		startRoutineXTimes(x, &wait, writeALotOfVolumesToDB, t, dbPath, 0, end)

		wait.Wait()

		checkVolumesFromDB(t, createDB(t, dbPath), 6, end)
	})

	t.Run("x writers same table, shared db object", func(t *testing.T) {
		defer os.Remove(dbPath)
		var wait sync.WaitGroup

		x := 10
		end := x * 10000

		startRoutineXTimesSharedDB(x, &wait, writeALotOfVolumesToDB, t, dbPath, 0, end)

		wait.Wait()
	})

	t.Run("x writers same table, error handling", func(t *testing.T) {
		defer os.Remove(dbPath)
		var wait sync.WaitGroup

		x := 1000
		end := x * 10

		startRoutineXTimes(x, &wait, writeALotOfVolumesToDBWithErrorHandling, t, dbPath, 0, end)

		wait.Wait()

		checkVolumesFromDB(t, createDB(t, dbPath), 0, end)
	})

	t.Run("only readers and deletes", func(t *testing.T) {
		defer os.Remove(dbPath)
		var wait sync.WaitGroup

		db := createDB(t, dbPath)
		//prep
		writeALotOfVolumesToDB(t, db, 0, 10000)


		go startRoutineXTimes(30, &wait, readALotOfVolumesFromDB, t, dbPath, 0, 5000)
		go startRoutineXTimes(30, &wait, deleteALotOfVolumesFromDB, t, dbPath, 5001, 10000)

		wait.Wait()

	})
}

func startRoutineXTimes(x int, wait *sync.WaitGroup, fun func(t *testing.T, db Access, start, end int), t *testing.T, dbPath string, start, end int) {
	var dbs []Access
	for i := 0; i <= x; i++ {
		dbs = append(dbs, createDB(t, dbPath))
	}
	jump := end / x
	j := 0
	for i := start; i <= end; i = i + jump + 1 {
		startRoutine(wait, fun, t, dbs[j], i, i+jump)
		j++
	}
}

func startRoutineXTimesSharedDB(x int, wait *sync.WaitGroup, fun func(t *testing.T, db Access, start, end int), t *testing.T, dbPath string, start, end int) {
	db := createDB(t, dbPath)
	jump := end / x
	j := 0
	for i := start; i <= end; i = i + jump + 1 {
		startRoutine(wait, fun, t, db, i, i+jump)
		j++
	}
}

func startRoutine(wait *sync.WaitGroup, fun func(t *testing.T, db Access, start, end int), t *testing.T, db Access, start, end int) {
	wait.Add(1)
	go func() {
		defer wait.Done()
		t.Logf("started routine from %d to %d", start, end)
		fun(t, db, start, end)
		t.Logf("ended routine from %d to %d", start, end)
	}()
}

func readALotOfDynakubesFromDB(t *testing.T, db Access, start, end int) {
	ctx := context.TODO()
	for i := start; i <= end; i++ {
		t.Logf("%s dk-reader-%d: reading %d", time.Now(), start, i)
		_, err := db.GetDynakube(ctx, generateTestDynakube(i).Name)
		if err != nil {
			t.Logf("failed to read dynakube %d, because %s", i, err.Error())
		}
		require.NoError(t, err)
	}
}

func readALotOfVolumesFromDB(t *testing.T, db Access, start, end int) {
	ctx := context.TODO()
	for i := start; i <= end; i++ {
		t.Logf("%s vl-reader-%d: reading %d", time.Now(), start, i)
		_, err := db.GetVolume(ctx, generateTestVolume(i).VolumeID)
		if err != nil {
			t.Logf("failed to read volume %d, because %s", i, err.Error())
		}
		require.NoError(t, err)
	}
}

func deleteALotOfVolumesFromDB(t *testing.T, db Access, start, end int) {
	ctx := context.TODO()
	for i := start; i <= end; i++ {
		t.Logf("%s vl-reader-%d: reading %d", time.Now(), start, i)
		_, err := db.GetVolume(ctx, generateTestVolume(i).VolumeID)
		if err != nil {
			t.Logf("failed to delete volume %d, because %s", i, err.Error())
		}
		require.NoError(t, err)
	}
}

func checkVolumesFromDB(t *testing.T, db Access, start, end int) {
	ctx := context.TODO()
	for i := start; i <= end; i++ {
		volume, err := db.GetVolume(ctx, generateTestVolume(i).VolumeID)
		if err != nil {
			t.Logf("failed to read volume %d, because %s", i, err.Error())
		}
		if volume == nil {
			t.Logf("missing volume %d", i)
		}
		require.NotNil(t, volume)
		require.Equal(t, *volume, *generateTestVolume(i))
	}
}

func writeALotOfDynakubesToDB(t *testing.T, db Access, start, end int) {
	ctx := context.TODO()
	for i := start; i <= end; i++ {
		t.Logf("%s dk-writer-%d: writing %d", time.Now(), start, i)
		err := db.InsertDynakube(ctx, generateTestDynakube(i))
		if err != nil {
			t.Logf("failed to write dynakube %d, because %s", i, err.Error())
		}
		require.NoError(t, err)
	}
}

func writeALotOfVolumesToDB(t *testing.T, db Access, start, end int) {
	ctx := context.TODO()
	for i := start; i <= end; i++ {
		t.Logf("%s vl-writer-%d: writing %d", time.Now(), start, i)
		err := db.InsertVolume(ctx, generateTestVolume(i))
		if err != nil {
			t.Logf("failed to write volume %d, because %s", i, err.Error())
		}
		require.NoError(t, err)
	}
}

func writeALotOfVolumesToDBWithErrorHandling(t *testing.T, db Access, start, end int) {
	ctx := context.TODO()
	for i := start; i <= end; i++ {
		t.Logf("%s vl-writer-%d: writing %d", time.Now(), start, i)
		err := db.InsertVolume(ctx, generateTestVolume(i))
		if err != nil {
			t.Logf("failed to write volume %d, because %s", i, err.Error())
			for err != nil && strings.Contains(err.Error(), "database is locked") {
				time.Sleep(time.Duration(rand.Intn(10)))
				err = db.InsertVolume(ctx, generateTestVolume(i))
			}
		}
		require.NoError(t, err)
	}
}

func createDB(t *testing.T, dbPath string) Access {
	ctx := context.TODO()
	db, err := NewAccess(ctx, dbPath)
	if err != nil {
		require.NoError(t, err)
	}
	return db
}

func generateTestDynakube(i int) *Dynakube {
	return &Dynakube{
		Name:                   fmt.Sprintf("dynakube-%d", i),
		TenantUUID:             fmt.Sprintf("tenant-%d", i),
		LatestVersion:          fmt.Sprintf("version-%d", i),
		ImageDigest:            fmt.Sprintf("digest-%d", i),
		MaxFailedMountAttempts: i,
	}
}

func generateTestVolume(i int) *Volume {
	return &Volume{
		VolumeID:      fmt.Sprintf("volume-%d", i),
		PodName:       fmt.Sprintf("pod-%d", i),
		Version:       fmt.Sprintf("version-%d", i),
		TenantUUID:    fmt.Sprintf("tenant-%d", i),
		MountAttempts: i,
	}
}

func TestNewAccess(t *testing.T) {
	db, err := NewAccess(context.TODO(), ":memory:")
	require.NoError(t, err)
	assert.NotNil(t, db.(*SqliteAccess).conn)
}

func TestSetup(t *testing.T) {
	db := SqliteAccess{}
	err := db.Setup(context.TODO(), ":memory:")

	require.NoError(t, err)
	assert.True(t, checkIfTablesExist(&db))
}

func TestSetup_badPath(t *testing.T) {
	db := SqliteAccess{}
	err := db.Setup(context.TODO(), "/asd")
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
	ctx := context.TODO()
	t.Run("volume table is created correctly", func(t *testing.T) {
		db := emptyMemoryDB()

		err := db.createTables(ctx)
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

		err := db.createTables(ctx)
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
				assert.Equal(t, fmt.Sprint(dynatracev1beta1.DefaultMaxFailedCsiMountAttempts), *defaultValue)
				assert.Equal(t, dynatracev1beta1.DefaultMaxFailedCsiMountAttempts, maxFailedMountAttempts)
				assert.Equal(t, "1", notNull)
			}
		}
	})
}

func TestInsertDynakube(t *testing.T) {
	testDynakube1 := createTestDynakube(1)

	db := FakeMemoryDB()

	err := db.InsertDynakube(context.TODO(), &testDynakube1)
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

	gt, err := db.GetDynakube(context.TODO(), testDynakube1.TenantUUID)
	require.NoError(t, err)
	assert.Nil(t, gt)
}

func TestGetDynakube(t *testing.T) {
	ctx := context.TODO()
	t.Run("get dynakube", func(t *testing.T) {
		testDynakube1 := createTestDynakube(1)
		db := FakeMemoryDB()
		err := db.InsertDynakube(ctx, &testDynakube1)
		require.NoError(t, err)

		dynakube, err := db.GetDynakube(ctx, testDynakube1.Name)
		require.NoError(t, err)
		assert.Equal(t, testDynakube1, *dynakube)
	})
}

func TestUpdateDynakube(t *testing.T) {
	ctx := context.TODO()
	testDynakube1 := createTestDynakube(1)
	db := FakeMemoryDB()
	err := db.InsertDynakube(ctx, &testDynakube1)
	require.NoError(t, err)

	copyDynakube := testDynakube1
	copyDynakube.LatestVersion = "132.546"
	copyDynakube.ImageDigest = ""
	copyDynakube.MaxFailedMountAttempts = 10
	err = db.UpdateDynakube(ctx, &copyDynakube)
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
	ctx := context.TODO()
	testDynakube1 := createTestDynakube(1)
	testDynakube2 := createTestDynakube(2)

	db := FakeMemoryDB()
	err := db.InsertDynakube(ctx, &testDynakube1)
	require.NoError(t, err)
	err = db.InsertDynakube(ctx, &testDynakube2)
	require.NoError(t, err)

	dynakubes, err := db.GetTenantsToDynakubes(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(dynakubes))
	assert.Equal(t, testDynakube1.TenantUUID, dynakubes[testDynakube1.Name])
	assert.Equal(t, testDynakube2.TenantUUID, dynakubes[testDynakube2.Name])
}

func TestGetAllDynakubes(t *testing.T) {
	ctx := context.TODO()
	t.Run("get multiple dynakubes", func(t *testing.T) {
		testDynakube1 := createTestDynakube(1)
		testDynakube2 := createTestDynakube(2)

		db := FakeMemoryDB()
		err := db.InsertDynakube(ctx, &testDynakube1)
		require.NoError(t, err)
		err = db.InsertDynakube(ctx, &testDynakube2)
		require.NoError(t, err)

		dynakubes, err := db.GetAllDynakubes(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, len(dynakubes))
	})
}

func TestGetAllVolumes(t *testing.T) {
	ctx := context.TODO()
	testVolume1 := createTestVolume(1)
	testVolume2 := createTestVolume(2)

	db := FakeMemoryDB()
	err := db.InsertVolume(ctx, &testVolume1)
	require.NoError(t, err)
	err = db.InsertVolume(ctx, &testVolume2)
	require.NoError(t, err)

	volumes, err := db.GetAllVolumes(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(volumes))
	assert.Equal(t, testVolume1, *volumes[0])
	assert.Equal(t, testVolume2, *volumes[1])
}

func TestGetAllOsAgentVolumes(t *testing.T) {
	ctx := context.TODO()
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
	err := db.InsertOsAgentVolume(ctx, &osVolume1)
	require.NoError(t, err)
	err = db.InsertOsAgentVolume(ctx, &osVolume2)
	require.NoError(t, err)

	osVolumes, err := db.GetAllOsAgentVolumes(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(osVolumes))
}

func TestDeleteDynakube(t *testing.T) {
	ctx := context.TODO()
	testDynakube1 := createTestDynakube(1)
	testDynakube2 := createTestDynakube(2)

	db := FakeMemoryDB()
	err := db.InsertDynakube(ctx, &testDynakube1)
	require.NoError(t, err)
	err = db.InsertDynakube(ctx, &testDynakube2)
	require.NoError(t, err)

	err = db.DeleteDynakube(ctx, testDynakube1.Name)
	require.NoError(t, err)
	dynakubes, err := db.GetTenantsToDynakubes(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(dynakubes), 1)
	assert.Equal(t, testDynakube2.TenantUUID, dynakubes[testDynakube2.Name])
}

func TestGetVolume_Empty(t *testing.T) {
	ctx := context.TODO()
	testVolume1 := createTestVolume(1)
	db := FakeMemoryDB()

	vo, err := db.GetVolume(ctx, testVolume1.PodName)
	require.NoError(t, err)
	assert.Nil(t, vo)
}

func TestInsertVolume(t *testing.T) {
	ctx := context.TODO()
	testVolume1 := createTestVolume(1)
	db := FakeMemoryDB()

	err := db.InsertVolume(ctx, &testVolume1)
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
	err = db.InsertVolume(ctx, &testVolume1)
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

	err := db.InsertOsAgentVolume(context.TODO(), &volume)
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
	ctx := context.TODO()
	testDynakube1 := createTestDynakube(1)
	db := FakeMemoryDB()

	now := time.Now()
	expected := OsAgentVolume{
		VolumeID:     "vol-4",
		TenantUUID:   testDynakube1.TenantUUID,
		Mounted:      true,
		LastModified: &now,
	}

	err := db.InsertOsAgentVolume(ctx, &expected)
	require.NoError(t, err)
	actual, err := db.GetOsAgentVolumeViaVolumeID(ctx, expected.VolumeID)
	require.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, expected.VolumeID, actual.VolumeID)
	assert.Equal(t, expected.TenantUUID, actual.TenantUUID)
	assert.Equal(t, expected.Mounted, actual.Mounted)
	assert.True(t, expected.LastModified.Equal(*actual.LastModified))
}

func TestGetOsAgentVolumeViaTennatUUID(t *testing.T) {
	ctx := context.TODO()
	testDynakube1 := createTestDynakube(1)
	db := FakeMemoryDB()

	now := time.Now()
	expected := OsAgentVolume{
		VolumeID:     "vol-4",
		TenantUUID:   testDynakube1.TenantUUID,
		Mounted:      true,
		LastModified: &now,
	}

	err := db.InsertOsAgentVolume(ctx, &expected)
	require.NoError(t, err)
	actual, err := db.GetOsAgentVolumeViaTenantUUID(ctx, expected.TenantUUID)
	require.NoError(t, err)
	assert.Equal(t, expected.VolumeID, actual.VolumeID)
	assert.Equal(t, expected.TenantUUID, actual.TenantUUID)
	assert.Equal(t, expected.Mounted, actual.Mounted)
	assert.True(t, expected.LastModified.Equal(*actual.LastModified))
}

func TestUpdateOsAgentVolume(t *testing.T) {
	ctx := context.TODO()
	testDynakube1 := createTestDynakube(1)
	db := FakeMemoryDB()

	now := time.Now()
	old := OsAgentVolume{
		VolumeID:     "vol-4",
		TenantUUID:   testDynakube1.TenantUUID,
		Mounted:      true,
		LastModified: &now,
	}

	err := db.InsertOsAgentVolume(ctx, &old)
	require.NoError(t, err)
	new := old
	new.Mounted = false
	err = db.UpdateOsAgentVolume(ctx, &new)
	require.NoError(t, err)

	actual, err := db.GetOsAgentVolumeViaVolumeID(ctx, old.VolumeID)
	require.NoError(t, err)
	assert.Equal(t, old.VolumeID, actual.VolumeID)
	assert.Equal(t, old.TenantUUID, actual.TenantUUID)
	assert.NotEqual(t, old.Mounted, actual.Mounted)
	assert.True(t, old.LastModified.Equal(*actual.LastModified))
}

func TestGetVolume(t *testing.T) {
	ctx := context.TODO()
	testVolume1 := createTestVolume(1)
	db := FakeMemoryDB()
	err := db.InsertVolume(ctx, &testVolume1)
	require.NoError(t, err)

	volume, err := db.GetVolume(ctx, testVolume1.VolumeID)
	require.NoError(t, err)
	assert.Equal(t, testVolume1, *volume)
}

func TestUpdateVolume(t *testing.T) {
	ctx := context.TODO()
	testVolume1 := createTestVolume(1)
	db := FakeMemoryDB()
	err := db.InsertVolume(ctx, &testVolume1)

	require.NoError(t, err)

	testVolume1.PodName = "different pod name"
	testVolume1.Version = "new version"
	testVolume1.TenantUUID = "asdf-1234"
	testVolume1.MountAttempts = 10
	err = db.InsertVolume(ctx, &testVolume1)

	require.NoError(t, err)

	insertedVolume, err := db.GetVolume(ctx, testVolume1.VolumeID)

	assert.NoError(t, err)
	assert.Equal(t, testVolume1.VolumeID, insertedVolume.VolumeID)
	assert.Equal(t, testVolume1.PodName, insertedVolume.PodName)
	assert.Equal(t, testVolume1.Version, insertedVolume.Version)
	assert.Equal(t, testVolume1.TenantUUID, insertedVolume.TenantUUID)
	assert.Equal(t, testVolume1.MountAttempts, insertedVolume.MountAttempts)
}

func TestGetUsedVersions(t *testing.T) {
	ctx := context.TODO()
	testVolume1 := createTestVolume(1)
	db := FakeMemoryDB()
	err := db.InsertVolume(ctx, &testVolume1)
	testVolume11 := testVolume1
	testVolume11.VolumeID = "vol-11"
	testVolume11.Version = "321"
	require.NoError(t, err)
	err = db.InsertVolume(ctx, &testVolume11)
	require.NoError(t, err)

	versions, err := db.GetUsedVersions(ctx, testVolume1.TenantUUID)
	require.NoError(t, err)
	assert.Equal(t, len(versions), 2)
	assert.True(t, versions[testVolume1.Version])
	assert.True(t, versions[testVolume11.Version])
}

func TestGetAllUsedVersions(t *testing.T) {
	ctx := context.TODO()
	db := FakeMemoryDB()
	testVolume1 := createTestVolume(1)
	err := db.InsertVolume(ctx, &testVolume1)
	testVolume11 := testVolume1
	testVolume11.VolumeID = "vol-11"
	testVolume11.Version = "321"
	require.NoError(t, err)
	err = db.InsertVolume(ctx, &testVolume11)
	require.NoError(t, err)

	versions, err := db.GetAllUsedVersions(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(versions), 2)
	assert.True(t, versions[testVolume1.Version])
	assert.True(t, versions[testVolume11.Version])
}

func TestGetUsedImageDigests(t *testing.T) {
	ctx := context.TODO()
	db := FakeMemoryDB()
	testDynakube1 := createTestDynakube(1)
	err := db.InsertDynakube(ctx, &testDynakube1)
	require.NoError(t, err)

	copyDynakube := testDynakube1
	copyDynakube.Name = "copy"
	err = db.InsertDynakube(ctx, &copyDynakube)
	require.NoError(t, err)

	testDynakube2 := createTestDynakube(2)
	err = db.InsertDynakube(ctx, &testDynakube2)
	require.NoError(t, err)

	digests, err := db.GetUsedImageDigests(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(digests))
	assert.True(t, digests[testDynakube1.ImageDigest])
	assert.True(t, digests[copyDynakube.ImageDigest])
	assert.True(t, digests[testDynakube2.ImageDigest])
}

func TestIsImageDigestUsed(t *testing.T) {
	ctx := context.TODO()
	db := FakeMemoryDB()

	isUsed, err := db.IsImageDigestUsed(ctx, "test")
	require.NoError(t, err)
	require.False(t, isUsed)

	testDynakube1 := createTestDynakube(1)
	err = db.InsertDynakube(ctx, &testDynakube1)
	require.NoError(t, err)

	isUsed, err = db.IsImageDigestUsed(ctx, testDynakube1.ImageDigest)
	require.NoError(t, err)
	require.True(t, isUsed)
}

func TestGetPodNames(t *testing.T) {
	ctx := context.TODO()
	testVolume1 := createTestVolume(1)
	testVolume2 := createTestVolume(2)

	db := FakeMemoryDB()
	err := db.InsertVolume(ctx, &testVolume1)
	require.NoError(t, err)
	err = db.InsertVolume(ctx, &testVolume2)
	require.NoError(t, err)

	podNames, err := db.GetPodNames(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(podNames), 2)
	assert.Equal(t, testVolume1.VolumeID, podNames[testVolume1.PodName])
	assert.Equal(t, testVolume2.VolumeID, podNames[testVolume2.PodName])
}

func TestDeleteVolume(t *testing.T) {
	ctx := context.TODO()
	testVolume1 := createTestVolume(1)
	testVolume2 := createTestVolume(2)

	db := FakeMemoryDB()
	err := db.InsertVolume(ctx, &testVolume1)
	require.NoError(t, err)
	err = db.InsertVolume(ctx, &testVolume2)
	require.NoError(t, err)

	err = db.DeleteVolume(ctx, testVolume2.VolumeID)
	require.NoError(t, err)
	podNames, err := db.GetPodNames(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(podNames), 1)
	assert.Equal(t, testVolume1.VolumeID, podNames[testVolume1.PodName])
}
