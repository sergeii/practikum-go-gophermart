package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/application"
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/database"
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/logging"
	"github.com/sergeii/practikum-go-gophermart/internal/services/rest"
	httpserver "github.com/sergeii/practikum-go-gophermart/pkg/http/server"
	"github.com/sergeii/practikum-go-gophermart/pkg/random"
)

func main() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	cfg, err := config.Init()
	if err != nil {
		panic(err)
	}

	logger, err := logging.Configure(cfg)
	if err != nil {
		panic(err)
	}
	log.Logger = logger

	// must have properly seeded rng
	if err := random.Seed(); err != nil {
		log.Panic().Err(err).Msg("Unable to start without rand")
	}

	// must have working database connection
	pgpool, err := database.Configure(cfg)
	if err != nil {
		log.Panic().Err(err).Msg("Unable to start without database")
	}
	defer pgpool.Close()

	// must have fully configured app
	app, err := application.Configure(cfg, pgpool)
	if err != nil {
		log.Panic().Err(err).Msg("Unable to configure application")
	}

	router, err := rest.New(app)
	if err != nil {
		log.Panic().Err(err).Msg("Unable to configure rest router")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-shutdown
		log.Info().Msg("Exiting due to shutdown signal")
		cancel()
	}()

	svr, err := httpserver.New(
		cfg.ServerListenAddr,
		httpserver.WithShutdownTimeout(cfg.ServerShutdownTimeout),
		httpserver.WithReadTimeout(cfg.ServerReadTimeout),
		httpserver.WithWriteTimeout(cfg.ServerWriteTimeout),
		httpserver.WithHandler(router),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup HTTP server")
		return
	}
	if err = svr.ListenAndServe(ctx); err != nil {
		log.Error().Err(err).Msg("HTTP server exited prematurely")
		return
	}
}
