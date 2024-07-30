package metadata

import (
	"context"
	"database/sql"
)

func emptyMemoryDB() *SqliteAccess {
	db := SqliteAccess{}
	_ = db.connect(sqliteDriverName, ":memory:")

	return &db
}

func FakeMemoryDB() *SqliteAccess {
	db := SqliteAccess{}
	ctx := context.Background()
	_ = db.Setup(ctx, ":memory:")
	_ = db.createTables(ctx)

	return &db
}

func checkIfTablesExist(db *SqliteAccess) bool {
	var volumesTable string

	row := db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", volumesTableName)

	err := row.Scan(&volumesTable)
	if err != nil {
		return false
	}

	var tenantsTable string

	row = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", dynakubesTableName)

	err = row.Scan(&tenantsTable)
	if err != nil {
		return false
	}

	if tenantsTable != dynakubesTableName || volumesTable != volumesTableName {
		return false
	}

	return true
}

type FakeFailDB struct{}

func (f *FakeFailDB) Setup(_ context.Context, _ string) error { return sql.ErrTxDone }
func (f *FakeFailDB) InsertDynakube(_ context.Context, _ *Dynakube) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) UpdateDynakube(_ context.Context, _ *Dynakube) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) DeleteDynakube(_ context.Context, _ string) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) GetDynakube(_ context.Context, _ string) (*Dynakube, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetTenantsToDynakubes(_ context.Context) (map[string]string, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetAllDynakubes(_ context.Context) ([]*Dynakube, error) {
	return nil, sql.ErrTxDone
}

func (f *FakeFailDB) InsertOsAgentVolume(_ context.Context, _ *OsAgentVolume) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) GetOsAgentVolumeViaVolumeID(_ context.Context, _ string) (*OsAgentVolume, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetOsAgentVolumeViaTenantUUID(_ context.Context, _ string) (*OsAgentVolume, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) UpdateOsAgentVolume(_ context.Context, _ *OsAgentVolume) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) GetAllOsAgentVolumes(_ context.Context) ([]*OsAgentVolume, error) {
	return nil, sql.ErrTxDone
}

func (f *FakeFailDB) InsertVolume(_ context.Context, _ *Volume) error { return sql.ErrTxDone }
func (f *FakeFailDB) DeleteVolume(_ context.Context, _ string) error  { return sql.ErrTxDone }
func (f *FakeFailDB) GetVolume(_ context.Context, _ string) (*Volume, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetAllVolumes(_ context.Context) ([]*Volume, error) { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetPodNames(_ context.Context) (map[string]string, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetUsedVersions(_ context.Context, _ string) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetAllUsedVersions(_ context.Context) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetLatestVersions(_ context.Context) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}

func (f *FakeFailDB) GetUsedImageDigests(_ context.Context) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}

func (f *FakeFailDB) IsImageDigestUsed(_ context.Context, _ string) (bool, error) {
	return false, sql.ErrTxDone
}
