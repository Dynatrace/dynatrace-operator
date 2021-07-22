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
		UUID VARCHAR,
		LatestVersion VARCHAR,
		Dynakube VARCHAR,
		PRIMARY KEY (UUID)
	); `

	podsTableName       = "pods"
	podsCreateStatement = `
	CREATE TABLE IF NOT EXISTS pods (
		UID VARCHAR,
		VolumeID VARCHAR,
		Version VARCHAR,
		TenantUUID VARCHAR,
		PRIMARY KEY (UID)
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

	insertPodStatement = `
	INSERT INTO pods (UID, VolumeID, Version, TenantUUID)
	VALUES (?,?,?,?);
	`

	getPodViaVolumeIDStatement = `
	SELECT UID, Version, TenantUUID
	FROM pods
	WHERE VolumeID = ?;
	`

	deletePodStatement = "DELETE FROM pods WHERE UID = ?;"

	getUsedVersionsStatement = `
	SELECT Version
	FROM pods
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
	if _, err := a.conn.Exec(podsCreateStatement); err != nil {
		return fmt.Errorf("couldn't create the table %s, err: %s", podsTableName, err)
	}
	return nil
}

func (a *SqliteAccess) InsertTenant(tenant *Tenant) error {
	_, err := a.conn.Exec(insertTenantStatement, tenant.UUID, tenant.LatestVersion, tenant.Dynakube)
	if err != nil {
		return fmt.Errorf("couldn't insert tenant, UUID %s, LatestVersion %s, Dynakube %s, err: %s",
			tenant.UUID, tenant.LatestVersion, tenant.Dynakube, err)
	}
	return nil
}

func (a *SqliteAccess) UpdateTenant(tenant *Tenant) error {
	_, err := a.conn.Exec(updateTenantStatement, tenant.LatestVersion, tenant.Dynakube, tenant.UUID)
	if err != nil {
		return fmt.Errorf("couldn't update tenant, UUID %s, LatestVersion %s, Dynakube %s, err: %s",
			tenant.UUID, tenant.LatestVersion, tenant.Dynakube, err)
	}
	return nil
}

func (a *SqliteAccess) GetTenant(uuid string) (*Tenant, error) {
	var latestVersion string
	var dynakube string
	row := a.conn.QueryRow(getTenantStatement, uuid)
	err := row.Scan(&latestVersion, &dynakube)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("couldn't get tenant for UUID %s, err: %s", uuid, err)
	}
	if err == sql.ErrNoRows {
		return &Tenant{UUID: uuid}, nil
	}
	return &Tenant{uuid, latestVersion, dynakube}, nil
}

func (a *SqliteAccess) GetTenantViaDynakube(dynakube string) (*Tenant, error) {
	var tenantUUID string
	var latestVersion string
	row := a.conn.QueryRow(getTenantViaDynakubeStatement, dynakube)
	err := row.Scan(&tenantUUID, &latestVersion)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("couldn't get tenant field for Dynakube %s, err: %s", dynakube, err)
	}
	return &Tenant{tenantUUID, latestVersion, dynakube}, nil
}

func (a *SqliteAccess) InsertPodInfo(pod *Pod) error {
	_, err := a.conn.Exec(insertPodStatement, pod.UID, pod.VolumeID, pod.Version, pod.TenantUUID)
	if err != nil {
		return fmt.Errorf("couldn't insert pod info, UID %s, VolumeID %s, Version %s, TenantUUId: %s err: %s",
			pod.UID, pod.VolumeID, pod.Version, pod.TenantUUID, err)
	}
	return nil
}

func (a *SqliteAccess) GetPodViaVolumeId(volumeID string) (*Pod, error) {
	var uid string
	var version string
	var tenantUUID string
	row := a.conn.QueryRow(getPodViaVolumeIDStatement, volumeID)
	err := row.Scan(&uid, &version, &tenantUUID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("couldn't get pod field for VolumeID %s, err: %s", volumeID, err)
	}
	return &Pod{uid, volumeID, version, tenantUUID}, nil
}

func (a *SqliteAccess) DeletePodInfo(pod *Pod) error {
	_, err := a.conn.Exec(deletePodStatement, pod.UID)
	if err != nil {
		return fmt.Errorf("couldn't delete pod, UID %s, err: %s", pod.UID, err)
	}
	return nil
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
