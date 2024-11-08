package models

import (
	"context"
	"database/sql"

	"github.com/canonical/go-dqlite/v3/app"
)

var (
	DB *sql.DB
)

func NewDB(dir, address string, peers []string) error {
	dqliteApp, err := app.New(dir, app.WithAddress(address), app.WithCluster(peers))
	if err != nil {
		return err
	}

	if err := dqliteApp.Ready(context.Background()); err != nil {
		return err
	}

	DB, err = dqliteApp.Open(context.Background(), "file:beacon.db?_foreign_keys=on")
	if err != nil {
		return err
	}

	return nil
}

func MigrateDB() error {
	_, err := DB.Exec(schema)
	return err
}
