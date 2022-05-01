package testutils

import (
	crand "crypto/rand"
	"net/http/httptest"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/application"
	urepo "github.com/sergeii/practikum-go-gophermart/internal/domain/user/repository/db"
	uservice "github.com/sergeii/practikum-go-gophermart/internal/domain/user/service"
	"github.com/sergeii/practikum-go-gophermart/internal/services/rest"
)

func PrepareTestServer() (*httptest.Server, *application.App, func()) {
	cfg := config.Config{}
	crand.Read(cfg.SecretKey) // nolint: errcheck
	db, cancelDatabase := PrepareTestDatabase()
	app := application.NewApp(
		cfg,
		application.WithUserService(uservice.New(
			urepo.New(db),
			uservice.WithBcryptPasswordHasher(),
		)),
	)
	router, _ := rest.New(app)
	ts := httptest.NewServer(router)
	return ts, app, func() {
		ts.Close()
		cancelDatabase()
	}
}
