package orders

import (
	"context"
	"errors"

	"github.com/sergeii/practikum-go-gophermart/internal/models"
)

var ErrOrderAlreadyExists = errors.New("order with this number has already been uploaded")
var ErrOrderNotFound = errors.New("order not found")

func AddNoop(models.Order) error {
	return nil
}

type Repository interface {
	Add(context.Context, models.Order, func(models.Order) error) (models.Order, error)
	UpdateStatus(context.Context, int, models.OrderStatus, float64) error
	GetByNumber(context.Context, string) (models.Order, error)
	GetListForUser(context.Context, int) ([]models.Order, error)
}
