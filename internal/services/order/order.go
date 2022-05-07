package order

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"

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

func (s Service) UploadOrder(
	ctx context.Context,
	number string,
	userID int,
	extraAction func(models.Order) error,
) (models.Order, error) {
	newOrder := models.NewOrder(number, userID)
	addedOrder, err := s.orders.Add(ctx, newOrder, extraAction)
	// check whether the order has been uploaded by the same user or not
	if err != nil && errors.Is(err, orders.ErrOrderAlreadyExists) {
		dupOrder, getErr := s.orders.GetByNumber(ctx, number)
		if getErr != nil {
			return models.Order{}, getErr
		}
		if dupOrder.User.ID == userID {
			return models.Order{}, ErrOrderAlreadyUploaded
		}
		return models.Order{}, ErrOrderUploadedByAnotherUser
	} else if err != nil {
		return models.Order{}, err
	}
	return addedOrder, nil
}

func (s Service) UpdateOrder(
	ctx context.Context, orderNumber string,
	newStatus models.OrderStatus, accrual float64,
) error {
	o, err := s.orders.GetByNumber(ctx, orderNumber)
	if err != nil {
		if errors.Is(err, orders.ErrOrderNotFound) {
			log.Error().Str("order", orderNumber).Msg("Unable to update non-existent order")
			return err
		}
		log.Error().Err(err).Str("order", orderNumber).Msg("Failed to obtain order")
		return err
	}

	if err = s.orders.UpdateStatus(ctx, o.ID, newStatus, accrual); err != nil {
		log.Error().
			Err(err).
			Int("ID", o.ID).
			Str("number", orderNumber).
			Str("status", string(newStatus)).
			Float64("accrual", accrual).
			Msg("Failed to update order status")
		return err
	}
	return nil
}

func (s Service) GetUserOrders(ctx context.Context, userID int) ([]models.Order, error) {
	return s.orders.GetListForUser(ctx, userID)
}
