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

type Option func(a *App)

func NewApp(cfg config.Config, opts ...Option) *App {
	app := &App{
		Cfg: cfg,
	}
	for _, opt := range opts {
		opt(app)
	}
	return app
}

func WithUserService(s account.Service) Option {
	return func(a *App) {
		a.UserService = s
	}
}

func WithOrderService(s order.Service) Option {
	return func(a *App) {
		a.OrderService = s
	}
}

func WithWithdrawalService(s withdrawal.Service) Option {
	return func(a *App) {
		a.WithdrawalService = s
	}
}
