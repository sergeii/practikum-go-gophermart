package orders

import (
	"context"
	"errors"

	"github.com/sergeii/practikum-go-gophermart/internal/models"
)

var ErrOrderAlreadyExists = errors.New("order with this number has already been uploaded")
var ErrOrderNotFound = errors.New("order not found")

type Repository interface {
	Add(context.Context, models.Order) (models.Order, error)
	GetByNumber(context.Context, string) (models.Order, error)
}
