package application

import (
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/services/account"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
	"github.com/sergeii/practikum-go-gophermart/internal/services/withdrawal"
)

type App struct {
	UserService       account.Service
	OrderService      order.Service
	WithdrawalService withdrawal.Service
	Cfg               config.Config
}

func NewApp(
	cfg config.Config,
	userService account.Service,
	orderService order.Service,
	withdrawalService withdrawal.Service,
) *App {
	return &App{
		Cfg:               cfg,
		UserService:       userService,
		OrderService:      orderService,
		WithdrawalService: withdrawalService,
	}
}
