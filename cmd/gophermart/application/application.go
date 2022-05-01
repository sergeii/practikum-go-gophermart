package application

import (
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/application"
	userrepo "github.com/sergeii/practikum-go-gophermart/internal/domain/user/repository/db"
	userservice "github.com/sergeii/practikum-go-gophermart/internal/domain/user/service"
)

func Configure(cfg config.Config, pgpool *pgxpool.Pool) (*application.App, error) {
	app := application.NewApp(
		cfg,
		application.WithUserService(userservice.New(
			userrepo.New(pgpool, userrepo.WithQueryTimeout(cfg.DatabaseQueryTimeout)),
			userservice.WithBcryptPasswordHasher(),
		)),
	)
	return app, nil
}
