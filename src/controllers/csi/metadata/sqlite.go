package metadata

import (
	"database/sql"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
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

	// ALTER
	dynakubesAlterStatementImageDigestColumn = `
	ALTER TABLE dynakubes
	ADD COLUMN ImageDigest VARCHAR NOT NULL DEFAULT '';
	`

	// INSERT
	insertDynakubeStatement = `
	INSERT INTO dynakubes (Name, TenantUUID, LatestVersion, ImageDigest)
	VALUES (?,?,?,?);
	`

	insertVolumeStatement = `
	INSERT INTO volumes (ID, PodName, Version, TenantUUID)
	VALUES (?,?,?,?)
	ON CONFLICT(ID) DO UPDATE SET
	  PodName=excluded.PodName,
	  Version=excluded.Version,
	  TenantUUID=excluded.TenantUUID;
	`

	insertOsAgentVolumeStatement = `
	INSERT INTO osagent_volumes (TenantUUID, VolumeID, Mounted, LastModified)
	VALUES (?,?,?,?);
	`

	// UPDATE
	updateDynakubeStatement = `
	UPDATE dynakubes
	SET LatestVersion = ?, TenantUUID = ?, ImageDigest = ?
	WHERE Name = ?;
	`

	updateOsAgentVolumeStatement = `
	UPDATE osagent_volumes
	SET VolumeID = ?, Mounted = ?, LastModified = ?
	WHERE TenantUUID = ?;
	`

	// GET
	getDynakubeStatement = `
	SELECT TenantUUID, LatestVersion, ImageDigest
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

	// GET ALL
	getAllDynakubesStatement = `
		SELECT Name, TenantUUID, LatestVersion, ImageDigest
		FROM dynakubes;
		`

	getAllVolumesStatement = `
		SELECT ID, PodName, Version, TenantUUID
		FROM volumes;
		`

	getAllOsAgentVolumes = `
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

	getUsedImageDigestStatement = `
	SELECT ImageDigest
	FROM dynakubes;
	`

	getPodNamesStatement = `
	SELECT ID, PodName
	FROM volumes;
	`

	getTenantsToDynakubesStatement = `
	SELECT tenantUUID, Name
	FROM dynakubes;
	`
)

type SqliteAccess struct {
	conn *sql.DB
}

// NewAccess creates a new SqliteAccess, connects to the database.
func NewAccess(path string) (Access, error) {
	access := SqliteAccess{}
	err := access.Setup(path)
	if err != nil {
		log.Error(err, "failed to connect to the database")
		return nil, err
	}
	return &access, nil
}

func (access *SqliteAccess) connect(driver, path string) error {
	db, err := sql.Open(driver, path)
	if err != nil {
		err := errors.WithStack(errors.WithMessagef(err, "couldn't connect to db %s", path))
		access.conn = nil
		return err
	}
	access.conn = db
	return nil
}

func (access *SqliteAccess) createTables() error {
	if err := access.setupDynakubeTable(); err != nil {
		return err
	}

	if _, err := access.conn.Exec(volumesCreateStatement); err != nil {
		return errors.WithStack(errors.WithMessagef(err, "couldn't create the table %s", volumesTableName))
	}
	if _, err := access.conn.Exec(osAgentVolumesCreateStatement); err != nil {
		return errors.WithStack(errors.WithMessagef(err, "couldn't create the table %s", osAgentVolumesTableName))
	}
	return nil
}

// setupDynakubeTable creates the dynakubes table if it doesn't exist and tries to add additional columns
func (access *SqliteAccess) setupDynakubeTable() error {
	if _, err := access.conn.Exec(dynakubesCreateStatement); err != nil {
		return errors.WithStack(errors.WithMessagef(err, "couldn't create the table %s", dynakubesTableName))
	}

	if _, err := access.conn.Exec(dynakubesAlterStatementImageDigestColumn); err != nil {
		sqliteError := err.(sqlite3.Error)
		if sqliteError.Code != sqlite3.ErrError {
			return errors.WithStack(errors.WithMessage(err, "couldn't add ingest column"))
		}
		// generic sql error, column already exists
		log.Info("column ImageDigest already exists")
	}
	return nil
}

// Setup connects to the database and creates the necessary tables if they don't exist
func (access *SqliteAccess) Setup(path string) error {
	if err := access.connect(sqliteDriverName, path); err != nil {
		return err
	}
	if err := access.createTables(); err != nil {
		return err
	}
	return nil
}

// InsertDynakube inserts a new Dynakube
func (access *SqliteAccess) InsertDynakube(dynakube *Dynakube) error {
	err := access.executeStatement(insertDynakubeStatement, dynakube.Name, dynakube.TenantUUID, dynakube.LatestVersion, dynakube.ImageDigest)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't insert dynakube entry, tenantUUID '%s', latest version '%s', name '%s', image digest '%s'",
			dynakube.TenantUUID,
			dynakube.LatestVersion,
			dynakube.Name,
			dynakube.ImageDigest)
	}
	return err
}

// UpdateDynakube updates an existing Dynakube by matching the name
func (access *SqliteAccess) UpdateDynakube(dynakube *Dynakube) error {
	err := access.executeStatement(updateDynakubeStatement, dynakube.LatestVersion, dynakube.TenantUUID, dynakube.ImageDigest, dynakube.Name)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't update dynakube, tenantUUID '%s', latest version '%s', name '%s', image digest '%s'",
			dynakube.TenantUUID,
			dynakube.LatestVersion,
			dynakube.Name,
			dynakube.ImageDigest)
	}
	return err
}

// DeleteDynakube deletes an existing Dynakube using its name
func (access *SqliteAccess) DeleteDynakube(dynakubeName string) error {
	err := access.executeStatement(deleteDynakubeStatement, dynakubeName)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't delete dynakube, name '%s'", dynakubeName)
	}
	return err
}

// GetDynakube gets Dynakube using its name
func (access *SqliteAccess) GetDynakube(dynakubeName string) (*Dynakube, error) {
	var tenantUUID string
	var latestVersion string
	var imageDigest string
	err := access.querySimpleStatement(getDynakubeStatement, dynakubeName, &tenantUUID, &latestVersion, &imageDigest)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't get dynakube, name '%s'", dynakubeName)
	}
	return NewDynakube(dynakubeName, tenantUUID, latestVersion, imageDigest), err
}

// InsertVolume inserts a new Volume
func (access *SqliteAccess) InsertVolume(volume *Volume) error {
	err := access.executeStatement(insertVolumeStatement, volume.VolumeID, volume.PodName, volume.Version, volume.TenantUUID)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't insert volume info, volume id '%s', pod '%s', version '%s', dynakube '%s'",
			volume.VolumeID,
			volume.PodName,
			volume.Version,
			volume.TenantUUID)
	}
	return err
}

// GetVolume gets Volume by its ID
func (access *SqliteAccess) GetVolume(volumeID string) (*Volume, error) {
	var podName string
	var version string
	var tenantUUID string
	err := access.querySimpleStatement(getVolumeStatement, volumeID, &podName, &version, &tenantUUID)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't get volume field for volume id '%s'", volumeID)
	}
	return NewVolume(volumeID, podName, version, tenantUUID), err
}

// DeleteVolume deletes a Volume by its ID
func (access *SqliteAccess) DeleteVolume(volumeID string) error {
	err := access.executeStatement(deleteVolumeStatement, volumeID)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't delete volume for volume id '%s'", volumeID)
	}
	return err
}

// InsertOsAgentVolume inserts a new OsAgentVolume
func (access *SqliteAccess) InsertOsAgentVolume(volume *OsAgentVolume) error {
	err := access.executeStatement(insertOsAgentVolumeStatement, volume.TenantUUID, volume.VolumeID, volume.Mounted, volume.LastModified)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't insert osAgentVolume info, volume id '%s', tenant UUID '%s', mounted '%t', last modified '%s'",
			volume.VolumeID,
			volume.TenantUUID,
			volume.Mounted,
			volume.LastModified)
	}
	return err
}

// UpdateOsAgentVolume updates an existing OsAgentVolume by matching the tenantUUID
func (access *SqliteAccess) UpdateOsAgentVolume(volume *OsAgentVolume) error {
	err := access.executeStatement(updateOsAgentVolumeStatement, volume.VolumeID, volume.Mounted, volume.LastModified, volume.TenantUUID)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't update osAgentVolume info, tenantUUID '%s', mounted '%t', last modified '%s', volume id '%s'",
			volume.TenantUUID,
			volume.Mounted,
			volume.LastModified,
			volume.VolumeID)
	}
	return err
}

// GetOsAgentVolumeViaVolumeID gets an OsAgentVolume by its VolumeID
func (access *SqliteAccess) GetOsAgentVolumeViaVolumeID(volumeID string) (*OsAgentVolume, error) {
	var tenantUUID string
	var mounted bool
	var lastModified time.Time
	err := access.querySimpleStatement(getOsAgentVolumeViaVolumeIDStatement, volumeID, &tenantUUID, &mounted, &lastModified)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't get osAgentVolume info for volume id '%s'", volumeID)
	}
	return NewOsAgentVolume(volumeID, tenantUUID, mounted, &lastModified), err
}

// GetOsAgentVolumeViaTenantUUID gets an OsAgentVolume by its tenantUUID
func (access *SqliteAccess) GetOsAgentVolumeViaTenantUUID(tenantUUID string) (*OsAgentVolume, error) {
	var volumeID string
	var mounted bool
	var lastModified time.Time
	err := access.querySimpleStatement(getOsAgentVolumeViaTenantUUIDStatement, tenantUUID, &volumeID, &mounted, &lastModified)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't get osAgentVolume info for tenant uuid '%s'", tenantUUID)
	}
	return NewOsAgentVolume(volumeID, tenantUUID, mounted, &lastModified), err
}

// GetAllVolumes gets all the Volumes from the database
func (access *SqliteAccess) GetAllVolumes() ([]*Volume, error) {
	rows, err := access.conn.Query(getAllVolumesStatement)
	if err != nil {
		return nil, errors.WithStack(errors.WithMessage(err, "couldn't get all the volumes"))
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
			return nil, errors.WithStack(errors.WithMessage(err, "couldn't scan volume from database"))
		}
		volumes = append(volumes, NewVolume(id, podName, version, tenantUUID))
	}
	return volumes, nil
}

// GetAllDynakubes gets all the Dynakubes from the database
func (access *SqliteAccess) GetAllDynakubes() ([]*Dynakube, error) {
	rows, err := access.conn.Query(getAllDynakubesStatement)
	if err != nil {
		return nil, errors.WithStack(errors.WithMessage(err, "couldn't get all the dynakubes"))
	}
	dynakubes := []*Dynakube{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		var version string
		var tenantUUID string
		var imageDigest string
		err := rows.Scan(&name, &tenantUUID, &version, &imageDigest)
		if err != nil {
			return nil, errors.WithStack(errors.WithMessage(err, "couldn't scan dynakube from database"))
		}
		dynakubes = append(dynakubes, NewDynakube(name, tenantUUID, version, imageDigest))
	}
	return dynakubes, nil
}

// GetAllOsAgentVolumes gets all the OsAgentVolume from the database
func (access *SqliteAccess) GetAllOsAgentVolumes() ([]*OsAgentVolume, error) {
	rows, err := access.conn.Query(getAllOsAgentVolumes)
	if err != nil {
		return nil, errors.WithStack(errors.WithMessage(err, "couldn't get all the osagent volumes"))
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
			return nil, errors.WithStack(errors.WithMessage(err, "couldn't scan osagent volume from database"))
		}
		osVolumes = append(osVolumes, NewOsAgentVolume(volumeID, tenantUUID, mounted, &timeStamp))
	}
	return osVolumes, nil
}

// GetUsedVersions gets all UNIQUE versions present in the `volumes` database in map.
// Map is used to make sure we don't return the same version multiple time,
// it's also easier to check if a version is in it or not. (a Set in style of Golang)
func (access *SqliteAccess) GetUsedVersions(tenantUUID string) (map[string]bool, error) {
	rows, err := access.conn.Query(getUsedVersionsStatement, tenantUUID)
	if err != nil {
		return nil, errors.WithStack(errors.WithMessagef(err, "couldn't get used version info for tenant uuid '%s'", tenantUUID))
	}
	versions := map[string]bool{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var version string
		err := rows.Scan(&version)
		if err != nil {
			return nil, errors.WithStack(errors.WithMessagef(err, "couldn't scan used version info for tenant uuid '%s'", tenantUUID))
		}
		if _, ok := versions[version]; !ok {
			versions[version] = true
		}
	}
	return versions, nil
}

// GetUsedImageDigests gets all UNIQUE image digests present in the `dynakubes` database in a map.
// Map is used to make sure we don't return the same digest multiple time,
// it's also easier to check if a digest is in it or not. (a Set in style of Golang)
func (access *SqliteAccess) GetUsedImageDigests() (map[string]bool, error) {
	rows, err := access.conn.Query(getUsedImageDigestStatement)
	if err != nil {
		return nil, errors.WithStack(errors.WithMessage(err, "couldn't get used image digests from database"))
	}
	imageDigests := map[string]bool{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var digest string
		err := rows.Scan(&digest)
		if err != nil {
			return nil, errors.WithStack(errors.WithMessage(err, "failed to scan from image digests database"))
		}
		if _, ok := imageDigests[digest]; !ok {
			imageDigests[digest] = true
		}
	}
	return imageDigests, nil
}

// GetPodNames gets all PodNames present in the `volumes` database in map with their corresponding volumeIDs.
func (access *SqliteAccess) GetPodNames() (map[string]string, error) {
	rows, err := access.conn.Query(getPodNamesStatement)
	if err != nil {
		return nil, errors.WithStack(errors.WithMessage(err, "couldn't get all pod names"))
	}
	podNames := map[string]string{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var podName string
		var volumeID string
		err := rows.Scan(&volumeID, &podName)
		if err != nil {
			return nil, errors.WithStack(errors.WithMessage(err, "couldn't scan pod name from database"))
		}
		podNames[podName] = volumeID
	}
	return podNames, nil
}

// GetTenantsToDynakubes gets all Dynakubes and maps their name to the corresponding TenantUUID.
func (access *SqliteAccess) GetTenantsToDynakubes() (map[string]string, error) {
	rows, err := access.conn.Query(getTenantsToDynakubesStatement)
	if err != nil {
		return nil, errors.WithStack(errors.WithMessage(err, "couldn't get all tenants to dynakube metadata"))
	}
	dynakubes := map[string]string{}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var uuid string
		var dynakube string
		err := rows.Scan(&uuid, &dynakube)
		if err != nil {
			return nil, errors.WithStack(errors.WithMessage(err, "couldn't scan tenant to dynakube metadata from database"))
		}
		dynakubes[dynakube] = uuid
	}
	return dynakubes, nil
}

// Executes the provided SQL statement on the database.
// The `vars` are passed to the SQL statement (in-order), to fill in the SQL wildcards.
func (access *SqliteAccess) executeStatement(statement string, vars ...interface{}) error {
	_, err := access.conn.Exec(statement, vars...)
	return errors.WithStack(err)
}

// Executes the provided SQL SELECT statement on the database.
// The SQL statement should always return a single row.
// The `id` is passed to the SQL query to fill in an SQL wildcard
// The `vars` are filled with the values of the return of the SELECT statement, so the `vars` need to be pointers.
func (access *SqliteAccess) querySimpleStatement(statement, id string, vars ...interface{}) error {
	row := access.conn.QueryRow(statement, id)
	err := row.Scan(vars...)
	if err != nil && err != sql.ErrNoRows {
		return errors.WithStack(err)
	}
	return nil
}
