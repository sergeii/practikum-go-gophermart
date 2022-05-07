package testutils

import (
	crand "crypto/rand"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	appcfg "github.com/sergeii/practikum-go-gophermart/cmd/gophermart/application"
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/application"
	"github.com/sergeii/practikum-go-gophermart/internal/services/rest"
)

type TestServerOpt func(*config.Config)

func PrepareTestServer(opts ...TestServerOpt) (*httptest.Server, *application.App, func()) {
	cfg := config.Config{
		AccrualSystemURL: "http://localhost:8081",
		AccrualQueueSize: 10,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	crand.Read(cfg.SecretKey) // nolint: errcheck
	db, cancelDatabase := PrepareTestDatabase()
	app, err := appcfg.Configure(cfg, db)
	if err != nil {
		panic(err)
	}
	gin.SetMode(gin.ReleaseMode) // prevent gin from overwriting middlewares
	router, err := rest.New(app)
	if err != nil {
		panic(err)
	}
	ts := httptest.NewServer(router)
	return ts, app, func() {
		ts.Close()
		cancelDatabase()
	}
}
