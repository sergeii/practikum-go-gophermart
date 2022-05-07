package restapi

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/internal/application"
	"github.com/sergeii/practikum-go-gophermart/internal/services/rest"
	httpserver "github.com/sergeii/practikum-go-gophermart/pkg/http/server"
)

func Run(ctx context.Context, app *application.App, wg *sync.WaitGroup, failure chan error) {
	defer wg.Done()

	router, err := rest.New(app)
	if err != nil {
		log.Panic().Err(err).Msg("Unable to configure rest router")
		failure <- err
		return
	}

	svr, err := httpserver.New(
		app.Cfg.ServerListenAddr,
		httpserver.WithShutdownTimeout(app.Cfg.ServerShutdownTimeout),
		httpserver.WithReadTimeout(app.Cfg.ServerReadTimeout),
		httpserver.WithWriteTimeout(app.Cfg.ServerWriteTimeout),
		httpserver.WithHandler(router),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup HTTP server")
		failure <- err
		return
	}
	if err = svr.ListenAndServe(ctx); err != nil {
		log.Error().Err(err).Msg("HTTP server exited prematurely")
		failure <- err
		return
	}
}
