package database

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
)

func Configure(cfg config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DatabaseConnectTimeout)
	defer cancel()
	db, err := pgxpool.Connect(ctx, cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}
	return db, nil
}
