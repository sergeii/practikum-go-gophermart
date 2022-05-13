package bootstrap

import (
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/application"
	ordersPG "github.com/sergeii/practikum-go-gophermart/internal/core/orders/postgres"
	usersPG "github.com/sergeii/practikum-go-gophermart/internal/core/users/postgres"
	withdrawalsPG "github.com/sergeii/practikum-go-gophermart/internal/core/withdrawals/postgres"
	"github.com/sergeii/practikum-go-gophermart/internal/persistence/postgres"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/accrual"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/queue/memory"
	"github.com/sergeii/practikum-go-gophermart/internal/services/account"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
	"github.com/sergeii/practikum-go-gophermart/internal/services/withdrawal"
	"github.com/sergeii/practikum-go-gophermart/pkg/security/hasher/bcrypt"
)

func App(cfg config.Config, pg *postgres.Database) (*application.App, error) {
	accrualService, err := accrual.New(cfg.AccrualSystemURL)
	if err != nil {
		log.Error().Err(err).Msg("Unable to configure accrual service")
		return nil, err
	}

	accrualQueue, err := memory.New(cfg.AccrualQueueSize)
	if err != nil {
		return nil, err
	}

	// repos
	users := usersPG.New(pg)
	orders := ordersPG.New(pg)
	withdrawals := withdrawalsPG.New(pg)

	app := application.NewApp(
		cfg,
		account.New(users, bcrypt.New()),
		order.New(
			orders, users, pg,
			accrualQueue, accrualService,
		),
		withdrawal.New(withdrawals, users, pg),
	)
	return app, nil
}
