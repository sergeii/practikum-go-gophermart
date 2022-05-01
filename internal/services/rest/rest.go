package rest

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/internal/application"
	"github.com/sergeii/practikum-go-gophermart/internal/services/rest/handlers"
)

func New(app *application.App) (*gin.Engine, error) {
	router := newRouter(app)
	if err := registerMiddlewares(router); err != nil {
		return nil, err
	}
	if err := registerValidators(); err != nil {
		return nil, err
	}
	return router, nil
}

func newRouter(app *application.App) *gin.Engine {
	handler := handlers.New(app)
	router := gin.Default()
	router.POST("/api/user/register", handler.RegisterUser)
	router.POST("/api/user/login", handler.LoginUser)
	return router
}

func registerMiddlewares(router *gin.Engine) error {
	router.Use(gin.LoggerWithWriter(log.Logger))
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
