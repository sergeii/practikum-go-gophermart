package orders

import (
	"context"
	"errors"
)

var ErrOrderNotFound = errors.New("order not found")

type Repository interface {
	Add(context.Context, Order) (Order, error)
	Update(context.Context, int, Order) error
	GetByNumber(context.Context, string) (Order, error)
	GetListForUser(context.Context, int) ([]Order, error)
}
