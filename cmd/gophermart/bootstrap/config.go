package bootstrap

import (
	crand "crypto/rand"
	"encoding/hex"
	"flag"
	"time"

	"github.com/caarlos0/env/v6"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
)

const SecretKeyLength = 32

func Config() (config.Config, error) {
	cfg := config.Config{}

	if err := env.Parse(&cfg); err != nil {
		return config.Config{}, err
	}

	flag.StringVar(&cfg.ServerListenAddr, "a", cfg.ServerListenAddr, "Address to listen on")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "Database DSN (only postgresql is accepted)")
	flag.StringVar(&cfg.AccrualSystemURL, "r", cfg.AccrualSystemURL, "Accrual system url")
	flag.DurationVar(
		&cfg.ServerShutdownTimeout, "server.shutdown-timeout", time.Second*10,
		"The maximum duration the server should wait for connections to finish before exiting",
	)
	flag.DurationVar(
		&cfg.ServerReadTimeout, "http.read-timeout", time.Second*5,
		"Limits the time it takes from accepting a new connection till reading of the request body",
	)
	flag.DurationVar(
		&cfg.ServerWriteTimeout, "http.write-timeout", time.Second*5,
		"Limits the time it takes from reading the body of a request till the end of the response",
	)
	flag.DurationVar(
		&cfg.DatabaseConnectTimeout, "database.connect-timeout", time.Second*5,
		"Database connection timeout",
	)
	flag.StringVar(
		&cfg.LogLevel, "log.level", "info",
		"Only log messages with the given severity or above.\n"+
			"For example: debug, info, warn, error and other levels supported by zerolog",
	)
	flag.StringVar(
		&cfg.LogOutput, "log.output", "console",
		"Output format of log messages. Available options: console, stdout, json",
	)
	flag.IntVar(
		&cfg.AccrualQueueSize, "accrual.queue-size", 100,
		"Maximum size of the accrual processing queue",
	)
	flag.BoolVar(
		&cfg.Production, "production", false,
		"Run service in production mode",
	)

	flag.Parse()

	// ensure we have a non-empty secret key configured
	if err := configureSecretKey(&cfg); err != nil {
		return config.Config{}, err
	}

	return cfg, nil
}

func configureSecretKey(cfg *config.Config) error {
	if cfg.SecretKeyEncoded != "" {
		confKey, err := hex.DecodeString(cfg.SecretKeyEncoded)
		if err != nil {
			return err
		}
		cfg.SecretKey = confKey
		return nil
	}
	randKey := make([]byte, SecretKeyLength)
	if _, err := crand.Read(randKey); err != nil {
		return err
	}
	cfg.SecretKey = randKey
	return nil
}
