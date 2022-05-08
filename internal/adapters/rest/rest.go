package rest

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/internal/adapters/rest/handlers"
	"github.com/sergeii/practikum-go-gophermart/internal/adapters/rest/middleware/auth"
	"github.com/sergeii/practikum-go-gophermart/internal/adapters/rest/validate"
	"github.com/sergeii/practikum-go-gophermart/internal/application"
)

func New(app *application.App) (*gin.Engine, error) {
	router := gin.New()
	if err := registerMiddlewares(router, app); err != nil {
		return nil, err
	}
	if err := registerValidators(); err != nil {
		return nil, err
	}
	if err := registerRoutes(router, app); err != nil {
		return nil, err
	}
	return router, nil
}

func registerRoutes(r *gin.Engine, app *application.App) error { // nolint: unparam
	handler := handlers.New(app)
	privateRoutes := r.Group("/", auth.RequireAuthentication)
	registerPublicRoutes(r, handler)
	registerPrivateRoutes(privateRoutes, handler)
	return nil
}

func registerPublicRoutes(r *gin.Engine, h *handlers.Handler) {
	r.POST("/api/user/register", h.RegisterUser)
	r.POST("/api/user/login", h.LoginUser)
}

func registerPrivateRoutes(r *gin.RouterGroup, h *handlers.Handler) {
	r.POST("/api/user/orders", h.UploadOrder)
	r.GET("/api/user/orders", h.ListUserOrders)
	r.GET("/api/user/balance", h.ShowUserBalance)
	r.POST("/api/user/balance/withdraw", h.RequestWithdrawal)
	r.GET("/api/user/balance/withdrawals", h.ListUserWithdrawals)
}

func registerMiddlewares(router *gin.Engine, app *application.App) error { // nolint: unparam
	router.Use(gin.LoggerWithWriter(log.Logger))
	router.Use(gin.Recovery())
	router.Use(auth.Authentication(app.Cfg))
	return nil
}

func registerValidators() error {
	var customValidators = [...]struct {
		name      string
		validator validator.Func
	}{
		{
			"notblank",
			validators.NotBlank,
		},
		{
			"luhn",
			validate.LuhnNumber,
		},
	}
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		for _, val := range customValidators {
			if err := v.RegisterValidation(val.name, val.validator); err != nil {
				return err
			}
		}
	}
	return nil
}
