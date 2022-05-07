package application

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/application"
	orders "github.com/sergeii/practikum-go-gophermart/internal/core/orders/db"
	users "github.com/sergeii/practikum-go-gophermart/internal/core/users/db"
	"github.com/sergeii/practikum-go-gophermart/internal/services/account"
	"github.com/sergeii/practikum-go-gophermart/internal/services/accrual"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
	"github.com/sergeii/practikum-go-gophermart/internal/services/processing"
)

func Configure(cfg config.Config, pgpool *pgxpool.Pool) (*application.App, error) {
	orderService := order.New(orders.New(pgpool))
	accrualService, err := accrual.New(cfg.AccrualSystemURL)
	if err != nil {
		log.Error().Err(err).Msg("unable to configure accrual service")
		return nil, err
	}
	app := application.NewApp(
		cfg,
		application.WithUserService(account.New(users.New(pgpool), account.WithBcryptPasswordHasher())),
		application.WithOrderService(orderService),
		application.WithProcessingService(processing.New(
			processing.WithOrderService(orderService),
			processing.WithAccrualService(accrualService),
			processing.WithInMemoryQueue(cfg.AccrualQueueSize),
		)),
	)
	return app, nil
}
