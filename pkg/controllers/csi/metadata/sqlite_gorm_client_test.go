package metadata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTenantConfig(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	tc := &TenantConfig{
		Name:                        "somename",
		ConfigDirPath:               "somewhere",
		DownloadedCodeModuleVersion: "1.2.3",
		TenantUUID:                  "abc123",
	}
	err = db.CreateTenantConfig(context.Background(), tc)
	require.NoError(t, err)

	var tcs []TenantConfig

	db.db.Raw(`SELECT * FROM tenant_configs WHERE tenant_uuid = "abc123"`).Scan(&tcs)
	assert.NotEmpty(t, tcs)
	assert.Equal(t, "somename", tcs[0].Name)
}

func TestReadTenantConfig(t *testing.T) {
	db, err := setupDB()
	setupPostReconcileData(context.Background(), db)

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
	db, err := setupDB()
	require.NoError(t, err)

	setupPostReconcileData(context.Background(), db)

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
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

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
	db, err := setupDB()
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
	db, err := setupDB()
	require.NoError(t, err)

	setupPostReconcileData(context.Background(), db)

	codeModule, err := db.ReadCodeModuleByVersion(context.Background(), "1.2.3")
	require.NoError(t, err)

	assert.NotNil(t, codeModule)
	assert.Equal(t, "someplace", codeModule.Location)

	_, err = db.ReadCodeModuleByVersion(context.Background(), "")
	require.Error(t, err)

	_, err = db.ReadCodeModuleByVersion(context.Background(), "unknown")
	require.Error(t, err)
}

func TestSoftDeleteCodeModule(t *testing.T) {
	db, err := setupDB()
	require.NoError(t, err)

	setupPostReconcileData(context.Background(), db)

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
	db, err := setupDB()
	require.NoError(t, err)

	setupPostReconcileData(context.Background(), db)

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
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

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
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

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
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

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
	db, err := setupDB()
	require.NoError(t, err)

	setupPostReconcileData(context.Background(), db)

	tenantConfig, err := db.ReadTenantConfigByTenantUUID(context.Background(), "abc123")
	require.NoError(t, err)

	cm, err := db.ReadCodeModuleByVersion(context.Background(), tenantConfig.DownloadedCodeModuleVersion)
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
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

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
	db, err := setupDB()
	require.NoError(t, err)

	setupPostPublishData(context.Background(), db)

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

func setupDB() (*DBConn, error) {
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

func setupPostReconcileData(ctx context.Context, conn *DBConn) {
	ctxDB := conn.db.WithContext(ctx)
	ctxDB.Exec("INSERT INTO code_modules (version, location, created_at, updated_at, deleted_at) VALUES ('1.2.3', 'someplace', '2024-03-11 17:07:43.038661+01:00', '2024-03-11 17:07:43.038661+01:00', null);")
	ctxDB.Exec("INSERT INTO tenant_configs (created_at, updated_at, deleted_at, uid, name, downloaded_code_module_version, config_dir_path, tenant_uuid, max_failed_mount_attempts) VALUES ('2024-03-11 16:42:39.323198+01:00', '2024-03-11 16:42:39.323198+01:00', null, '033dcff9-5c76-4b3a-9e3a-ea0a78bf0b3f', 'somename', '1.2.3', 'somewhere', 'abc123', 10);")
}

func setupPostPublishData(ctx context.Context, conn *DBConn) {
	setupPostReconcileData(ctx, conn)

	ctxDB := conn.db.WithContext(ctx)

	ctxDB.Exec("INSERT INTO volume_meta (id, pod_uid, pod_name, pod_namespace, pod_service_account, created_at, updated_at, deleted_at) VALUES ('osmount1', 'pod1', 'podi', 'testnamespace', 'podsa', '2024-03-12 07:49:37.943527+01:00', '2024-03-12 07:49:37.943527+01:00', null);")
	ctxDB.Exec("INSERT INTO volume_meta (id, pod_uid, pod_name, pod_namespace, pod_service_account, created_at, updated_at, deleted_at) VALUES ('appmount1', 'pod111', 'podiv', 'testnamespace', 'podsa', '2024-03-12 07:53:39.906052+01:00', '2024-03-12 07:53:39.906052+01:00', null);")
	ctxDB.Exec("INSERT INTO volume_meta (id, pod_uid, pod_name, pod_namespace, pod_service_account, created_at, updated_at, deleted_at) VALUES ('appmount2', 'pod121', 'podii', 'testnamespace', 'podsa', '2024-03-12 07:54:12.65411+01:00', '2024-03-12 07:54:12.65411+01:00', null);")
	ctxDB.Exec("INSERT INTO volume_meta (id, pod_uid, pod_name, pod_namespace, pod_service_account, created_at, updated_at, deleted_at) VALUES ('appmount3', 'pod113', 'podiii', 'testnamespace', 'podsa', '2024-03-12 07:54:31.114752+01:00', '2024-03-12 07:54:31.114752+01:00', null);")

	ctxDB.Exec("INSERT INTO os_mounts (created_at, updated_at, deleted_at, tenant_config_uid, tenant_uuid, volume_meta_id, location, mount_attempts) VALUES ('2024-03-12 07:49:37.94492+01:00', '2024-03-12 07:49:37.94492+01:00', null, '033dcff9-5c76-4b3a-9e3a-ea0a78bf0b3f', 'abc123', 'osmount1', 'somewhere', 0);")

	ctxDB.Exec("INSERT INTO app_mounts (created_at, updated_at, deleted_at, volume_meta_id, code_module_version, location, mount_attempts) VALUES ('2024-03-12 07:53:39.906761+01:00', '2024-03-12 07:53:39.906761+01:00', null, 'appmount1', '1.2.3', 'loc1', 0);")
	ctxDB.Exec("INSERT INTO app_mounts (created_at, updated_at, deleted_at, volume_meta_id, code_module_version, location, mount_attempts) VALUES ('2024-03-12 07:54:12.654891+01:00', '2024-03-12 07:54:12.654891+01:00', null, 'appmount2', '1.2.3', 'loc2', 0);")
	ctxDB.Exec("INSERT INTO app_mounts (created_at, updated_at, deleted_at, volume_meta_id, code_module_version, location, mount_attempts) VALUES ('2024-03-12 07:54:31.115563+01:00', '2024-03-12 07:54:31.115563+01:00', null, 'appmount3', '1.2.3', 'loc3', 0);")
}
