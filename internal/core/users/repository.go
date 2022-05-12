package users

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserHasInsufficientBalance = errors.New("user has no enough points to withdraw")

type Repository interface {
	Create(context.Context, User) (User, error)
	GetByID(context.Context, int) (User, error)
	GetByLogin(context.Context, string) (User, error)
	AccruePoints(context.Context, int, decimal.Decimal) error
	WithdrawPoints(context.Context, int, decimal.Decimal) error
}
