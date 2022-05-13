package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/bootstrap"
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/run"
)

func main() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	cfg, err := bootstrap.Config()
	if err != nil {
		panic(err)
	}

	logger, err := bootstrap.Logging(cfg)
	if err != nil {
		panic(err)
	}
	log.Logger = logger

	// must have working database connection
	pg, err := bootstrap.Postgres(cfg)
	if err != nil {
		log.Panic().Err(err).Msg("Unable to start without database")
	}
	defer pg.Close()

	// must have fully configured app
	app, err := bootstrap.App(cfg, pg)
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
	go run.RestAPI(ctx, app, wg, failure)

	wg.Add(1)
	go run.Processing(ctx, app, wg, failure)

	wg.Wait()
}
