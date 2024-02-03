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
	ctx := context.TODO()
	db.Setup(ctx, ":memory:")
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

func (f *FakeFailDB) Setup(ctx context.Context, dbPath string) error { return sql.ErrTxDone }
func (f *FakeFailDB) InsertDynakube(ctx context.Context, tenant *Dynakube) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) UpdateDynakube(ctx context.Context, tenant *Dynakube) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) DeleteDynakube(ctx context.Context, dynakubeName string) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) GetDynakube(ctx context.Context, dynakubeName string) (*Dynakube, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetTenantsToDynakubes(ctx context.Context) (map[string]string, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetAllDynakubes(ctx context.Context) ([]*Dynakube, error) {
	return nil, sql.ErrTxDone
}

func (f *FakeFailDB) InsertOsAgentVolume(ctx context.Context, volume *OsAgentVolume) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) GetOsAgentVolumeViaVolumeID(ctx context.Context, volumeID string) (*OsAgentVolume, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetOsAgentVolumeViaTenantUUID(ctx context.Context, volumeID string) (*OsAgentVolume, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) UpdateOsAgentVolume(ctx context.Context, volume *OsAgentVolume) error {
	return sql.ErrTxDone
}
func (f *FakeFailDB) GetAllOsAgentVolumes(ctx context.Context) ([]*OsAgentVolume, error) {
	return nil, sql.ErrTxDone
}

func (f *FakeFailDB) InsertVolume(ctx context.Context, volume *Volume) error  { return sql.ErrTxDone }
func (f *FakeFailDB) DeleteVolume(ctx context.Context, volumeID string) error { return sql.ErrTxDone }
func (f *FakeFailDB) GetVolume(ctx context.Context, volumeID string) (*Volume, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetAllVolumes(ctx context.Context) ([]*Volume, error) { return nil, sql.ErrTxDone }
func (f *FakeFailDB) GetPodNames(ctx context.Context) (map[string]string, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetUsedVersions(ctx context.Context, tenantUUID string) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetAllUsedVersions(ctx context.Context) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}
func (f *FakeFailDB) GetLatestVersions(ctx context.Context) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}

func (f *FakeFailDB) GetUsedImageDigests(ctx context.Context) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}

func (f *FakeFailDB) IsImageDigestUsed(ctx context.Context, imageDigest string) (bool, error) {
	return false, sql.ErrTxDone
}
