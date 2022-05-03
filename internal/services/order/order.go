package order

import (
	"context"
	"errors"

	"github.com/sergeii/practikum-go-gophermart/internal/core/orders"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
)

var ErrOrderAlreadyUploaded = errors.New("order has already been uploaded by the same user")
var ErrOrderUploadedByAnotherUser = errors.New("order has already been uploaded by another user")

type Service struct {
	orders orders.Repository
}

func New(orders orders.Repository) Service {
	return Service{orders}
}

func (s Service) UploadOrder(ctx context.Context, user models.User, number string) (models.Order, error) {
	newOrder := models.NewOrder(user, number)
	addedOrder, err := s.orders.Add(ctx, newOrder)
	// check whether the order has been uploaded by the same user or not
	if err != nil && errors.Is(err, orders.ErrOrderAlreadyExists) {
		dupOrder, getErr := s.orders.GetByNumber(ctx, number)
		if getErr != nil {
			return models.Order{}, getErr
		}
		if dupOrder.User.ID == user.ID {
			return models.Order{}, ErrOrderAlreadyUploaded
		}
		return models.Order{}, ErrOrderUploadedByAnotherUser
	} else if err != nil {
		return models.Order{}, err
	}
	return addedOrder, nil
}
