package testutils

import (
	"context"
	"fmt"

	"github.com/caarlos0/env/v6"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // registers postgres support
	_ "github.com/golang-migrate/migrate/v4/source/file"       // registers file-based migrations support
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/sergeii/practikum-go-gophermart/db/migrations"
	"github.com/sergeii/practikum-go-gophermart/pkg/random"
)

func PrepareTestDatabase() (*pgxpool.Pool, func()) {
	type config struct {
		DatabaseDSN string `env:"DATABASE_URI" envDefault:"postgres://gophermart@localhost:5432/gophermart"`
	}
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}

	if err := random.Seed(); err != nil {
		panic(err)
	}
	// create a separate schema with random name, so concurrent tests' databases dont clash with each other
	schema := random.String(5, "abcdefghijklmnopqrstuvwxyz")
	db, err := pgx.Connect(context.TODO(), cfg.DatabaseDSN)
	if err != nil {
		panic(err)
	}
	if _, err := db.Exec(context.TODO(), fmt.Sprintf("CREATE SCHEMA %s", schema)); err != nil {
		panic(err)
	}
	// use the prepared schema
	dsn := fmt.Sprintf("%s?sslmode=disable&search_path=%s", cfg.DatabaseDSN, schema)
	pool, err := pgxpool.Connect(context.TODO(), dsn)
	if err != nil {
		panic(err)
	}

	// run migrations
	src, err := iofs.New(migrations.Embed, ".")
	if err != nil {
		panic(err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		panic(err)
	}
	if err = m.Up(); err != nil {
		panic(err)
	}

	return pool, func() {
		defer pool.Close()
		defer db.Close(context.TODO())
		if _, err := db.Exec(context.TODO(), fmt.Sprintf("DROP SCHEMA %s CASCADE", schema)); err != nil {
			panic(err)
		}
	}
}
