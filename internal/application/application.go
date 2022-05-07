package application

import (
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/services/account"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
	"github.com/sergeii/practikum-go-gophermart/internal/services/processing"
)

type App struct {
	UserService       account.Service
	OrderService      order.Service
	ProcessingService processing.Service
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

func WithProcessingService(s processing.Service) Option {
	return func(a *App) {
		a.ProcessingService = s
	}
}
