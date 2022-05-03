package testutils

import (
	crand "crypto/rand"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/application"
	orders "github.com/sergeii/practikum-go-gophermart/internal/core/orders/db"
	users "github.com/sergeii/practikum-go-gophermart/internal/core/users/db"
	"github.com/sergeii/practikum-go-gophermart/internal/services/account"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
	"github.com/sergeii/practikum-go-gophermart/internal/services/rest"
)

func PrepareTestServer() (*httptest.Server, *application.App, func()) {
	cfg := config.Config{}
	crand.Read(cfg.SecretKey) // nolint: errcheck
	db, cancelDatabase := PrepareTestDatabase()
	app := application.NewApp(
		cfg,
		application.WithUserService(account.New(users.New(db), account.WithBcryptPasswordHasher())),
		application.WithOrderService(order.New(orders.New(db))),
	)
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
