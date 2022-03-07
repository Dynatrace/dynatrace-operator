package metadata

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	sqliteDriverName = "sqlite3"

	// CREATE
	dynakubesTableName       = "dynakubes"
	dynakubesCreateStatement = `
	CREATE TABLE IF NOT EXISTS dynakubes (
		Name VARCHAR NOT NULL,
		TenantUUID VARCHAR NOT NULL,
		LatestVersion VARCHAR NOT NULL,
		PRIMARY KEY (Name)
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

	osAgentVolumesTableName       = "osagent_volumes"
	osAgentVolumesCreateStatement = `
	CREATE TABLE IF NOT EXISTS osagent_volumes (
		TenantUUID VARCHAR NOT NULL,
		VolumeID VARCHAR NOT NULL,
		Mounted BOOLEAN NOT NULL,
		LastModified DATETIME NOT NULL,
		PRIMARY KEY (TenantUUID)
	);`

	// INSERT
	insertDynakubeStatement = `
	INSERT INTO dynakubes (Name, TenantUUID, LatestVersion)
	VALUES (?,?,?);
	`

	insertVolumeStatement = `
	INSERT INTO volumes (ID, PodName, Version, TenantUUID)
	VALUES (?,?,?,?);
	`

	insertOsAgentVolumeStatement = `
	INSERT INTO osagent_volumes (TenantUUID, VolumeID, Mounted, LastModified)
	VALUES (?,?,?,?);
	`

	// UPDATE
	updateDynakubeStatement = `
	UPDATE dynakubes
	SET LatestVersion = ?, TenantUUID = ?
	WHERE Name = ?;
	`

	updateOsAgentVolumeStatement = `
	UPDATE osagent_volumes
	SET VolumeID = ?, Mounted = ?, LastModified = ?
	WHERE TenantUUID = ?;
	`

	// GET
	getDynakubeStatement = `
	SELECT TenantUUID, LatestVersion
	FROM dynakubes
	WHERE Name = ?;
	`

	getVolumeStatement = `
	SELECT PodName, Version, TenantUUID
	FROM volumes
	WHERE ID = ?;
	`

	getOsAgentVolumeViaVolumeIDStatement = `
	SELECT TenantUUID, Mounted, LastModified
	FROM osagent_volumes
	WHERE VolumeID = ?;
	`

	getOsAgentVolumeViaTenantUUIDStatement = `
	SELECT VolumeID, Mounted, LastModified
	FROM osagent_volumes
	WHERE TenantUUID = ?;
	`

	// Dump
	dumpDynakubesStatement = `
		SELECT Name, TenantUUID, LatestVersion
		FROM dynakubes;
		`

	dumpVolumesStatement = `
		SELECT ID, PodName, Version, TenantUUID
		FROM volumes;
		`

	dumpOsAgentVolumes = `
		SELECT TenantUUID, VolumeID, Mounted, LastModified
		FROM osagent_volumes;
		`

	// DELETE
	deleteVolumeStatement = "DELETE FROM volumes WHERE ID = ?;"

	deleteDynakubeStatement = "DELETE FROM dynakubes WHERE Name = ?;"

	// SPECIAL
	getUsedVersionsStatement = `
	SELECT Version
	FROM volumes
	WHERE TenantUUID = ?;
	`

	getPodNamesStatement = `
	SELECT ID, PodName
	FROM volumes;
	`

	getDynakubesStatement = `
	SELECT tenantUUID, Name
	FROM dynakubes;
	`
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
	if _, err := a.conn.Exec(dynakubesCreateStatement); err != nil {
		return fmt.Errorf("couldn't create the table %s, err: %s", dynakubesTableName, err)
	}
	if _, err := a.conn.Exec(volumesCreateStatement); err != nil {
		return fmt.Errorf("couldn't create the table %s, err: %s", volumesTableName, err)
	}
	if _, err := a.conn.Exec(osAgentVolumesCreateStatement); err != nil {
		return fmt.Errorf("couldn't create the table %s, err: %s", osAgentVolumesTableName, err)
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

// InsertDynakube inserts a new Dynakube
func (a *SqliteAccess) InsertDynakube(dynakube *Dynakube) error {
	err := a.executeStatement(insertDynakubeStatement, dynakube.Name, dynakube.TenantUUID, dynakube.LatestVersion)
	if err != nil {
		err = fmt.Errorf("couldn't insert dynakube entry, tenantUUID '%s', latest version '%s', dynakube '%s', err: %s",
			dynakube.TenantUUID,
			dynakube.LatestVersion,
			dynakube.Name,
			err)
	}
	return err
}

// UpdateDynakube updates an existing Dynakube by matching the name
func (a *SqliteAccess) UpdateDynakube(dynakube *Dynakube) error {
	err := a.executeStatement(updateDynakubeStatement, dynakube.LatestVersion, dynakube.TenantUUID, dynakube.Name)
	if err != nil {
		err = fmt.Errorf("couldn't update dynakube, tenantUUID '%s', latest version '%s', name '%s', err: %s",
			dynakube.TenantUUID,
			dynakube.LatestVersion,
			dynakube.Name,
			err)
	}
	return err
}

// DeleteDynakube deletes an existing Dynakube using its name
func (a *SqliteAccess) DeleteDynakube(dynakubeName string) error {
	err := a.executeStatement(deleteDynakubeStatement, dynakubeName)
	if err != nil {
		err = fmt.Errorf("couldn't delete dynakube, name '%s', err: %s", dynakubeName, err)
	}
	return err
}

// GetDynakube gets Dynakube using its name
func (a *SqliteAccess) GetDynakube(dynakubeName string) (*Dynakube, error) {
	var tenantUUID string
	var latestVersion string
	err := a.querySimpleStatement(getDynakubeStatement, dynakubeName, &tenantUUID, &latestVersion)
	if err != nil {
		err = fmt.Errorf("couldn't get dynakube, name '%s', err: %s", dynakubeName, err)
	}
	return NewDynakube(dynakubeName, tenantUUID, latestVersion), err
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
	var podName string
	var version string
	var tenantUUID string
	err := a.querySimpleStatement(getVolumeStatement, volumeID, &podName, &version, &tenantUUID)
	if err != nil {
		err = fmt.Errorf("couldn't get volume field for volume id '%s', err: %s", volumeID, err)
	}
	return NewVolume(volumeID, podName, version, tenantUUID), err
}

// DeleteVolume deletes a Volume by its ID
func (a *SqliteAccess) DeleteVolume(volumeID string) error {
	err := a.executeStatement(deleteVolumeStatement, volumeID)
	if err != nil {
		err = fmt.Errorf("couldn't delete volume for volume id '%s', err: %s", volumeID, err)
	}
	return err
}

// InsertOsAgentVolume inserts a new OsAgentVolume
func (a *SqliteAccess) InsertOsAgentVolume(volume *OsAgentVolume) error {
	err := a.executeStatement(insertOsAgentVolumeStatement, volume.TenantUUID, volume.VolumeID, volume.Mounted, volume.LastModified)
	if err != nil {
		err = fmt.Errorf("couldn't insert osAgentVolume info, volume id '%s', tenant UUID '%s', mounted '%t', last modified '%s', err: %s",
			volume.VolumeID,
			volume.TenantUUID,
			volume.Mounted,
			volume.LastModified,
			err)
	}
	return err
}

// UpdateOsAgentVolume updates an existing OsAgentVolume by matching the tenantUUID
func (a *SqliteAccess) UpdateOsAgentVolume(volume *OsAgentVolume) error {
	err := a.executeStatement(updateOsAgentVolumeStatement, volume.VolumeID, volume.Mounted, volume.LastModified, volume.TenantUUID)
	if err != nil {
		err = fmt.Errorf("couldn't update osAgentVolume info, tenantUUID '%s', mounted '%t', last modified '%s', volume id %s, err: %s",
			volume.TenantUUID,
			volume.Mounted,
			volume.LastModified,
			volume.VolumeID,
			err)
	}
	return err
}

// GetOsAgentVolumeViaVolumeID gets an OsAgentVolume by its VolumeID
func (a *SqliteAccess) GetOsAgentVolumeViaVolumeID(volumeID string) (*OsAgentVolume, error) {
	var tenantUUID string
	var mounted bool
	var lastModified time.Time
	err := a.querySimpleStatement(getOsAgentVolumeViaVolumeIDStatement, volumeID, &tenantUUID, &mounted, &lastModified)
	if err != nil {
		err = fmt.Errorf("couldn't get osAgentVolume info for volume id '%s', err: %s", volumeID, err)
	}
	return NewOsAgentVolume(volumeID, tenantUUID, mounted, &lastModified), err
}

// GetOsAgentVolumeViaTenantUUID gets an OsAgentVolume by its tenantUUID
func (a *SqliteAccess) GetOsAgentVolumeViaTenantUUID(tenantUUID string) (*OsAgentVolume, error) {
	var volumeID string
	var mounted bool
	var lastModified time.Time
	err := a.querySimpleStatement(getOsAgentVolumeViaTenantUUIDStatement, tenantUUID, &volumeID, &mounted, &lastModified)
	if err != nil {
		err = fmt.Errorf("couldn't get osAgentVolume info for tenant uuid '%s', err: %s", tenantUUID, err)
	}
	return NewOsAgentVolume(volumeID, tenantUUID, mounted, &lastModified), err
}

// GetAllVolumes gets all the Volumes from the database
func (a *SqliteAccess) GetAllVolumes() ([]*Volume, error) {
	rows, err := a.conn.Query(dumpVolumesStatement)
	if err != nil {
		return nil, fmt.Errorf("couldn't get all the volumes, err: %s", err)
	}
	volumes := []*Volume{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		var podName string
		var version string
		var tenantUUID string
		err := rows.Scan(&id, &podName, &version, &tenantUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan from database for volumes, err: %s", err)
		}
		volumes = append(volumes, NewVolume(id, podName, version, tenantUUID))
	}
	return volumes, nil
}

// GetAllDynakubes gets all the Dynakubes from the database
func (a *SqliteAccess) GetAllDynakubes() ([]*Dynakube, error) {
	rows, err := a.conn.Query(dumpDynakubesStatement)
	if err != nil {
		return nil, fmt.Errorf("couldn't get all the volumes, err: %s", err)
	}
	dynakubes := []*Dynakube{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		var version string
		var tenantUUID string
		err := rows.Scan(&name, &tenantUUID, &version)
		if err != nil {
			return nil, fmt.Errorf("failed to scan from database for volumes, err: %s", err)
		}
		dynakubes = append(dynakubes, NewDynakube(name, tenantUUID, version))
	}
	return dynakubes, nil
}

// GetAllOsAgentVolumes gets all the OsAgentVolume from the database
func (a *SqliteAccess) GetAllOsAgentVolumes() ([]*OsAgentVolume, error) {
	rows, err := a.conn.Query(dumpOsAgentVolumes)
	if err != nil {
		return nil, fmt.Errorf("couldn't get all the volumes, err: %s", err)
	}
	osVolumes := []*OsAgentVolume{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var volumeID string
		var tenantUUID string
		var mounted bool
		var timeStamp time.Time
		err := rows.Scan(&tenantUUID, &volumeID, &mounted, &timeStamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan from database for volumes, err: %s", err)
		}
		osVolumes = append(osVolumes, NewOsAgentVolume(volumeID, tenantUUID, mounted, &timeStamp))
	}
	return osVolumes, nil
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

// GetDynakubes gets all Dynakubes and maps their name to the corresponding TenantUUID.
func (a *SqliteAccess) GetDynakubes() (map[string]string, error) {
	rows, err := a.conn.Query(getDynakubesStatement)
	if err != nil {
		return nil, fmt.Errorf("couldn't get dynakubes, err: %s", err)
	}
	dynakubes := map[string]string{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var uuid string
		var dynakube string
		err := rows.Scan(&uuid, &dynakube)
		if err != nil {
			return nil, fmt.Errorf("failed to scan from database for dynakube, err: %s", err)
		}
		dynakubes[dynakube] = uuid
	}
	return dynakubes, nil
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
