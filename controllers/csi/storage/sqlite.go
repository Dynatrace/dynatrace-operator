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
		PodName VARCHAR NOT NULL,
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
	INSERT INTO volumes (ID, PodName, Version, TenantUUID)
	VALUES (?,?,?,?);
	`

	getVolumeStatement = `
	SELECT PodName, Version, TenantUUID
	FROM volumes
	WHERE ID = ?;
	`

	deleteVolumeStatement = "DELETE FROM volumes WHERE ID = ?;"

	getUsedVersionsStatement = `
	SELECT Version
	FROM volumes
	WHERE TenantUUID = ?;
	`

	getPodNamesStatement = `
	SELECT PodName
	FROM volumes;
	`
)

var (
	log = logger.NewDTLogger().WithName("provisioner")
)

type SqliteAccess struct {
	conn *sql.DB
}

// Creates a new SqliteAccess,
//connects to the database and creates the necessary tables if they don't exists
func NewAccess() Access {
	a := SqliteAccess{}
	err := a.Connect(sqliteDriverName, dbPath)
	if err != nil {
		log.Error(err, "Failed to init the database, err: %s", err.Error())
	}
	return &a
}

// Connects to the database via the provided driver and path to the database.
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

//Connects to the database and creates the necessary tables if they don't exists
func (a *SqliteAccess) Setup() error {
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
	return a.executeStatement(insertTenantStatement, errMessageTemplate, tenant.UUID, tenant.LatestVersion, tenant.Dynakube)
}

func (a *SqliteAccess) UpdateTenant(tenant *Tenant) error {
	errMessageTemplate := "couldn't update tenant, LatestVersion %s, Dynakube %s, UUID %s, err: %s"
	return a.executeStatement(updateTenantStatement, errMessageTemplate, tenant.LatestVersion, tenant.Dynakube, tenant.UUID)
}

// Gets a Tenant from the database, return (nil, nil) if the tenant is not in the database.
func (a *SqliteAccess) GetTenant(uuid string) (*Tenant, error) {
	var latestVersion string
	var dynakube string
	errMessageTemplate := "couldn't get tenant for UUID %s, err: %s"
	err := a.querySimpleStatement(getTenantStatement, uuid, errMessageTemplate, &latestVersion, &dynakube)
	return NewTenant(uuid, latestVersion, dynakube), err
}

// Gets a Tenant from the database via its dynakube, return (nil, nil) if the tenant is not in the database.
// Needed during NodePublishVolume.
func (a *SqliteAccess) GetTenantViaDynakube(dynakube string) (*Tenant, error) {
	var tenantUUID string
	var latestVersion string
	errMessageTemplate := "couldn't get tenant field for Dynakube %s, err: %s"
	err := a.querySimpleStatement(getTenantViaDynakubeStatement, dynakube, errMessageTemplate, &tenantUUID, &latestVersion)
	return NewTenant(tenantUUID, latestVersion, dynakube), err
}

func (a *SqliteAccess) InsertVolumeInfo(volume *Volume) error {
	errMessageTemplate := "couldn't insert volume info, UID %s, VolumeID %s, Version %s, TenantUUId: %s err: %s"
	return a.executeStatement(insertVolumeStatement, errMessageTemplate, volume.ID, volume.PodName, volume.Version, volume.TenantUUID)
}

// Gets a Volume from the database, return (nil, nil) if the volume is not in the database.
func (a *SqliteAccess) GetVolumeInfo(volumeID string) (*Volume, error) {
	var PodName string
	var version string
	var tenantUUID string
	errMessageTemplate := "couldn't get volume field for VolumeID %s, err: %s"
	err := a.querySimpleStatement(getVolumeStatement, volumeID, errMessageTemplate, &PodName, &version, &tenantUUID)
	return NewVolume(volumeID, PodName, version, tenantUUID), err
}

func (a *SqliteAccess) DeleteVolumeInfo(volumeID string) error {
	errMessageTemplate := "couldn't delete pod, UID %s, err: %s"
	return a.executeStatement(deleteVolumeStatement, errMessageTemplate, volumeID)
}

// Gets all unique versions present in the `volumes` database in map.
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

// Gets all PodNames present in the `volumes` database in map with their corresponding volumeIDs.
func (a *SqliteAccess) GetPodNames() (map[string]string, error) {
	rows, err := a.conn.Query(getPodNamesStatement)
	if err != nil {
		return nil, fmt.Errorf("couldn't get PodName info for, err: %s", err)
	}
	podNames := map[string]string{}
	defer rows.Close()
	for rows.Next() {
		var podName string
		var volumeID string
		err := rows.Scan(&podName, &volumeID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan from database for PodName, err: %s", err)
		}
		podNames[podName] = volumeID
	}
	return podNames, nil
}

// Excutes the provided SQL statement on the database.
// The `vars` are passed to the SQL statement (in-order), to fill in the SQL wildcards.
// The `errMessageTemplate` will be passed the same `vars` + the `err` object, so the template needs to have len(vars) + 1 wildcards.
func (a *SqliteAccess) executeStatement(statement, errMessageTemplate string, vars ...interface{}) error {
	_, err := a.conn.Exec(statement, vars...)
	if err != nil {
		vars = append(vars, err)
		return fmt.Errorf(errMessageTemplate, vars...)
	}
	return nil
}

// Excutes the provided SQL SELECT statement on the database.
// The SQL statement should always return a single row.
// The `id` is passed to the SQL query to fill in an SQL wildcard
// The `vars` are filled with the values of the return of the SELECT statement, so the `vars` need to be pointers.
// The `errMessageTemplate` will be passed the `id` + the `err` object, so the template needs to have 2 wildcards.
func (a *SqliteAccess) querySimpleStatement(statement, id, errMessageTemplate string, vars ...interface{}) error {
	row := a.conn.QueryRow(statement, id)
	err := row.Scan(vars...)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf(errMessageTemplate, id, err)
	}
	return nil
}
