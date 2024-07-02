package metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fillWithTenantConfigs(t *testing.T, db *GormConn, amount int) {
	for i := range amount {
		tConfig := generateTenantConfig(i)
		err := db.CreateTenantConfig(tConfig)
		require.NoError(t, err)

		if i%2 == 0 {
			err := db.DeleteTenantConfig(tConfig, false)
			require.NoError(t, err)
		}
	}
}

func generateTenantConfig(i int) *TenantConfig {
	return &TenantConfig{
		Name:                        fmt.Sprintf("tenant-%d", i),
		DownloadedCodeModuleVersion: fmt.Sprintf("version-%d", i),
		ConfigDirPath:               fmt.Sprintf("path-%d", i),
		TenantUUID:                  fmt.Sprintf("uuid-%d", i),
		MaxFailedMountAttempts:      int64(i),
	}
}

func TestListDeletedTenantConfigs(t *testing.T) {
	t.Run("empty database => no error", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		configs, err := db.ListDeletedTenantConfigs()
		require.NoError(t, err)
		assert.Empty(t, configs)
	})

	t.Run("only list deleted", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		initialLength := 6
		fillWithTenantConfigs(t, db, initialLength)

		configs, err := db.ListDeletedTenantConfigs()
		require.NoError(t, err)
		assert.Len(t, configs, initialLength/2)
	})
}

func TestPurgeTenantConfig(t *testing.T) {
	t.Run("nil/empty input => error", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		err = db.PurgeTenantConfig(nil)
		require.Error(t, err)

		err = db.PurgeTenantConfig(&TenantConfig{})
		require.Error(t, err)
	})

	t.Run("delete everything", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		initialLength := 6
		fillWithTenantConfigs(t, db, initialLength)

		for i := range initialLength {
			err = db.PurgeTenantConfig(generateTenantConfig(i))
			require.NoError(t, err)
		}

		tcs, err := db.ListDeletedTenantConfigs()
		require.NoError(t, err)
		assert.Empty(t, tcs)
	})
}

func fillWithCodeModules(t *testing.T, db *GormConn, amount int) {
	for i := range amount {
		cm := generateCodeModule(i)
		err := db.CreateCodeModule(cm)
		require.NoError(t, err)

		if i%2 == 0 {
			err := db.DeleteCodeModule(cm)
			require.NoError(t, err)
		}
	}
}

func generateCodeModule(i int) *CodeModule {
	return &CodeModule{
		Version:  fmt.Sprintf("version-%d", i),
		Location: fmt.Sprintf("location-%d", i),
	}
}

func TestListDeletedCodeModules(t *testing.T) {
	t.Run("empty database => no error", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		cms, err := db.ListDeletedCodeModules()
		require.NoError(t, err)
		assert.Empty(t, cms)
	})

	t.Run("only list deleted", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		initialLength := 6
		fillWithCodeModules(t, db, initialLength)

		cms, err := db.ListDeletedCodeModules()
		require.NoError(t, err)
		assert.Len(t, cms, initialLength/2)
	})
}

func TestPurgeCodeModule(t *testing.T) {
	t.Run("nil/empty input => error", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		err = db.PurgeCodeModule(nil)
		require.Error(t, err)

		err = db.PurgeCodeModule(&CodeModule{})
		require.Error(t, err)
	})
	t.Run("delete everything", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		initialLength := 6
		fillWithCodeModules(t, db, initialLength)

		for i := range initialLength {
			err = db.PurgeCodeModule(generateCodeModule(i))
			require.NoError(t, err)
		}

		cms, err := db.ListDeletedCodeModules()
		require.NoError(t, err)
		assert.Empty(t, cms)
	})
}

func fillWithAppMounts(t *testing.T, db *GormConn, amount int) {
	for i := range amount {
		am := generateAppMount(i)
		err := db.CreateAppMount(am)
		require.NoError(t, err)

		if i%2 == 0 {
			err := db.DeleteAppMount(am)
			require.NoError(t, err)
		}
	}
}

func generateAppMount(i int) *AppMount {
	return &AppMount{
		VolumeMetaID:      fmt.Sprintf("id-%d", i),
		VolumeMeta:        VolumeMeta{ID: fmt.Sprintf("id-%d", i)},
		CodeModule:        CodeModule{Version: fmt.Sprintf("version-%d", i)},
		CodeModuleVersion: fmt.Sprintf("version-%d", i),
		Location:          fmt.Sprintf("location-%d", i),
	}
}

func TestListDeletedAppMounts(t *testing.T) {
	t.Run("empty database => no error", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		mounts, err := db.ListDeletedAppMounts()
		require.NoError(t, err)
		assert.Empty(t, mounts)
	})
	t.Run("only list deleted", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		initialLength := 6
		fillWithAppMounts(t, db, initialLength)

		ams, err := db.ListDeletedAppMounts()
		require.NoError(t, err)
		assert.Len(t, ams, initialLength/2)
	})
}

func TestPurgeAppMount(t *testing.T) {
	t.Run("nil/empty input => error", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		err = db.PurgeAppMount(nil)
		require.Error(t, err)

		err = db.PurgeAppMount(&AppMount{})
		require.Error(t, err)
	})
	t.Run("delete everything", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		initialLength := 6
		fillWithAppMounts(t, db, initialLength)

		for i := range initialLength {
			err = db.PurgeAppMount(generateAppMount(i))
			require.NoError(t, err)
		}

		tcs, err := db.ListDeletedAppMounts()
		require.NoError(t, err)
		assert.Empty(t, tcs)
	})
}

func TestListDeletedOSMounts(t *testing.T) {
	t.Run("empty database => no error", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		mounts, err := db.ListDeletedOSMounts()
		require.NoError(t, err)
		assert.Empty(t, mounts)
	})
	t.Run("only list deleted", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		initialLength := 7
		fillWithOSMounts(t, db, initialLength)

		oms, err := db.ListDeletedOSMounts()
		require.NoError(t, err)
		assert.Len(t, oms, 2)

		osMount, err := db.ReadOSMount(OSMount{VolumeMetaID: "restore"})
		require.NoError(t, err)
		assert.NotEmpty(t, osMount)

		osMount, err = db.ReadOSMount(OSMount{TenantConfig: TenantConfig{Name: "restore"}})
		require.NoError(t, err)
		assert.NotEmpty(t, osMount)
	})
}

func TestPurgeOSMount(t *testing.T) {
	t.Run("nil/empty input => error", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		err = db.PurgeOSMount(nil)
		require.Error(t, err)

		err = db.PurgeOSMount(&OSMount{})
		require.Error(t, err)
	})
	t.Run("delete everything", func(t *testing.T) {
		db, err := setupDB()
		require.NoError(t, err)

		initialLength := 6
		fillWithOSMounts(t, db, initialLength)

		for i := range initialLength {
			// I don't understand why I can't just pass in the generated OSMount.
			// It just doesn't clean up the already soft-deleted entries if I do.
			// (I tested it live, and it has no problems)
			tmp := generateOSMount(i)
			err = db.PurgeOSMount(&OSMount{TenantUUID: tmp.TenantUUID, VolumeMeta: VolumeMeta{ID: tmp.VolumeMetaID}})
			require.NoError(t, err)
		}

		tcs, err := db.ListDeletedOSMounts()
		require.NoError(t, err)
		assert.Empty(t, tcs)
	})
}

func fillWithOSMounts(t *testing.T, db *GormConn, amount int) {
	for i := range amount {
		om := generateOSMount(i)
		err := db.CreateOSMount(om)
		require.NoError(t, err)

		if i%2 == 0 {
			err := db.DeleteOSMount(om)
			require.NoError(t, err)
		}

		if i%4 == 0 {
			tmp, err := db.ReadUnscopedOSMount(OSMount{TenantUUID: om.TenantUUID})
			require.NoError(t, err)
			assert.NotNil(t, tmp)
			tmp.VolumeMeta = VolumeMeta{ID: "restore"}
			tmp.TenantConfig = TenantConfig{Name: "restore"}
			tmp, err = db.RestoreOSMount(tmp)
			require.NoError(t, err)
			require.NotNil(t, tmp)
		}
	}
}

func generateOSMount(i int) *OSMount {
	return &OSMount{
		VolumeMetaID:    fmt.Sprintf("id-%d", i),
		VolumeMeta:      VolumeMeta{ID: fmt.Sprintf("id-%d", i)},
		TenantConfigUID: fmt.Sprintf("t-id-%d", i),
		TenantConfig:    TenantConfig{UID: fmt.Sprintf("t-id-%d", i), TenantUUID: fmt.Sprintf("uuid-%d", i)},
		Location:        fmt.Sprintf("location-%d", i),
		TenantUUID:      fmt.Sprintf("uuid-%d", i),
		MountAttempts:   int64(i),
	}
}
