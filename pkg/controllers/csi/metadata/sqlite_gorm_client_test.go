package metadata

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	MEM            = ":memory:"
	TMP            = "test.db"
	POST_RECONCILE = "reconcile.db"
	POST_PUBLISH   = "publish.db"
)

func createTestTenant() *TenantConfig {
	return &TenantConfig{
		Name:                        "somename",
		ConfigDirPath:               "somewhere",
		DownloadedCodeModuleVersion: "1.2.3",
		TenantUUID:                  "abc123",
	}
}

func TestCreateTenantConfig(t *testing.T) {
	db, err := setupSqLiteDB(MEM)
	// db, err := NewDBAccess(POST_RECONCILE) //todo
	require.NoError(t, err)

	err = db.SchemaMigration(context.Background())
	require.NoError(t, err)

	tc := createTestTenant()
	err = db.CreateTenantConfig(context.Background(), tc)
	require.NoError(t, err)

	var tcs []TenantConfig

	db.db.Raw(`SELECT * FROM tenant_configs WHERE tenant_uuid = "abc123"`).Scan(&tcs)
	assert.NotEmpty(t, tcs)
	assert.Equal(t, "somename", tcs[0].Name)
}

func TestReadTenantConfig(t *testing.T) {
	db, err := setupSqLiteDB(POST_RECONCILE)
	require.NoError(t, err)

	tc, err := db.ReadTenantConfigByTenantUUID(context.Background(), "abc123")
	require.NoError(t, err)

	assert.NotNil(t, tc)
	assert.Equal(t, "abc123", tc.TenantUUID)

	_, err = db.ReadTenantConfigByTenantUUID(context.Background(), "")
	require.Error(t, err)

	_, err = db.ReadTenantConfigByTenantUUID(context.Background(), "unknown")
	require.Error(t, err)
}

func TestUpdateTenantConfig(t *testing.T) {
	db, err := setupSqLiteDB(POST_PUBLISH)
	require.NoError(t, err)

	tenantConfig, err := db.ReadTenantConfigByTenantUUID(context.Background(), "abc123")
	require.NoError(t, err)
	tenantConfig.DownloadedCodeModuleVersion = "2.3.4"
	err = db.UpdateTenantConfig(context.Background(), tenantConfig)
	require.NoError(t, err)

	var tenantConfigs []TenantConfig

	db.db.Raw(`SELECT * FROM tenant_configs WHERE tenant_uuid = "abc123"`).Scan(&tenantConfigs)
	assert.NotEmpty(t, tenantConfigs)
	assert.Equal(t, "2.3.4", tenantConfigs[0].DownloadedCodeModuleVersion)
}

func TestSoftDeleteTenantConfig(t *testing.T) {
	db, err := setupSqLiteDB(POST_PUBLISH)
	require.NoError(t, err)

	tenantConfig, err := db.ReadTenantConfigByTenantUUID(context.Background(), "abc123")
	require.NoError(t, err)

	err = db.DeleteTenantConfig(context.Background(), tenantConfig)
	require.NoError(t, err)

	var result []TenantConfig

	db.db.Raw(`SELECT * FROM tenant_configs WHERE tenant_uuid = "abc123"`).Scan(&result)
	assert.NotEmpty(t, result)
	assert.True(t, result[0].DeletedAt.Valid)
}

func TestCreateCodeModule(t *testing.T) {
	db, err := setupSqLiteDB(MEM)
	// db, err := NewDBAccess(POST_RECONCILE) //todo
	require.NoError(t, err)

	err = db.SchemaMigration(context.Background())
	require.NoError(t, err)

	codeModule := &CodeModule{
		Version:  "1.2.3",
		Location: "someplace",
	}
	err = db.CreateCodeModule(context.Background(), codeModule)
	require.NoError(t, err)

	var cms []CodeModule

	db.db.Raw(`SELECT * FROM code_modules WHERE version = "1.2.3"`).Scan(&cms)
	assert.NotEmpty(t, cms)
	assert.Equal(t, "someplace", cms[0].Location)
}

func TestReadCodeModule(t *testing.T) {
	db, err := setupSqLiteDB(POST_RECONCILE)
	require.NoError(t, err)

	codeModule, err := db.ReadCodeModuleByVersion(context.Background(), "1.2.3")
	require.NoError(t, err)

	assert.NotNil(t, codeModule)
	assert.Equal(t, "someplace", codeModule.Location)

	_, err = db.ReadCodeModuleByVersion(context.Background(), "")
	require.Error(t, err)

	_, err = db.ReadCodeModuleByVersion(context.Background(), "unknown")
	require.Error(t, err)

	codeModule, err = db.ReadCodeModuleByTenantUUID(context.Background(), "abc123")
	require.NoError(t, err)
	assert.Equal(t, "someplace", codeModule.Location)
}

func TestSoftDeleteCodeModule(t *testing.T) {
	db, err := setupSqLiteDB(POST_PUBLISH)
	require.NoError(t, err)

	codeModule, err := db.ReadCodeModuleByVersion(context.Background(), "1.2.3")
	require.NoError(t, err)

	assert.NotNil(t, codeModule)
	assert.Equal(t, "someplace", codeModule.Location)

	err = db.DeleteCodeModule(context.Background(), codeModule)
	require.NoError(t, err)

	var result []CodeModule

	db.db.Raw(`SELECT * FROM code_modules WHERE version = "1.2.3"`).Scan(&result)
	assert.NotEmpty(t, result)
	assert.True(t, result[0].DeletedAt.Valid)
}

func TestCreateOsMount(t *testing.T) {
	db, err := setupSqLiteDB(POST_RECONCILE)
	// db, err := NewDBAccess(TMP) //todo
	require.NoError(t, err)

	tenant, err := db.ReadTenantConfigByTenantUUID(context.Background(), "abc123")
	require.NoError(t, err)

	osMount := OSMount{
		VolumeMeta: VolumeMeta{
			ID:                "osmount1",
			PodUid:            "pod1",
			PodName:           "podi",
			PodNamespace:      "testnamespace",
			PodServiceAccount: "podsa",
		},
		Location:        "somewhere",
		MountAttempts:   0,
		TenantUUID:      tenant.TenantUUID,
		TenantConfigUID: tenant.UID,
	}
	err = db.CreateOSMount(context.Background(), &osMount)
	require.NoError(t, err)

	var oms []OSMount

	db.db.Raw(`SELECT * FROM os_mounts WHERE tenant_uuid = "abc123"`).Scan(&oms)
	assert.NotEmpty(t, oms)
	assert.Equal(t, "somewhere", oms[0].Location)
}

func TestReadOSMount(t *testing.T) {
	db, err := setupSqLiteDB(POST_PUBLISH)
	require.NoError(t, err)

	appMount, err := db.ReadOSMountByTenantUUID(context.Background(), "abc123")
	require.NoError(t, err)

	assert.NotNil(t, appMount)
	assert.Equal(t, "osmount1", appMount.VolumeMeta.ID)

	_, err = db.ReadOSMountByTenantUUID(context.Background(), "")
	require.Error(t, err)

	_, err = db.ReadOSMountByTenantUUID(context.Background(), "unknown")
	require.Error(t, err)
}

func TestUpdateOsMount(t *testing.T) {
	db, err := setupSqLiteDB(POST_PUBLISH)
	require.NoError(t, err)

	osMount, err := db.ReadOSMountByTenantUUID(context.Background(), "abc123")
	require.NoError(t, err)

	osMount.MountAttempts = 5

	err = db.UpdateOSMount(context.Background(), osMount)
	require.NoError(t, err)

	var result []OSMount

	db.db.Raw(`SELECT * FROM os_mounts WHERE tenant_uuid = "abc123"`).Scan(&result)
	assert.NotEmpty(t, result)
	assert.Equal(t, int64(5), result[0].MountAttempts)
}

func TestSoftDeleteOSMount(t *testing.T) {
	db, err := setupSqLiteDB(POST_PUBLISH)
	require.NoError(t, err)

	osMount, err := db.ReadOSMountByTenantUUID(context.Background(), "abc123")
	require.NoError(t, err)

	assert.NotNil(t, osMount)
	assert.Equal(t, "osmount1", osMount.VolumeMeta.ID)

	err = db.DeleteOSMount(context.Background(), osMount)
	require.NoError(t, err)

	var result []OSMount

	db.db.Raw(`SELECT * FROM os_mounts WHERE tenant_uuid = "abc123"`).Scan(&result)
	assert.NotEmpty(t, result)
	assert.True(t, result[0].DeletedAt.Valid)
}

func TestCreateAppMount(t *testing.T) {
	db, err := setupSqLiteDB(POST_RECONCILE)
	// db, err := NewDBAccess(TMP) //todo
	require.NoError(t, err)
	// todo
	_, err = db.ReadTenantConfigByTenantUUID(context.Background(), "abc123")
	require.NoError(t, err)

	cm, err := db.ReadCodeModuleByTenantUUID(context.Background(), "abc123")
	require.NoError(t, err)

	appMount := &AppMount{
		VolumeMeta: VolumeMeta{
			ID:                "appmount1",
			PodUid:            "pod111",
			PodName:           "podiv",
			PodNamespace:      "testnamespace",
			PodServiceAccount: "podsa",
		},
		Location:      "loc1",
		MountAttempts: 0,
		CodeModule:    *cm,
	}

	err = db.CreateAppMount(context.Background(), appMount)
	require.NoError(t, err)

	var result []AppMount

	db.db.Raw(`SELECT * FROM app_mounts WHERE volume_meta_id = "appmount1"`).Scan(&result)
	assert.NotEmpty(t, result)
}

func TestReadAppMounts(t *testing.T) {
	db, err := setupSqLiteDB(POST_PUBLISH)
	require.NoError(t, err)

	appMounts, err := db.ReadAppMounts(context.Background())
	require.NoError(t, err)

	assert.NotNil(t, appMounts)
	assert.NotEmpty(t, len(appMounts))
	assert.Equal(t, "appmount1", appMounts[0].VolumeMeta.ID)
}

func TestReadAppMount(t *testing.T) {
	db, err := setupSqLiteDB(POST_PUBLISH)
	require.NoError(t, err)

	appMount, err := db.ReadAppMountByVolumeMetaID(context.Background(), "appmount1")
	require.NoError(t, err)

	assert.NotNil(t, appMount)
	assert.Equal(t, "appmount1", appMount.VolumeMeta.ID)

	_, err = db.ReadAppMountByVolumeMetaID(context.Background(), "")
	require.Error(t, err)

	_, err = db.ReadAppMountByVolumeMetaID(context.Background(), "unknown")
	require.Error(t, err)
}

func TestUpdateAppMount(t *testing.T) {
	db, err := setupSqLiteDB(POST_PUBLISH)
	require.NoError(t, err)

	appMount, err := db.ReadAppMountByVolumeMetaID(context.Background(), "appmount1")
	require.NoError(t, err)

	appMount.MountAttempts = 5

	err = db.UpdateAppMount(context.Background(), appMount)
	require.NoError(t, err)

	var result []OSMount

	db.db.Raw(`SELECT * FROM app_mounts WHERE volume_meta_id = "appmount1"`).Scan(&result)
	assert.NotEmpty(t, result)
	assert.Equal(t, int64(5), result[0].MountAttempts)
}

func TestSoftDeleteAppMount(t *testing.T) {
	db, err := setupSqLiteDB(POST_PUBLISH)
	require.NoError(t, err)

	appMount, err := db.ReadAppMountByVolumeMetaID(context.Background(), "appmount1")
	require.NoError(t, err)

	assert.NotNil(t, appMount)
	assert.Equal(t, "appmount1", appMount.VolumeMeta.ID)

	err = db.DeleteAppMount(context.Background(), appMount)
	require.NoError(t, err)

	var result []AppMount

	db.db.Raw(`SELECT * FROM app_mounts WHERE volume_meta_id = "appmount1"`).Scan(&result)
	assert.NotEmpty(t, result)
	assert.True(t, result[0].DeletedAt.Valid)
}

func setupSqLiteDB(templateDB string) (*DBConn, error) {
	if templateDB == MEM {
		db, err := NewDBAccess("file:csi_testdb?mode=memory")
		if err != nil {
			return nil, err
		}

		err = db.SchemaMigration(context.Background())
		if err != nil {
			return nil, err
		}

		return &db, nil
	}

	_ = os.Remove(TMP)

	err := copyFile(templateDB, TMP)
	if err != nil {
		return nil, err
	}

	db, err := NewDBAccess(TMP)

	return &db, err
}

func copyFile(source, dest string) error {
	from, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func(from *os.File) {
		_ = from.Close()
	}(from)

	to, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer func(to *os.File) {
		_ = to.Close()
	}(to)

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}

	return nil
}
