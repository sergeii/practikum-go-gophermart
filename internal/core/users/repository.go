package users

import (
	"context"
	"errors"

	"github.com/sergeii/practikum-go-gophermart/internal/models"
)

var ErrUserLoginIsOccupied = errors.New("login is occupied by another user")
var ErrUserNotFoundInRepo = errors.New("user not found")

type Repository interface {
	Create(context.Context, models.User) (models.User, error)
	GetByID(context.Context, int) (models.User, error)
	GetByLogin(context.Context, string) (models.User, error)
}
