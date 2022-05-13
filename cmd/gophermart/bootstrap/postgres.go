package bootstrap

import (
	"context"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // registers postgres support
	_ "github.com/golang-migrate/migrate/v4/source/file"       // registers file-based migrations support
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/db/migrations"
	"github.com/sergeii/practikum-go-gophermart/internal/persistence/postgres"
)

func Postgres(cfg config.Config) (*postgres.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DatabaseConnectTimeout)
	defer cancel()
	// apply migrations when in development mode
	if !cfg.Production {
		log.Info().Msg("Applying migrations")
		if err := autoMigrate(cfg.DatabaseDSN); err != nil {
			log.Error().Err(err).Msg("Failed to apply migrations")
			return nil, err
		}
	}
	pgpool, err := pgxpool.Connect(ctx, cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}
	return postgres.New(pgpool), nil
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
