package application

import (
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/application"
	orders "github.com/sergeii/practikum-go-gophermart/internal/core/orders/db"
	users "github.com/sergeii/practikum-go-gophermart/internal/core/users/db"
	"github.com/sergeii/practikum-go-gophermart/internal/services/account"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
)

func Configure(cfg config.Config, pgpool *pgxpool.Pool) (*application.App, error) {
	app := application.NewApp(
		cfg,
		application.WithUserService(account.New(users.New(pgpool), account.WithBcryptPasswordHasher())),
		application.WithOrderService(order.New(orders.New(pgpool))),
	)
	return app, nil
}
