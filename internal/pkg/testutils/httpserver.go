package testutils

import (
	crand "crypto/rand"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	appcfg "github.com/sergeii/practikum-go-gophermart/cmd/gophermart/application"
	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/adapters/rest"
	"github.com/sergeii/practikum-go-gophermart/internal/application"
)

type TestServerOpt func(*config.Config)

func PrepareTestServer(opts ...TestServerOpt) (*httptest.Server, *application.App, func()) {
	secretKey := make([]byte, 32)
	if _, err := crand.Read(secretKey); err != nil {
		panic(err)
	}
	cfg := config.Config{
		AccrualSystemURL: "http://localhost:8081",
		AccrualQueueSize: 10,
		SecretKey:        secretKey,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	pg, _, cancelDatabase := PrepareTestDatabase()
	app, err := appcfg.Configure(cfg, pg)
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
		defer cancelDatabase()
		defer ts.Close()
	}
}
