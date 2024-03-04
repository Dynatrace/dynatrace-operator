package metadata

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	dynakubesAlterStatementMaxFailedMountAttempts = `
	ALTER TABLE dynakubes
	ADD COLUMN MaxFailedMountAttempts INT NOT NULL DEFAULT ` + strconv.FormatInt(dynatracev1beta1.DefaultMaxFailedCsiMountAttempts, 10) + ";"
	// "Not null"-columns need a default value set
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

	volumesAlterStatementMountAttempts = `
	ALTER TABLE volumes
	ADD COLUMN MountAttempts INT NOT NULL DEFAULT 0;`

	// INSERT
	insertDynakubeStatement = `
	INSERT INTO dynakubes (Name, TenantUUID, LatestVersion, ImageDigest, MaxFailedMountAttempts)
	VALUES (?,?,?,?, ?);
	`

	insertVolumeStatement = `
	INSERT INTO volumes (ID, PodName, Version, TenantUUID, MountAttempts)
	VALUES (?,?,?,?,?)
	ON CONFLICT(ID) DO UPDATE SET
	  PodName=excluded.PodName,
	  Version=excluded.Version,
	  TenantUUID=excluded.TenantUUID,
  	  MountAttempts=excluded.MountAttempts;
	`

	insertOsAgentVolumeStatement = `
	INSERT INTO osagent_volumes (TenantUUID, VolumeID, Mounted, LastModified)
	VALUES (?,?,?,?);
	`

	// UPDATE
	updateDynakubeStatement = `
	UPDATE dynakubes
	SET LatestVersion = ?, TenantUUID = ?, ImageDigest = ?, MaxFailedMountAttempts = ?
	WHERE Name = ?;
	`

	updateOsAgentVolumeStatement = `
	UPDATE osagent_volumes
	SET VolumeID = ?, Mounted = ?, LastModified = ?
	WHERE TenantUUID = ?;
	`

	// GET
	getDynakubeStatement = `
	SELECT TenantUUID, LatestVersion, ImageDigest, MaxFailedMountAttempts
	FROM dynakubes
	WHERE Name = ?;
	`

	getVolumeStatement = `
	SELECT PodName, Version, TenantUUID, MountAttempts
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
		SELECT Name, TenantUUID, LatestVersion, ImageDigest, MaxFailedMountAttempts
		FROM dynakubes;
		`

	getAllVolumesStatement = `
		SELECT ID, PodName, Version, TenantUUID, MountAttempts
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
	SELECT DISTINCT Version
	FROM volumes
	WHERE TenantUUID = ?;
	`

	getAllUsedVersionsStatement = `
	SELECT DISTINCT Version
	FROM volumes;
	`

	getUsedImageDigestStatement = `
	SELECT DISTINCT ImageDigest
	FROM dynakubes
	WHERE ImageDigest != "";
	`

	getLatestVersionsStatement = `
	SELECT DISTINCT LatestVersion
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

	countImageDigestStatement = `
	SELECT COUNT(*)
	FROM dynakubes
	WHERE ImageDigest = ?;
	`
)

type SqliteAccess struct {
	conn *sql.DB
}

type DBConn struct {
	db *gorm.DB
}

// NewDBAccess creates a new gorm db connection to the database.
func NewDBAccess(path string) (DBConn, error) {
	// we should explicitly enable foreign_keys for sqlite
	if strings.Contains(path, "?") {
		path += "&_foreign_keys=on"
	} else {
		path += "?_foreign_keys=on"
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default})

	if err != nil {
		return DBConn{}, err
	}

	return DBConn{db: db}, nil
}

// NewAccess creates a new SqliteAccess, connects to the database.
func NewAccess(ctx context.Context, path string) (Access, error) {
	access := SqliteAccess{}

	err := access.Setup(ctx, path)
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

func (access *SqliteAccess) createTables(ctx context.Context) error {
	err := access.setupDynakubeTable(ctx)
	if err != nil {
		return err
	}

	err = access.setupVolumeTable(ctx)
	if err != nil {
		return err
	}

	if _, err := access.conn.ExecContext(ctx, osAgentVolumesCreateStatement); err != nil {
		return errors.WithStack(errors.WithMessagef(err, "couldn't create the table %s", osAgentVolumesTableName))
	}

	return nil
}

func (access *SqliteAccess) setupVolumeTable(ctx context.Context) error {
	_, err := access.conn.Exec(volumesCreateStatement)
	if err != nil {
		return errors.WithMessagef(err, "couldn't create the table %s", volumesTableName)
	}

	err = access.executeAlterStatement(ctx, volumesAlterStatementMountAttempts)
	if err != nil {
		return err
	}

	return nil
}

// setupDynakubeTable creates the dynakubes table if it doesn't exist and tries to add additional columns
func (access *SqliteAccess) setupDynakubeTable(ctx context.Context) error {
	if _, err := access.conn.Exec(dynakubesCreateStatement); err != nil {
		return errors.WithStack(errors.WithMessagef(err, "couldn't create the table %s", dynakubesTableName))
	}

	err := access.executeAlterStatement(ctx, dynakubesAlterStatementImageDigestColumn)
	if err != nil {
		return err
	}

	err = access.executeAlterStatement(ctx, dynakubesAlterStatementMaxFailedMountAttempts)
	if err != nil {
		return err
	}

	return nil
}

func (access *SqliteAccess) executeAlterStatement(ctx context.Context, statement string) error {
	if _, err := access.conn.ExecContext(ctx, statement); err != nil {
		sqliteErr := sqlite3.Error{}
		isSqliteErr := errors.As(err, &sqliteErr)

		if isSqliteErr && sqliteErr.Code != sqlite3.ErrError {
			return errors.WithStack(err)
		}
	}

	return nil
}

// Setup connects to the database and creates the necessary tables if they don't exist
func (access *SqliteAccess) Setup(ctx context.Context, path string) error {
	if err := access.connect(sqliteDriverName, path); err != nil {
		return err
	}

	if err := access.createTables(ctx); err != nil {
		return err
	}

	return nil
}

// InsertDynakube inserts a new Dynakube
func (access *SqliteAccess) InsertDynakube(ctx context.Context, dynakube *Dynakube) error {
	err := access.executeStatement(ctx, insertDynakubeStatement, dynakube.Name, dynakube.TenantUUID, dynakube.LatestVersion, dynakube.ImageDigest, dynakube.MaxFailedMountAttempts)
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
func (access *SqliteAccess) UpdateDynakube(ctx context.Context, dynakube *Dynakube) error {
	err := access.executeStatement(ctx, updateDynakubeStatement, dynakube.LatestVersion, dynakube.TenantUUID, dynakube.ImageDigest, dynakube.MaxFailedMountAttempts, dynakube.Name)
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
func (access *SqliteAccess) DeleteDynakube(ctx context.Context, dynakubeName string) error {
	err := access.executeStatement(ctx, deleteDynakubeStatement, dynakubeName)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't delete dynakube, name '%s'", dynakubeName)
	}

	return err
}

// GetDynakube gets Dynakube using its name
func (access *SqliteAccess) GetDynakube(ctx context.Context, dynakubeName string) (*Dynakube, error) {
	var tenantUUID string

	var latestVersion string

	var imageDigest string

	var maxFailedMountAttempts int

	err := access.querySimpleStatement(ctx, getDynakubeStatement, dynakubeName, &tenantUUID, &latestVersion, &imageDigest, &maxFailedMountAttempts)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't get dynakube, name '%s'", dynakubeName)
	}

	return NewDynakube(dynakubeName, tenantUUID, latestVersion, imageDigest, maxFailedMountAttempts), err
}

// InsertVolume inserts a new Volume
func (access *SqliteAccess) InsertVolume(ctx context.Context, volume *Volume) error {
	err := access.executeStatement(ctx, insertVolumeStatement, volume.VolumeID, volume.PodName, volume.Version, volume.TenantUUID, volume.MountAttempts)
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
func (access *SqliteAccess) GetVolume(ctx context.Context, volumeID string) (*Volume, error) {
	var podName string

	var version string

	var tenantUUID string

	var mountAttempts int

	err := access.querySimpleStatement(ctx, getVolumeStatement, volumeID, &podName, &version, &tenantUUID, &mountAttempts)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't get volume field for volume id '%s'", volumeID)
	}

	return NewVolume(volumeID, podName, version, tenantUUID, mountAttempts), err
}

// DeleteVolume deletes a Volume by its ID
func (access *SqliteAccess) DeleteVolume(ctx context.Context, volumeID string) error {
	err := access.executeStatement(ctx, deleteVolumeStatement, volumeID)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't delete volume for volume id '%s'", volumeID)
	}

	return err
}

// InsertOsAgentVolume inserts a new OsAgentVolume
func (access *SqliteAccess) InsertOsAgentVolume(ctx context.Context, volume *OsAgentVolume) error {
	err := access.executeStatement(ctx, insertOsAgentVolumeStatement, volume.TenantUUID, volume.VolumeID, volume.Mounted, volume.LastModified)
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
func (access *SqliteAccess) UpdateOsAgentVolume(ctx context.Context, volume *OsAgentVolume) error {
	err := access.executeStatement(ctx, updateOsAgentVolumeStatement, volume.VolumeID, volume.Mounted, volume.LastModified, volume.TenantUUID)
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
func (access *SqliteAccess) GetOsAgentVolumeViaVolumeID(ctx context.Context, volumeID string) (*OsAgentVolume, error) {
	var tenantUUID string

	var mounted bool

	var lastModified time.Time

	err := access.querySimpleStatement(ctx, getOsAgentVolumeViaVolumeIDStatement, volumeID, &tenantUUID, &mounted, &lastModified)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't get osAgentVolume info for volume id '%s'", volumeID)
	}

	return NewOsAgentVolume(volumeID, tenantUUID, mounted, &lastModified), err
}

// GetOsAgentVolumeViaTenantUUID gets an OsAgentVolume by its tenantUUID
func (access *SqliteAccess) GetOsAgentVolumeViaTenantUUID(ctx context.Context, tenantUUID string) (*OsAgentVolume, error) {
	var volumeID string

	var mounted bool

	var lastModified time.Time

	err := access.querySimpleStatement(ctx, getOsAgentVolumeViaTenantUUIDStatement, tenantUUID, &volumeID, &mounted, &lastModified)
	if err != nil {
		err = errors.WithMessagef(err, "couldn't get osAgentVolume info for tenant uuid '%s'", tenantUUID)
	}

	return NewOsAgentVolume(volumeID, tenantUUID, mounted, &lastModified), err
}

// GetAllVolumes gets all the Volumes from the database
func (access *SqliteAccess) GetAllVolumes(ctx context.Context) ([]*Volume, error) {
	rows, err := access.conn.QueryContext(ctx, getAllVolumesStatement)
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

		var mountAttempts int

		err := rows.Scan(&id, &podName, &version, &tenantUUID, &mountAttempts)
		if err != nil {
			return nil, errors.WithStack(errors.WithMessage(err, "couldn't scan volume from database"))
		}

		volumes = append(volumes, NewVolume(id, podName, version, tenantUUID, mountAttempts))
	}

	return volumes, nil
}

// GetAllDynakubes gets all the Dynakubes from the database
func (access *SqliteAccess) GetAllDynakubes(ctx context.Context) ([]*Dynakube, error) {
	rows, err := access.conn.QueryContext(ctx, getAllDynakubesStatement)
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

		var maxFailedMountAttempts int

		err := rows.Scan(&name, &tenantUUID, &version, &imageDigest, &maxFailedMountAttempts)
		if err != nil {
			return nil, errors.WithStack(errors.WithMessage(err, "couldn't scan dynakube from database"))
		}

		dynakubes = append(dynakubes, NewDynakube(name, tenantUUID, version, imageDigest, maxFailedMountAttempts))
	}

	return dynakubes, nil
}

// GetAllOsAgentVolumes gets all the OsAgentVolume from the database
func (access *SqliteAccess) GetAllOsAgentVolumes(ctx context.Context) ([]*OsAgentVolume, error) {
	rows, err := access.conn.QueryContext(ctx, getAllOsAgentVolumes)
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

// GetUsedVersions gets all UNIQUE versions present in the `volumes` for a given tenantUUID database in map.
// Map is used to make sure we don't return the same version multiple time,
// it's also easier to check if a version is in it or not. (a Set in style of Golang)
func (access *SqliteAccess) GetUsedVersions(ctx context.Context, tenantUUID string) (map[string]bool, error) {
	rows, err := access.conn.QueryContext(ctx, getUsedVersionsStatement, tenantUUID)
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

		versions[version] = true
	}

	return versions, nil
}

// GetUsedVersions gets all UNIQUE versions present in the `volumes` database in map.
// Map is used to make sure we don't return the same version multiple time,
// it's also easier to check if a version is in it or not. (a Set in style of Golang)
func (access *SqliteAccess) GetAllUsedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := access.conn.QueryContext(ctx, getAllUsedVersionsStatement)
	if err != nil {
		return nil, errors.WithStack(errors.WithMessagef(err, "couldn't get all used version info"))
	}

	versions := map[string]bool{}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var version string

		err := rows.Scan(&version)
		if err != nil {
			return nil, errors.WithStack(errors.WithMessagef(err, "couldn't scan used version info"))
		}

		if _, ok := versions[version]; !ok {
			versions[version] = true
		}
	}

	return versions, nil
}

// GetLatestVersions gets all UNIQUE latestVersions present in the `dynakubes` database in map.
// Map is used to make sure we don't return the same version multiple time,
// it's also easier to check if a version is in it or not. (a Set in style of Golang)
func (access *SqliteAccess) GetLatestVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := access.conn.QueryContext(ctx, getLatestVersionsStatement)
	if err != nil {
		return nil, errors.WithStack(errors.WithMessage(err, "couldn't get all the latests version info for tenant uuid"))
	}

	versions := map[string]bool{}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var version string

		err := rows.Scan(&version)
		if err != nil {
			return nil, errors.WithStack(errors.WithMessage(err, "couldn't scan latest version info "))
		}

		versions[version] = true
	}

	return versions, nil
}

// GetUsedImageDigests gets all UNIQUE image digests present in the `dynakubes` database in a map.
// Map is used to make sure we don't return the same digest multiple time,
// it's also easier to check if a digest is in it or not. (a Set in style of Golang)
func (access *SqliteAccess) GetUsedImageDigests(ctx context.Context) (map[string]bool, error) {
	rows, err := access.conn.QueryContext(ctx, getUsedImageDigestStatement)
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

// IsImageDigestUsed checks if the specified image digest is present in the database.
func (access *SqliteAccess) IsImageDigestUsed(ctx context.Context, imageDigest string) (bool, error) {
	var count int

	err := access.querySimpleStatement(ctx, countImageDigestStatement, imageDigest, &count)
	if err != nil {
		return false, errors.WithMessagef(err, "couldn't count usage of image digest: %s", imageDigest)
	}

	return count > 0, nil
}

// GetPodNames gets all PodNames present in the `volumes` database in map with their corresponding volumeIDs.
func (access *SqliteAccess) GetPodNames(ctx context.Context) (map[string]string, error) {
	rows, err := access.conn.QueryContext(ctx, getPodNamesStatement)
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
func (access *SqliteAccess) GetTenantsToDynakubes(ctx context.Context) (map[string]string, error) {
	rows, err := access.conn.QueryContext(ctx, getTenantsToDynakubesStatement)
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
func (access *SqliteAccess) executeStatement(ctx context.Context, statement string, vars ...any) error {
	_, err := access.conn.ExecContext(ctx, statement, vars...)

	return errors.WithStack(err)
}

// Executes the provided SQL SELECT statement on the database.
// The SQL statement should always return a single row.
// The `id` is passed to the SQL query to fill in an SQL wildcard
// The `vars` are filled with the values of the return of the SELECT statement, so the `vars` need to be pointers.
func (access *SqliteAccess) querySimpleStatement(ctx context.Context, statement, id string, vars ...any) error {
	row := access.conn.QueryRowContext(ctx, statement, id)

	err := row.Scan(vars...)
	if err != nil && err != sql.ErrNoRows {
		return errors.WithStack(err)
	}

	return nil
}

// SchemaMigration runs gormigrate migrations to create tables
func (conn *DBConn) SchemaMigration(ctx context.Context) error {
	m := gormigrate.New(conn.db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// your migrations here
	})

	m.InitSchema(func(tx *gorm.DB) error {
		err := tx.AutoMigrate(
			&TenantConfig{},
			&CodeModule{},
			&OSMount{},
			&AppMount{},
			&VolumeMeta{},
		)
		if err != nil {
			return err
		}

		// all other constraints, indexes, etc...
		return nil
	})

	return m.Migrate()
}
