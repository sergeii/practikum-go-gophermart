package database

import (
	"context"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // registers postgres support
	_ "github.com/golang-migrate/migrate/v4/source/file"       // registers file-based migrations support
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/db/migrations"
)

func Configure(cfg config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DatabaseConnectTimeout)
	defer cancel()

	dsn, err := url.Parse(cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}
	// run initial migrations when running tests on github
	if dsn.Path == "/praktikum" {
		if err = autoMigrate(cfg.DatabaseDSN); err != nil {
			panic(err)
		}
	}
	db, err := pgxpool.Connect(ctx, cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func autoMigrate(dsn string) error {
	src, err := iofs.New(migrations.Embed, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return err
	}
	if err = m.Up(); err != nil {
		return err
	}
	return nil
}
