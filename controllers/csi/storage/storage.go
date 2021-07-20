package storage

import (
	"database/sql"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/logger"
	_ "modernc.org/sqlite"
)

const (
	driverName = "sqlite"
	dbPath     = "./csi.db"

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
		UUID VARCHAR,
		VolumeId VARCHAR,
		Version VARCHAR,
		TenantUUID VARCHAR,
		PRIMARY KEY (UUID)
	);`

	latestVersionStatement = `
	SELECT LatestVersion
	FROM tenants
	WHERE UUID = ?;
	`

	updateLatestVersionStatement = `
	UPDATE tenants
	SET LatestVersion = ?
	WHERE UUID = ?;
	`

	getDynakubeStatement = `
	SELECT Dynakube
	FROM tenants
	WHERE UUID = ?;
	`

	updateDynakubeStatement = `
	UPDATE tenants
	SET Dynakube = ?
	WHERE UUID = ?;
	`

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

	getTenantUUIDStatement = `
	SELECT UUID
	FROM tenants
	WHERE Dynakube = ?;
	`
)

var (
	log = logger.NewDTLogger().WithName("provisioner")
)

type Tenant struct {
	UUID          string
	LatestVersion string
	Dynakube      string
}

type Pod struct {
	UUID       string
	VolumeId   string
	Version    string
	TenantUUID string
}

type Access struct {
	conn *sql.DB
}

func NewAccess() Access {
	a := Access{}
	err := a.init()
	if err != nil {
		log.Error(err, "Failed to init the database, err: %s", err.Error())
	}
	return a
}

func (a *Access) Connect(driver, path string) error {
	db, err := sql.Open(driver, path)
	if err != nil {
		err := fmt.Errorf("couldn't connect to db %s, err: %s", path, err)
		return err
	}
	a.conn = db
	return nil
}

func (a *Access) init() error {
	if err := a.Connect(driverName, dbPath); err != nil {
		return err
	}
	if err := a.createTables(); err != nil {
		return err
	}
	return nil
}

func (a *Access) createTables() error {
	if _, err := a.conn.Exec(tenantsCreateStatement); err != nil {
		err = fmt.Errorf("couldn't create the table %s, err: %s", tenantsTableName, err)
		log.Info(err.Error())
		return err
	}
	if _, err := a.conn.Exec(podsCreateStatement); err != nil {
		err = fmt.Errorf("couldn't create the table %s, err: %s", podsTableName, err)
		log.Info(err.Error())
		return err
	}
	return nil
}

func (a *Access) GetLatestVersion(tenantUUID string) (string, error) {
	var latestVersion string
	row := a.conn.QueryRow(latestVersionStatement, tenantUUID)
	err := row.Scan(&latestVersion)
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("couldn't get latest version for tenant %s, err: %s", tenantUUID, err)
		log.Info(err.Error())
		return "", err
	}
	return latestVersion, nil
}

func (a *Access) UpdateLatestVersion(tenantUUID, version string) error {
	_, err := a.conn.Exec(updateLatestVersionStatement, version, tenantUUID)
	if err != nil {
		err = fmt.Errorf("couldn't update latest version for tenant %s, err: %s", tenantUUID, err)
		log.Info(err.Error())
		return err
	}
	return nil
}

// Returns ("",nil) if not there
func (a *Access) GetDynaKube(tenantUUID string) (string, error) {
	var dk string
	row := a.conn.QueryRow(getDynakubeStatement, tenantUUID)
	err := row.Scan(&dk)
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("couldn't get Dynakube field for tenant %s, err: %s", tenantUUID, err)
		log.Info(err.Error())
		return "", err
	}
	return dk, nil
}

func (a *Access) UpdateDynaKube(tenantUUID, dynakube string) error {
	_, err := a.conn.Exec(updateDynakubeStatement, dynakube, tenantUUID)
	if err != nil {
		err = fmt.Errorf("couldn't update Dynakube field for tenant %s, err: %s", tenantUUID, err)
		log.Info(err.Error())
		return err
	}
	return nil
}

func (a *Access) InsertTenant(tenant *Tenant) error {
	_, err := a.conn.Exec(insertTenantStatement, tenant.UUID, tenant.LatestVersion, tenant.Dynakube)
	if err != nil {
		err = fmt.Errorf("couldn't update tenant, UUID %s, LatestVersion %s, Dynakube %s, err: %s",
			tenant.UUID, tenant.LatestVersion, tenant.Dynakube, err)
		log.Info(err.Error())
		return err
	}
	return nil
}

func (a *Access) UpdateTenant(tenant *Tenant) error {
	_, err := a.conn.Exec(updateTenantStatement, tenant.LatestVersion, tenant.Dynakube, tenant.UUID)
	if err != nil {
		err = fmt.Errorf("couldn't insert tenant, UUID %s, LatestVersion %s, Dynakube %s, err: %s",
			tenant.UUID, tenant.LatestVersion, tenant.Dynakube, err)
		log.Info(err.Error())
		return err
	}
	return nil
}

// Returns (nil,nil) if not there
func (a *Access) GetTenant(uuid string) (*Tenant, error) {
	var latestVersion string
	var dynakube string
	row := a.conn.QueryRow(getTenantStatement, uuid)
	err := row.Scan(&latestVersion, &dynakube)
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("couldn't get tenant UUID field for Dynakube %s, err: %s", uuid, err)
		log.Info(err.Error())
		return nil, err
	}
	if err == sql.ErrNoRows {
		return &Tenant{UUID: uuid}, nil
	}
	return &Tenant{uuid, latestVersion, dynakube}, nil
}

// Returns ("",nil) if not there
func (a *Access) GetTenantUUID(dynakube string) (string, error) {
	var tenantUUID string
	row := a.conn.QueryRow(getTenantUUIDStatement, dynakube)
	err := row.Scan(&tenantUUID)
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("couldn't get tenant UUID field for Dynakube %s, err: %s", dynakube, err)
		log.Info(err.Error())
		return "", err
	}
	return tenantUUID, nil
}
