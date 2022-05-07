package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/application"
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/database"
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/logging"
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/server/processing"
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/server/restapi"
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

	wg := &sync.WaitGroup{}
	failure := make(chan error, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case err := <-failure:
			log.Warn().Err(err).Msg("Exiting due to failure")
			cancel()
		case <-shutdown:
			log.Info().Msg("Exiting due to shutdown signal")
			cancel()
		}
	}()

	wg.Add(1)
	go restapi.Run(ctx, app, wg, failure)

	wg.Add(1)
	go processing.Run(ctx, app, wg, failure)

	wg.Wait()
}
