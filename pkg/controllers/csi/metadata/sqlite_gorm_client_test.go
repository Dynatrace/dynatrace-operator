package metadata

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCreateTenantConfig(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	tenantConfig := &TenantConfig{
		Name:                        "somename",
		ConfigDirPath:               "somewhere",
		DownloadedCodeModuleVersion: "1.2.3",
		TenantUUID:                  "abc123",
	}

	err = db.CreateTenantConfig(context.Background(), tenantConfig)
	require.NoError(t, err)

	readTenantConfig := &TenantConfig{TenantUUID: "abc123"}
	db.db.WithContext(context.Background()).First(readTenantConfig)
	assert.Equal(t, readTenantConfig.UID, tenantConfig.UID)

	err = db.CreateTenantConfig(context.Background(), nil)
	require.Error(t, err)
}

func TestReadTenantConfig(t *testing.T) {
	db, err := setupDB()
	setupPostReconcileData(context.Background(), db)

	require.NoError(t, err)

	tc, err := db.ReadTenantConfig(context.Background(), TenantConfig{TenantUUID: "abc123"})
	require.NoError(t, err)

	assert.NotNil(t, tc)
	assert.Equal(t, "abc123", tc.TenantUUID)

	_, err = db.ReadTenantConfig(context.Background(), TenantConfig{})
	require.Error(t, err)

	_, err = db.ReadTenantConfig(context.Background(), TenantConfig{TenantUUID: "unknown"})
	require.Error(t, err)
}

func TestUpdateTenantConfig(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostReconcileData(context.Background(), db)

	tenantConfig, err := db.ReadTenantConfig(context.Background(), TenantConfig{TenantUUID: "abc123"})
	require.NoError(t, err)

	tenantConfig.DownloadedCodeModuleVersion = "2.3.4"
	err = db.UpdateTenantConfig(context.Background(), tenantConfig)
	require.NoError(t, err)

	readTenantConfig := &TenantConfig{TenantUUID: "abc123"}
	db.db.WithContext(context.Background()).First(readTenantConfig)
	assert.Equal(t, tenantConfig.UID, readTenantConfig.UID)
	assert.Equal(t, "2.3.4", readTenantConfig.DownloadedCodeModuleVersion)

	err = db.UpdateTenantConfig(context.Background(), nil)
	require.Error(t, err)

	err = db.UpdateTenantConfig(context.Background(), &TenantConfig{})
	require.Error(t, err)
}

func TestSoftDeleteTenantConfig(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

	tenantConfig, err := db.ReadTenantConfig(context.Background(), TenantConfig{TenantUUID: "abc123"})
	require.NoError(t, err)

	err = db.DeleteTenantConfig(context.Background(), tenantConfig, false)
	require.NoError(t, err)

	readTenantConfig := &TenantConfig{TenantUUID: "abc123"}
	db.db.WithContext(context.Background()).First(readTenantConfig)
	assert.Equal(t, int64(0), db.db.RowsAffected)

	err = db.DeleteTenantConfig(context.Background(), nil, false)
	require.Error(t, err)

	err = db.DeleteTenantConfig(context.Background(), &TenantConfig{}, false)
	require.Error(t, err)
}

func TestCreateCodeModule(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	codeModule := &CodeModule{
		Version:  "1.2.3",
		Location: "someplace",
	}
	err = db.CreateCodeModule(context.Background(), codeModule)
	require.NoError(t, err)

	readCodeModule := &CodeModule{Version: "1.2.3"}
	db.db.WithContext(context.Background()).First(readCodeModule)
	assert.Equal(t, "someplace", readCodeModule.Location)

	err = db.CreateCodeModule(context.Background(), nil)
	require.Error(t, err)

	err = db.CreateCodeModule(context.Background(), &CodeModule{
		Version: "1.2.3",
	})
	require.Error(t, err)
}

func TestReadCodeModule(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostReconcileData(context.Background(), db)

	codeModule, err := db.ReadCodeModule(context.Background(), CodeModule{Version: "1.2.3"})
	require.NoError(t, err)

	assert.NotNil(t, codeModule)
	assert.Equal(t, "someplace", codeModule.Location)

	_, err = db.ReadCodeModule(context.Background(), CodeModule{Version: ""})
	require.Error(t, err)

	_, err = db.ReadCodeModule(context.Background(), CodeModule{Version: "unknown"})
	require.Error(t, err)
}

func TestSoftDeleteCodeModule(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostReconcileData(context.Background(), db)

	codeModule, err := db.ReadCodeModule(context.Background(), CodeModule{Version: "1.2.3"})
	require.NoError(t, err)

	assert.NotNil(t, codeModule)
	assert.Equal(t, "someplace", codeModule.Location)

	err = db.DeleteCodeModule(context.Background(), codeModule)
	require.NoError(t, err)

	readCodeModule := CodeModule{Version: "1.2.3"}
	db.db.WithContext(context.Background()).First(readCodeModule)
	assert.Equal(t, int64(0), db.db.RowsAffected)

	err = db.DeleteCodeModule(context.Background(), nil)
	require.Error(t, err)

	err = db.DeleteCodeModule(context.Background(), &CodeModule{})
	require.Error(t, err)
}

func TestCreateOsMount(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostReconcileData(context.Background(), db)

	tenant, err := db.ReadTenantConfig(context.Background(), TenantConfig{TenantUUID: "abc123"})
	require.NoError(t, err)

	vm := VolumeMeta{
		ID:                "osmount1",
		PodUid:            "pod1",
		PodName:           "podi",
		PodNamespace:      "testnamespace",
		PodServiceAccount: "podsa",
	}

	osMount := OSMount{
		VolumeMeta:    vm,
		Location:      "somewhere",
		MountAttempts: 1,
		TenantUUID:    tenant.TenantUUID,
		TenantConfig:  *tenant,
	}

	err = db.CreateOSMount(context.Background(), &osMount)
	require.NoError(t, err)

	readOSMount := &OSMount{TenantUUID: "abc123"}
	db.db.WithContext(context.Background()).First(readOSMount)
	assert.Equal(t, "somewhere", readOSMount.Location)

	err = db.CreateOSMount(context.Background(), nil)
	require.Error(t, err)
}

func TestReadOSMount(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

	osMount, err := db.ReadOSMount(context.Background(), OSMount{TenantUUID: "abc123"})
	require.NoError(t, err)

	assert.NotNil(t, osMount)
	assert.Equal(t, "osmount1", osMount.VolumeMeta.ID)

	_, err = db.ReadOSMount(context.Background(), OSMount{TenantUUID: ""})
	require.Error(t, err)
	assert.Equal(t, err.Error(), "Can't query for empty OSMount")

	_, err = db.ReadOSMount(context.Background(), OSMount{TenantUUID: "unknown"})
	require.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestUpdateOsMount(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

	osMount, err := db.ReadOSMount(context.Background(), OSMount{TenantUUID: "abc123"})
	require.NoError(t, err)

	osMount.MountAttempts = 5

	err = db.UpdateOSMount(context.Background(), osMount)
	require.NoError(t, err)

	readOSMount := &OSMount{TenantUUID: "abc123"}
	db.db.WithContext(context.Background()).First(readOSMount)
	assert.Equal(t, int64(5), readOSMount.MountAttempts)

	err = db.UpdateOSMount(context.Background(), nil)
	require.Error(t, err)

	err = db.UpdateOSMount(context.Background(), &OSMount{})
	require.Error(t, err)
}

func TestRestoreOsMount(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

	baseOSMount := OSMount{TenantUUID: "abc123"}

	osMount, err := db.ReadOSMount(context.Background(), baseOSMount)
	require.NoError(t, err)

	err = db.DeleteOSMount(context.Background(), osMount)
	require.NoError(t, err)

	_, err = db.ReadOSMount(context.Background(), baseOSMount)
	require.Error(t, err)

	err = db.restoreOSMount(context.Background(), &baseOSMount)
	require.NoError(t, err)

	_, err = db.ReadOSMount(context.Background(), baseOSMount)
	require.NoError(t, err)
}

func TestCreateAppMount(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostReconcileData(context.Background(), db)

	tenantConfig, err := db.ReadTenantConfig(context.Background(), TenantConfig{TenantUUID: "abc123"})
	require.NoError(t, err)

	cm, err := db.ReadCodeModule(context.Background(), CodeModule{Version: tenantConfig.DownloadedCodeModuleVersion})
	require.NoError(t, err)

	vm := VolumeMeta{
		ID:                "appmount1",
		PodUid:            "pod111",
		PodName:           "podiv",
		PodNamespace:      "testnamespace",
		PodServiceAccount: "podsa",
	}
	appMount := &AppMount{
		VolumeMeta:    vm,
		Location:      "loc1",
		MountAttempts: 1,
		CodeModule:    *cm,
	}

	err = db.CreateAppMount(context.Background(), appMount)
	require.NoError(t, err)

	readAppMount := &AppMount{VolumeMetaID: "appmount1"}
	db.db.WithContext(context.Background()).First(readAppMount)
	assert.Equal(t, "loc1", readAppMount.Location)

	err = db.CreateAppMount(context.Background(), nil)
	require.Error(t, err)

	err = db.CreateAppMount(context.Background(), &AppMount{})
	require.Error(t, err)

	err = db.CreateAppMount(context.Background(), &AppMount{
		VolumeMeta: vm,
	})
	require.Error(t, err)

	err = db.CreateAppMount(context.Background(), &AppMount{
		VolumeMeta: vm,
		CodeModule: *cm,
	})
	require.Error(t, err)

	err = db.CreateAppMount(context.Background(), &AppMount{
		VolumeMeta: vm,
		CodeModule: *cm,
		Location:   "somewhere",
	})
	require.Error(t, err)
}

func TestReadAppMounts(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

	appMounts, err := db.ReadAppMounts(context.Background())
	require.NoError(t, err)

	assert.NotNil(t, appMounts)
	assert.NotEmpty(t, len(appMounts))
	assert.Equal(t, "appmount1", appMounts[0].VolumeMeta.ID)
}

func TestReadAppMount(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

	appMount, err := db.ReadAppMount(context.Background(), AppMount{VolumeMeta: VolumeMeta{ID: "appmount1"}})
	require.NoError(t, err)

	assert.NotNil(t, appMount)
	assert.Equal(t, "appmount1", appMount.VolumeMeta.ID)

	_, err = db.ReadAppMount(context.Background(), AppMount{VolumeMeta: VolumeMeta{ID: ""}})
	require.Error(t, err)

	_, err = db.ReadAppMount(context.Background(), AppMount{VolumeMetaID: "unknown", VolumeMeta: VolumeMeta{ID: "unknown"}})
	require.Error(t, err)
}

func TestUpdateAppMount(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

	appMount, err := db.ReadAppMount(context.Background(), AppMount{VolumeMeta: VolumeMeta{ID: "appmount1"}})
	require.NoError(t, err)

	appMount.MountAttempts = 5

	err = db.UpdateAppMount(context.Background(), appMount)
	require.NoError(t, err)

	readAppMount := &AppMount{VolumeMetaID: "appmount1"}
	db.db.WithContext(context.Background()).First(readAppMount)
	assert.Equal(t, int64(5), readAppMount.MountAttempts)

	err = db.UpdateAppMount(context.Background(), nil)
	require.Error(t, err)

	err = db.UpdateAppMount(context.Background(), &AppMount{})
	require.Error(t, err)
}

func TestSoftDeleteAppMount(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

	appMount, err := db.ReadAppMount(context.Background(), AppMount{VolumeMeta: VolumeMeta{ID: "appmount1"}})
	require.NoError(t, err)

	assert.NotNil(t, appMount)
	assert.Equal(t, "appmount1", appMount.VolumeMeta.ID)

	err = db.DeleteAppMount(context.Background(), appMount)
	require.NoError(t, err)

	readAppMount := &AppMount{VolumeMetaID: "appmount1"}
	db.db.WithContext(context.Background()).First(readAppMount)
	assert.Equal(t, int64(0), db.db.RowsAffected)

	err = db.DeleteAppMount(context.Background(), nil)
	require.Error(t, err)

	err = db.DeleteAppMount(context.Background(), &AppMount{})
	require.Error(t, err)
}
func TestVolumeMetaValidation(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostReconcileData(context.Background(), db)

	vm := &VolumeMeta{
		ID:                "appmount1",
		PodUid:            "pod111",
		PodName:           "podiv",
		PodNamespace:      "testnamespace",
		PodServiceAccount: "podsa",
	}
	db.db.Create(vm)

	vm2 := &VolumeMeta{
		ID:                "appmount2",
		PodName:           "podiv",
		PodNamespace:      "testnamespace",
		PodServiceAccount: "podsa",
	}
	db.db.Create(vm2)

	vm3 := &VolumeMeta{
		ID:                "appmount3",
		PodUid:            "pod111",
		PodNamespace:      "testnamespace",
		PodServiceAccount: "podsa",
	}
	db.db.Create(vm3)

	vm4 := &VolumeMeta{
		ID:                "appmount4",
		PodUid:            "pod111",
		PodName:           "podiv",
		PodServiceAccount: "podsa",
	}
	db.db.Create(vm4)

	vm5 := &VolumeMeta{
		ID:           "appmount5",
		PodUid:       "pod111",
		PodName:      "podiv",
		PodNamespace: "testnamespace",
	}
	db.db.Create(vm5)
}

func TestReadVolumeMeta(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

	appMount, err := db.ReadVolumeMeta(context.Background(), VolumeMeta{ID: "appmount1"})
	require.NoError(t, err)

	assert.NotNil(t, appMount)
	assert.Equal(t, "appmount1", appMount.ID)

	_, err = db.ReadVolumeMeta(context.Background(), VolumeMeta{ID: ""})
	require.Error(t, err)

	_, err = db.ReadVolumeMeta(context.Background(), VolumeMeta{ID: "unknown"})
	require.Error(t, err)
}

func setupDB() (*DBConn, error) {
	db, err := NewDBAccess("file:csi_testdb?mode=memory")
	if err != nil {
		return nil, err
	}

	err = db.InitGormSchema()

	if err != nil {
		return nil, err
	}

	return db, nil
}

func setupPostReconcileData(ctx context.Context, conn *DBConn) {
	ctxDB := conn.db.WithContext(ctx)

	tenantConfig := &TenantConfig{
		Name:                        "abc123",
		ConfigDirPath:               "somewhere",
		DownloadedCodeModuleVersion: "1.2.3",
		TenantUUID:                  "abc123",
	}
	ctxDB.Create(tenantConfig)

	codeModule := &CodeModule{
		Version:  "1.2.3",
		Location: "someplace",
	}
	ctxDB.Create(codeModule)
}

func setupPostPublishData(ctx context.Context, conn *DBConn) {
	ctxDB := conn.db.WithContext(ctx)
	tenantConfig := &TenantConfig{
		Name:                        "abc123",
		ConfigDirPath:               "somewhere",
		DownloadedCodeModuleVersion: "1.2.3",
		TenantUUID:                  "abc123",
	}
	ctxDB.Create(tenantConfig)

	codeModule := &CodeModule{
		Version:  "1.2.3",
		Location: "someplace",
	}
	ctxDB.Create(codeModule)

	vmOM := VolumeMeta{
		ID:                "osmount1",
		PodUid:            "pod1",
		PodName:           "podi",
		PodNamespace:      "testnamespace",
		PodServiceAccount: "podsa",
	}
	osMount := &OSMount{
		VolumeMeta:    vmOM,
		Location:      "somewhere",
		TenantUUID:    tenantConfig.TenantUUID,
		TenantConfig:  *tenantConfig,
		MountAttempts: 1,
	}
	ctxDB.Create(osMount)

	for i := range 3 {
		vmAP := VolumeMeta{
			ID:                fmt.Sprintf("appmount%d", i+1),
			PodUid:            fmt.Sprintf("pod%d", i+1),
			PodName:           fmt.Sprintf("podName%d", i+1),
			PodNamespace:      "testnamespace",
			PodServiceAccount: "podsa",
		}
		appMount := &AppMount{
			VolumeMeta:    vmAP,
			Location:      fmt.Sprintf("loc%d", i+1),
			MountAttempts: 1,
			CodeModule:    *codeModule,
		}
		ctxDB.Create(appMount)
	}
}
