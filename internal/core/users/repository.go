package users

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/models"
)

var ErrUserLoginIsOccupied = errors.New("login is occupied by another user")
var ErrUserNotFoundInRepo = errors.New("user not found")
var ErrUserHasInsufficientAccrual = errors.New("user has no enough points to withdraw")
var ErrUserCantWithdrawNegativeSum = errors.New("user must withdraw positive sum only")

type Repository interface {
	Create(context.Context, models.User) (models.User, error)
	GetByID(context.Context, int) (models.User, error)
	GetByLogin(context.Context, string) (models.User, error)
	AccruePoints(context.Context, int, decimal.Decimal) error
	WithdrawPoints(context.Context, int, decimal.Decimal) error
}
