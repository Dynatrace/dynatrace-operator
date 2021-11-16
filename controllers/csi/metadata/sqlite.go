package metadata

import (
	"database/sql"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/logger"
	_ "github.com/mattn/go-sqlite3"
)

const (
	sqliteDriverName = "sqlite3"

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

	ruxitRevTableName       = "ruxitRevissions"
	ruxitRevCreateStatement = `
	CREATE TABLE IF NOT EXISTS ruxitRevissions (
		TenantUUID VARCHAR NOT NULL,
		LatestRevision VARCHAR NOT NULL,
		PRIMARY KEY (TenantUUID)
	);`

	insertDynakubeStatement = `
	INSERT INTO dynakubes (Name, TenantUUID, LatestVersion)
	VALUES (?,?,?);
	`
	updateDynakubeStatement = `
	UPDATE dynakubes
	SET LatestVersion = ?, TenantUUID = ?
	WHERE Name = ?;
	`

	getDynakubeStatement = `
	SELECT TenantUUID, LatestVersion
	FROM dynakubes
	WHERE Name = ?;
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

	insertRuxitRevissionStatement = `
	INSERT INTO ruxitRevissions (TenantUUID, LatestRevision)
	VALUES (?,?);
	`
	updateRuxitRevissionStatement = `
	UPDATE ruxitRevissions
	SET LatestRevision = ?
	WHERE TenantUUID = ?;
	`

	getRuxitRevissionStatement = `
	SELECT LatestRevision
	FROM ruxitRevissions
	WHERE TenantUUID = ?;
	`

	ruxitRevisionPrunerTriggerName      = "revisionPruner"
	ruxitRevisionPrunerTriggerStatement = `
	CREATE TRIGGER IF NOT EXISTS revisionPruner
		AFTER DELETE ON dynakubes
		WHEN NOT EXISTS (SELECT 1 FROM dynakubes WHERE TenantUUID = OLD.TenantUUID)
	BEGIN
	    DELETE FROM ruxitRevissions WHERE TenantUUID = OLD.TenantUUID;
	END;
	`

	deleteVolumeStatement = "DELETE FROM volumes WHERE ID = ?;"

	deleteDynakubeStatement = "DELETE FROM dynakubes WHERE Name = ?;"

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
	if _, err := a.conn.Exec(dynakubesCreateStatement); err != nil {
		return fmt.Errorf("couldn't create the table %s, err: %s", dynakubesTableName, err)
	}
	if _, err := a.conn.Exec(volumesCreateStatement); err != nil {
		return fmt.Errorf("couldn't create the table %s, err: %s", volumesTableName, err)
	}
	if _, err := a.conn.Exec(ruxitRevCreateStatement); err != nil {
		return fmt.Errorf("couldn't create the table %s, err: %s", ruxitRevTableName, err)
	}
	if _, err := a.conn.Exec(ruxitRevisionPrunerTriggerStatement); err != nil {
		return fmt.Errorf("couldn't create the trigger for pruning table %s, err: %s", ruxitRevTableName, err)
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

// InsertRuxitRevission inserts a new RuxitRevission
func (a *SqliteAccess) InsertRuxitRevission(ruxitRev *RuxitRevision) error {
	err := a.executeStatement(insertRuxitRevissionStatement, ruxitRev.TenantUUID, ruxitRev.LatestRevission)
	if err != nil {
		err = fmt.Errorf("couldn't insert ruxitRevission, tenantUUID '%s', latestRevission '%d', err: %s",
			ruxitRev.TenantUUID,
			ruxitRev.LatestRevission,
			err)
	}
	return err
}

// UpdateRuxitRevission updates an existing RuxitRevission
func (a *SqliteAccess) UpdateRuxitRevission(ruxitRev *RuxitRevision) error {
	err := a.executeStatement(updateRuxitRevissionStatement, ruxitRev.LatestRevission, ruxitRev.TenantUUID)
	if err != nil {
		err = fmt.Errorf("couldn't update ruxitRevission, tenantUUID '%s', latestRevission '%d', err: %s",
			ruxitRev.TenantUUID,
			ruxitRev.LatestRevission,
			err)
	}
	return err
}

// GetRuxitRevission gets RuxitRevission by tenantUUID
func (a *SqliteAccess) GetRuxitRevission(tenantUUID string) (*RuxitRevision, error) {
	var revission uint
	err := a.querySimpleStatement(getRuxitRevissionStatement, tenantUUID, &revission)
	if err != nil {
		err = fmt.Errorf("couldn't get ruxitRevission, tenantUUID '%s', err: %s", tenantUUID, err)
	}
	return NewRuxitRevission(tenantUUID, revission), err
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
