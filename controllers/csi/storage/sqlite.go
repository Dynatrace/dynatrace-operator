package storage

import (
	"database/sql"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/logger"
	_ "github.com/mattn/go-sqlite3"
)

const (
	sqliteDriverName = "sqlite3"

	tenantsTableName       = "tenants"
	tenantsCreateStatement = `
	CREATE TABLE IF NOT EXISTS tenants (
		UUID VARCHAR NOT NULL,
		LatestVersion VARCHAR NOT NULL,
		Dynakube VARCHAR NOT NULL,
		PRIMARY KEY (UUID)
	); `

	volumesTableName       = "volumes"
	volumesCreateStatement = `
	CREATE TABLE IF NOT EXISTS volumes (
		ID VARCHAR NOT NULL,
		PodUID VARCHAR NOT NULL,
		Version VARCHAR NOT NULL,
		TenantUUID VARCHAR NOT NULL,
		PRIMARY KEY (ID)
	);`

	insertTenantStatement = `
	INSERT INTO tenants (UUID, LatestVersion, Dynakube)
	VALUES (?,?,?);
	`
	updateTenantStatement = `
	UPDATE tenants
	SET LatestVersion = ?, Dynakube = ?
	WHERE UUID = ?;
	`

	getTenantStatement = `
	SELECT LatestVersion, Dynakube
	FROM tenants
	WHERE UUID = ?;
	`

	getTenantViaDynakubeStatement = `
	SELECT UUID, LatestVersion
	FROM tenants
	WHERE Dynakube = ?;
	`

	insertVolumeStatement = `
	INSERT INTO volumes (ID, PodUID, Version, TenantUUID)
	VALUES (?,?,?,?);
	`

	getVolumeStatement = `
	SELECT PodUID, Version, TenantUUID
	FROM volumes
	WHERE ID = ?;
	`

	deleteVolumeStatement = "DELETE FROM volumes WHERE ID = ?;"

	getUsedVersionsStatement = `
	SELECT Version
	FROM volumes
	WHERE TenantUUID = ?;
	`
)

var (
	log = logger.NewDTLogger().WithName("provisioner")
)

type SqliteAccess struct {
	conn *sql.DB
}

func NewAccess() *SqliteAccess {
	a := SqliteAccess{}
	err := a.init()
	if err != nil {
		log.Error(err, "Failed to init the database, err: %s", err.Error())
	}
	return &a
}

func (a *SqliteAccess) Connect(driver, path string) error {
	db, err := sql.Open(driver, path)
	if err != nil {
		err := fmt.Errorf("couldn't connect to db %s, err: %s", path, err)
		a.conn = nil
		return err
	}
	a.conn = db
	return nil
}

func (a *SqliteAccess) init() error {
	if err := a.Connect(sqliteDriverName, dbPath); err != nil {
		return err
	}
	if err := a.createTables(); err != nil {
		return err
	}
	return nil
}

func (a *SqliteAccess) createTables() error {
	if _, err := a.conn.Exec(tenantsCreateStatement); err != nil {
		return fmt.Errorf("couldn't create the table %s, err: %s", tenantsTableName, err)
	}
	if _, err := a.conn.Exec(volumesCreateStatement); err != nil {
		return fmt.Errorf("couldn't create the table %s, err: %s", volumesTableName, err)
	}
	return nil
}

func (a *SqliteAccess) InsertTenant(tenant *Tenant) error {
	errMessageTemplate := "couldn't insert tenant, UUID %s, LatestVersion %s, Dynakube %s, err: %s"
	return a.exectueStatement(insertTenantStatement, errMessageTemplate, tenant.UUID, tenant.LatestVersion, tenant.Dynakube)
}

func (a *SqliteAccess) UpdateTenant(tenant *Tenant) error {
	errMessageTemplate := "couldn't update tenant, LatestVersion %s, Dynakube %s, UUID %s, err: %s"
	return a.exectueStatement(updateTenantStatement, errMessageTemplate, tenant.LatestVersion, tenant.Dynakube, tenant.UUID)
}

func (a *SqliteAccess) GetTenant(uuid string) (*Tenant, error) {
	var latestVersion string
	var dynakube string
	errMessageTemplate := "couldn't get tenant for UUID %s, err: %s"
	a.querySimpleStatement(getTenantStatement, uuid, errMessageTemplate, &latestVersion, &dynakube)
	return NewTenant(uuid, latestVersion, dynakube), nil
}

func (a *SqliteAccess) GetTenantViaDynakube(dynakube string) (*Tenant, error) {
	var tenantUUID string
	var latestVersion string
	errMessageTemplate := "couldn't get tenant field for Dynakube %s, err: %s"
	a.querySimpleStatement(getTenantViaDynakubeStatement, dynakube, errMessageTemplate, &tenantUUID, &latestVersion)
	return NewTenant(tenantUUID, latestVersion, dynakube), nil
}

func (a *SqliteAccess) InsertVolumeInfo(volume *Volume) error {
	errMessageTemplate := "couldn't insert volume info, UID %s, VolumeID %s, Version %s, TenantUUId: %s err: %s"
	return a.exectueStatement(insertVolumeStatement, errMessageTemplate, volume.ID, volume.PodUID, volume.Version, volume.TenantUUID)
}

func (a *SqliteAccess) GetVolumeInfo(volumeID string) (*Volume, error) {
	var podUID string
	var version string
	var tenantUUID string
	errMessageTemplate := "couldn't get volume field for VolumeID %s, err: %s"
	a.querySimpleStatement(getVolumeStatement, volumeID, errMessageTemplate, &podUID, &version, &tenantUUID)
	return NewVolume(volumeID, podUID, version, tenantUUID), nil
}

func (a *SqliteAccess) DeleteVolumeInfo(volumeID string) error {
	errMessageTemplate := "couldn't delete pod, UID %s, err: %s"
	return a.exectueStatement(deleteVolumeStatement, errMessageTemplate, volumeID)
}

func (a *SqliteAccess) GetUsedVersions(tenantUUID string) (map[string]bool, error) {
	rows, err := a.conn.Query(getUsedVersionsStatement, tenantUUID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get used version info for tenantUUID %s, err: %s", tenantUUID, err)
	}
	versions := map[string]bool{}
	defer rows.Close()
	for rows.Next() {
		var version string
		err := rows.Scan(&version)
		if err != nil {
			return nil, fmt.Errorf("failed to scan from database for tenantUUID %s, err: %s", tenantUUID, err)
		}
		if _, ok := versions[version]; !ok {
			versions[version] = true
		}
	}
	return versions, nil
}

func (a *SqliteAccess) exectueStatement(statement, errMessageTemplate string, vars ...interface{}) error {
	_, err := a.conn.Exec(statement, vars...)
	if err != nil {
		vars = append(vars, err)
		return fmt.Errorf(errMessageTemplate, vars...)
	}
	return nil
}

func (a *SqliteAccess) querySimpleStatement(statement, id, errMessageTemplate string, vars ...interface{}) error {
	row := a.conn.QueryRow(statement, id)
	err := row.Scan(vars...)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf(errMessageTemplate, id, err)
	}
	return nil
}

func emptyMemoryDB() SqliteAccess {
	path := ":memory:"
	db := SqliteAccess{}
	db.Connect(sqliteDriverName, path)
	return db
}

func FakeMemoryDB() *SqliteAccess {
	db := emptyMemoryDB()
	db.createTables()
	return &db
}
