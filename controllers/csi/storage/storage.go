package storage

import (
	"database/sql"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/logger"
	_ "modernc.org/sqlite"
)

const (
	driverName  = "sqlite"
	dbPath      = "./csi.db"
	dbTableName = "csi"
)

var (
	log = logger.NewDTLogger().WithName("provisioner")
)

type Access struct {
	conn *sql.DB
}

func NewAccess() Access {
	a := Access{}
	a.Connect(driverName, dbPath)
	a.init()
	return a
}

func (a *Access) Connect(driver, path string) {
	db, err := sql.Open(driver, path)
	if err != nil {
		log.Error(err, "Couldn't connect to db", "dataBaseName", dbPath)
	}
	a.conn = db
}

func (a *Access) init() error {
	if err := a.createTable(); err != nil {
		return err
	}
	return nil
}

func (a *Access) createTable() error {
	_, err := a.conn.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (tenantId INTEGER PRIMARY KEY);", dbTableName))
	if err != nil {
		err = fmt.Errorf("couldn't create the table %s, err: %s", dbTableName, err)
		log.Info(err.Error())
		return err
	}
	return nil
}
