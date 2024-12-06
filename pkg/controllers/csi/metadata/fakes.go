package metadata

import (
	"context"
	"database/sql"
)

func FakeMemoryDB() *SqliteAccess {
	db := SqliteAccess{}
	ctx := context.Background()
	_ = db.Setup(ctx, ":memory:")
	_ = db.createTables(ctx)

	return &db
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
func (f *FakeFailDB) GetAllAppMounts(_ context.Context) []*Volume {
	return nil
}
func (f *FakeFailDB) DeleteAppMount(_ context.Context, _ string) error { return nil }

func (f *FakeFailDB) GetUsedImageDigests(_ context.Context) (map[string]bool, error) {
	return nil, sql.ErrTxDone
}

func (f *FakeFailDB) IsImageDigestUsed(_ context.Context, _ string) (bool, error) {
	return false, sql.ErrTxDone
}
