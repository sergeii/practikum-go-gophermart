package order

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/orders"
	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/accrual"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/queue"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/transactor"
)

var ErrOrderAlreadyUploaded = errors.New("order has already been uploaded by the same user")
var ErrOrderUploadedByAnotherUser = errors.New("order has already been uploaded by another user")
var ErrOrderIsNotProcessedYet = errors.New("order is not processed yet")
var ErrOrderProcessingErrorIsHandled = errors.New("failed order is handled successfully")

const (
	PostProcessWaitOnFinishedRun = time.Millisecond * 50
	PostProcessWaitOnError       = time.Millisecond * 100
	PostProcessWaitOnEmptyQueue  = time.Second
)

type Service struct {
	orders         orders.Repository
	users          users.Repository
	processing     queue.Repository
	transactor     transactor.Transactor
	AccrualService accrual.Service
}

func New(
	orders orders.Repository,
	users users.Repository,
	transactor transactor.Transactor,
	processing queue.Repository,
	accrual accrual.Service,
) Service {
	return Service{
		orders:         orders,
		users:          users,
		transactor:     transactor,
		processing:     processing,
		AccrualService: accrual,
	}
}

// SubmitNewOrder creates a new order and attempts to add the new order to the processing queue.
// The operation is atomic: if either of the two operations fail,
// the order is not added neither to the queue nor into the repository
func (s Service) SubmitNewOrder(ctx context.Context, number string, userID int) (orders.Order, error) {
	// check whether an order with the same number has already been uploaded
	if conflict, err := s.orders.GetByNumber(ctx, number); !errors.Is(err, orders.ErrOrderNotFound) {
		if conflict.User.ID == userID {
			return orders.Blank, ErrOrderAlreadyUploaded
		}
		return orders.Blank, ErrOrderUploadedByAnotherUser
	}
	var order orders.Order
	err := s.transactor.WithTransaction(ctx, func(txCtx context.Context) error {
		o, err := s.orders.Add(txCtx, orders.New(number, userID))
		if err != nil {
			log.Error().
				Err(err).Str("order", o.Number).Int("userID", userID).
				Msg("Failed to add new order")
			return err
		}
		if err = s.processing.Push(txCtx, o.Number); err != nil {
			log.Error().
				Err(err).Str("order", o.Number).Int("userID", userID).
				Msg("Failed to submit new order to queue")
			return err
		}
		order = o
		return nil
	})
	if err != nil {
		return orders.Blank, err
	}
	return order, nil
}

func (s Service) UpdateOrderStatus(
	ctx context.Context, orderNumber string, newStatus orders.OrderStatus, accrual decimal.Decimal,
) error {
	order, err := s.orders.GetByNumber(ctx, orderNumber)
	if err != nil {
		if errors.Is(err, orders.ErrOrderNotFound) {
			log.Error().Str("order", orderNumber).Msg("Unable to update non-existent order")
			return err
		}
		log.Error().Err(err).Str("order", orderNumber).Msg("Failed to obtain order")
		return err
	}
	order.Status = newStatus
	order.Accrual = accrual
	if err = s.orders.Update(ctx, order.ID, order); err != nil {
		log.Error().
			Err(err).
			Int("ID", order.ID).
			Str("number", orderNumber).
			Str("status", string(newStatus)).
			Stringer("accrual", accrual).
			Msg("Failed to update order status")
		return err
	}
	return nil
}

// GetUserOrders returns all orders submitted by the specified user
func (s Service) GetUserOrders(ctx context.Context, userID int) ([]orders.Order, error) {
	return s.orders.GetListForUser(ctx, userID)
}

// ProcessingLength returns the current length of the processing queue,
// i.e. the number of orders currently waiting to be processed with the accrual system
func (s *Service) ProcessingLength(ctx context.Context) (int, error) {
	return s.processing.Len(ctx)
}

// ProcessNextOrder picks an order from the processing queue
// and then checks the order's status in the accrual system.
// Depending on the result of the check, the order may be marked as processed or invalid,
// or put back into the queue for further processing, until the order's status is finalized.
// The method returns a channel that the caller is recommended to wait on
// before starting to process the next order.
// The returned channel contains a timer with varying duration.
// In its turn, the varying duration depends on the busyness of the accrual system
func (s *Service) ProcessNextOrder(ctx context.Context) <-chan time.Time {
	orderNumber, err := s.processing.Pop(ctx)
	if err != nil {
		// queue is currently empty, wait a bit
		if errors.Is(err, queue.ErrQueueIsEmpty) {
			log.Debug().Msg("Accrual order queue is empty")
			return time.After(PostProcessWaitOnEmptyQueue)
		}
		log.Error().Err(err).Str("order", orderNumber).Msg("Unable to retrieve order from queue")
		return time.After(PostProcessWaitOnError)
	}

	log.Info().Str("order", orderNumber).Msg("Checking order in accrual system")
	orderStatus, err := s.AccrualService.CheckOrder(orderNumber)

	if err != nil {
		// try to put back order to the queue, unless the error was successfully handled
		customWait, handleErr := s.handleProcessingError(ctx, err, orderNumber)
		if handleErr != nil && !errors.Is(handleErr, ErrOrderProcessingErrorIsHandled) {
			s.maybeResubmitOrder(ctx, orderNumber)
		}
		// unless the accrual system wants us to wait for specific duration,
		// use the standard timer
		if customWait != nil {
			return customWait
		}
		return time.After(PostProcessWaitOnError)
	}

	if handleErr := s.handleProcessingResult(ctx, orderNumber, orderStatus); handleErr != nil {
		log.Warn().
			Err(handleErr).Str("order", orderNumber).Str("status", orderStatus.Status).
			Msg("Failed to handle checked order")
		// return the order to the queue, so it will be checked later
		// better luck next time
		s.maybeResubmitOrder(ctx, orderNumber)
	}

	return time.After(PostProcessWaitOnFinishedRun)
}

func (s *Service) handleProcessingError(ctx context.Context, err error, orderNumber string) (<-chan time.Time, error) {
	var tooManyReqs *accrual.TooManyRequestError
	// for some reason, accrual system does not know anything about this order
	if errors.Is(err, accrual.ErrOrderNotFound) {
		log.Warn().Str("order", orderNumber).Msg("Order could not be found in accrual system")
		// We mark it invalid and never return to this order again, unless there is a problem saving the status
		updErr := s.UpdateOrderStatus(ctx, orderNumber, orders.OrderStatusInvalid, decimal.NewFromInt(0))
		if updErr != nil {
			log.Error().
				Err(updErr).Str("order", orderNumber).
				Msg("Failed to mark unknown order invalid")
			return nil, updErr
		}
		return nil, ErrOrderProcessingErrorIsHandled
	}
	// accrual system is busy, gotta wait some time as reported with the Retry-After header value
	if errors.As(err, &tooManyReqs) {
		log.Info().
			Err(err).Str("order", orderNumber).Uint("wait", tooManyReqs.RetryAfter).
			Msg("Accrual system is busy")
		return time.After(time.Second * time.Duration(tooManyReqs.RetryAfter)), tooManyReqs
	}
	log.Error().Err(err).Str("order", orderNumber).Msg("Failed to check order status at accrual system")
	return nil, err
}

func (s *Service) handleProcessingResult(ctx context.Context, orderNumber string, os accrual.OrderStatus) error {
	logOrderStatus := log.Info().Str("order", orderNumber).Str("status", os.Status)
	switch os.Status {
	case "INVALID":
		logOrderStatus.Msg("Order is not eligible for accrual")
		err := s.UpdateOrderStatus(ctx, orderNumber, orders.OrderStatusInvalid, decimal.NewFromInt(0))
		if err != nil {
			return err
		}
	case "PROCESSED":
		logOrderStatus.Stringer("points", os.Accrual).Msg("Points accrued for order")
		txErr := s.transactor.WithTransaction(ctx, func(txCtx context.Context) error {
			if err := s.UpdateOrderStatus(txCtx, orderNumber, orders.OrderStatusProcessed, os.Accrual); err != nil {
				return err
			}
			o, err := s.orders.GetByNumber(txCtx, orderNumber)
			if err != nil {
				return err
			}
			if err := s.users.AccruePoints(txCtx, o.User.ID, os.Accrual); err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			log.Error().Err(txErr).Str("order", orderNumber).Msg("Failed to accrue points for order")
			return txErr
		}
	default:
		// other statuses are not finial, so we put back the order into the queue
		logOrderStatus.Msg("Order is not processed yet")
		return ErrOrderIsNotProcessedYet
	}
	return nil
}

func (s *Service) maybeResubmitOrder(ctx context.Context, orderNumber string) {
	log.Info().Str("order", orderNumber).Msg("Returning order to queue")
	if err := s.processing.Push(ctx, orderNumber); err != nil {
		log.Error().
			Err(err).
			Str("order", orderNumber).
			Msg("Unable to return order to queue")
	}
}
