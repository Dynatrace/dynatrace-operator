package metadata

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
		PRIMARY KEY (Dynakube)
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
	SET LatestVersion = ?, UUID = ?
	WHERE Dynakube = ?;
	`

	getTenantStatement = `
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

	deleteTenantStatement = "DELETE FROM tenants WHERE Dynakube = ?;"

	getUsedVersionsStatement = `
	SELECT Version
	FROM volumes
	WHERE TenantUUID = ?;
	`

	getPodNamesStatement = `
	SELECT ID, PodName
	FROM volumes;
	`

	getTenantsStatement = `
	SELECT UUID, Dynakube
	FROM tenants;
	`
)

var (
	log = logger.NewDTLogger().WithName("storage")
)

type SqliteAccess struct {
	conn *sql.DB
}

// NewAccess creates a new SqliteAccess, connects to the database.
func NewAccess(path string) (Access, error) {
	a := SqliteAccess{}
	err := a.Setup(path)
	if err != nil {
		log.Error(err, "failed to connect to the database, err: %s", err.Error())
		return nil, err
	}
	return &a, nil
}

func (a *SqliteAccess) connect(driver, path string) error {
	db, err := sql.Open(driver, path)
	if err != nil {
		err := fmt.Errorf("couldn't connect to db %s, err: %s", path, err)
		a.conn = nil
		return err
	}
	a.conn = db
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

// Setup connects to the database and creates the necessary tables if they don't exist
func (a *SqliteAccess) Setup(path string) error {
	if err := a.connect(sqliteDriverName, path); err != nil {
		return err
	}
	if err := a.createTables(); err != nil {
		return err
	}
	return nil
}

// InsertTenant inserts a new Tenant
func (a *SqliteAccess) InsertTenant(tenant *Tenant) error {
	err := a.executeStatement(insertTenantStatement, tenant.TenantUUID, tenant.LatestVersion, tenant.Dynakube)
	if err != nil {
		err = fmt.Errorf("couldn't insert tenant, uuid '%s', latest version '%s', dynakube '%s', err: %s",
			tenant.TenantUUID,
			tenant.LatestVersion,
			tenant.Dynakube,
			err)
	}
	return err
}

// UpdateTenant updates an existing Tenant by matching the Dynakube name
func (a *SqliteAccess) UpdateTenant(tenant *Tenant) error {
	err := a.executeStatement(updateTenantStatement, tenant.LatestVersion, tenant.TenantUUID, tenant.Dynakube)
	if err != nil {
		err = fmt.Errorf("couldn't update tenant, uuid '%s', latest version '%s', dynakube '%s', err: %s",
			tenant.TenantUUID,
			tenant.LatestVersion,
			tenant.Dynakube,
			err)
	}
	return err
}

// DeleteTenant deletes an existing Tenant by Dynakube name
func (a *SqliteAccess) DeleteTenant(dynakubeName string) error {
	err := a.executeStatement(deleteTenantStatement, dynakubeName)
	if err != nil {
		err = fmt.Errorf("couldn't delete tenant, dynakube '%s', err: %s", dynakubeName, err)
	}
	return err
}

// GetTenant gets Tenant by Dynakube name
func (a *SqliteAccess) GetTenant(dynakubeName string) (*Tenant, error) {
	var tenantUUID string
	var latestVersion string
	err := a.querySimpleStatement(getTenantStatement, dynakubeName, &tenantUUID, &latestVersion)
	if err != nil {
		err = fmt.Errorf("couldn't get tenant, dynakube '%s', err: %s", dynakubeName, err)
	}
	return NewTenant(tenantUUID, latestVersion, dynakubeName), err
}

// InsertVolume inserts a new Volume
func (a *SqliteAccess) InsertVolume(volume *Volume) error {
	err := a.executeStatement(insertVolumeStatement, volume.VolumeID, volume.PodName, volume.Version, volume.TenantUUID)
	if err != nil {
		err = fmt.Errorf("couldn't insert volume info, volume id '%s', pod '%s', version '%s', dynakube '%s', err: %s",
			volume.VolumeID,
			volume.PodName,
			volume.Version,
			volume.TenantUUID,
			err)
	}
	return err
}

// GetVolume gets Volume by its ID
func (a *SqliteAccess) GetVolume(volumeID string) (*Volume, error) {
	var PodName string
	var version string
	var tenantUUID string
	err := a.querySimpleStatement(getVolumeStatement, volumeID, &PodName, &version, &tenantUUID)
	if err != nil {
		err = fmt.Errorf("couldn't get volume field for volume id '%s', err: %s", volumeID, err)
	}
	return NewVolume(volumeID, PodName, version, tenantUUID), err
}

// DeleteVolume deletes a Volume by its ID
func (a *SqliteAccess) DeleteVolume(volumeID string) error {
	err := a.executeStatement(deleteVolumeStatement, volumeID)
	if err != nil {
		err = fmt.Errorf("couldn't delete volume for volume id '%s', err: %s", volumeID, err)
	}
	return err
}

// GetUsedVersions gets all UNIQUE versions present in the `volumes` database in map.
// Map is used to make sure we don't return the same version multiple time,
// it's also easier to check if a version is in it or not. (a Set in style of Golang)
func (a *SqliteAccess) GetUsedVersions(tenantUUID string) (map[string]bool, error) {
	rows, err := a.conn.Query(getUsedVersionsStatement, tenantUUID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get used version info for tenant uuid '%s', err: %s", tenantUUID, err)
	}
	versions := map[string]bool{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var version string
		err := rows.Scan(&version)
		if err != nil {
			return nil, fmt.Errorf("failed to scan from database for tenant uuid '%s', err: %s", tenantUUID, err)
		}
		if _, ok := versions[version]; !ok {
			versions[version] = true
		}
	}
	return versions, nil
}

// GetPodNames gets all PodNames present in the `volumes` database in map with their corresponding volumeIDs.
func (a *SqliteAccess) GetPodNames() (map[string]string, error) {
	rows, err := a.conn.Query(getPodNamesStatement)
	if err != nil {
		return nil, fmt.Errorf("couldn't get pod names, err: %s", err)
	}
	podNames := map[string]string{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var podName string
		var volumeID string
		err := rows.Scan(&volumeID, &podName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan from database for pod names, err: %s", err)
		}
		podNames[podName] = volumeID
	}
	return podNames, nil
}

// GetTenants gets all Tenants and maps their tenantUUID to the corresponding Dynakubes.
func (a *SqliteAccess) GetTenants() (map[string]string, error) {
	rows, err := a.conn.Query(getTenantsStatement)
	if err != nil {
		return nil, fmt.Errorf("couldn't get tenants, err: %s", err)
	}
	tenants := map[string]string{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var uuid string
		var dynakube string
		err := rows.Scan(&uuid, &dynakube)
		if err != nil {
			return nil, fmt.Errorf("failed to scan from database for tenant, err: %s", err)
		}
		tenants[dynakube] = uuid
	}
	return tenants, nil
}

// Executes the provided SQL statement on the database.
// The `vars` are passed to the SQL statement (in-order), to fill in the SQL wildcards.
func (a *SqliteAccess) executeStatement(statement string, vars ...interface{}) error {
	_, err := a.conn.Exec(statement, vars...)
	return err
}

// Executes the provided SQL SELECT statement on the database.
// The SQL statement should always return a single row.
// The `id` is passed to the SQL query to fill in an SQL wildcard
// The `vars` are filled with the values of the return of the SELECT statement, so the `vars` need to be pointers.
func (a *SqliteAccess) querySimpleStatement(statement, id string, vars ...interface{}) error {
	row := a.conn.QueryRow(statement, id)
	err := row.Scan(vars...)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}
