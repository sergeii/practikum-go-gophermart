package application

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/application"
	orders "github.com/sergeii/practikum-go-gophermart/internal/core/orders/db"
	users "github.com/sergeii/practikum-go-gophermart/internal/core/users/db"
	withdrawals "github.com/sergeii/practikum-go-gophermart/internal/core/withdrawals/db"
	"github.com/sergeii/practikum-go-gophermart/internal/persistence/db"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/accrual"
	"github.com/sergeii/practikum-go-gophermart/internal/services/account"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
	"github.com/sergeii/practikum-go-gophermart/internal/services/withdrawal"
)

func Configure(cfg config.Config, pgpool *pgxpool.Pool) (*application.App, error) {
	accrualService, err := accrual.New(cfg.AccrualSystemURL)
	if err != nil {
		log.Error().Err(err).Msg("unable to configure accrual service")
		return nil, err
	}

	database := db.New(pgpool)
	userRepo := users.New(database)
	orderRepo := orders.New(database)
	withdrawalRepo := withdrawals.New(database)

	app := application.NewApp(
		cfg,
		application.WithUserService(
			account.New(
				userRepo,
				account.WithBcryptPasswordHasher(),
			),
		),
		application.WithOrderService(
			order.New(
				orderRepo, userRepo,
				order.WithTransactor(database),
				order.WithAccrualService(accrualService),
				order.WithInMemoryQueue(cfg.AccrualQueueSize),
			),
		),
		application.WithWithdrawalService(
			withdrawal.New(
				withdrawalRepo, userRepo,
				withdrawal.WithTransactor(database),
			),
		),
	)
	return app, nil
}
