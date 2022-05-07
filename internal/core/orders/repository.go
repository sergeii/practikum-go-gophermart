package orders

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v4"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/models"
)

var ErrOrderAlreadyExists = errors.New("order with this number has already been uploaded")
var ErrOrderNotFound = errors.New("order not found")

func AddNoop(models.Order, pgx.Tx) error {
	return nil
}

type Repository interface {
	Add(context.Context, models.Order, func(models.Order, pgx.Tx) error) (models.Order, error)
	UpdateStatus(context.Context, int, models.OrderStatus, decimal.Decimal) error
	GetByNumber(context.Context, string) (models.Order, error)
	GetListForUser(context.Context, int) ([]models.Order, error)
}
