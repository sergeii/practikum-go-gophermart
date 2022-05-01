package application

import (
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/domain/user/service"
)

type App struct {
	UserService *service.Service
	Cfg         config.Config
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

func WithUserService(s *service.Service) Option {
	return func(a *App) {
		a.UserService = s
	}
}
