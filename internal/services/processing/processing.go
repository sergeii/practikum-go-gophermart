package processing

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/queue"
	"github.com/sergeii/practikum-go-gophermart/internal/core/queue/memory"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/services/accrual"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
)

var ErrOrderIsNotProcessedYet = errors.New("order is not processed yet")
var ErrFailedOrderIsHandled = errors.New("failed order is handled successfully")

const (
	SleepOnFinishedRun = time.Millisecond * 50
	SleepOnError       = time.Millisecond * 100
	SleepOnEmptyQueue  = time.Second
)

type Service struct {
	queue          queue.Repository
	OrderService   order.Service
	AccrualService accrual.Service
}

type Option func(s *Service)

func WithOrderService(os order.Service) Option {
	return func(s *Service) {
		s.OrderService = os
	}
}

func WithAccrualService(as accrual.Service) Option {
	return func(s *Service) {
		s.AccrualService = as
	}
}

func WithQueueRepository(r queue.Repository) Option {
	return func(s *Service) {
		s.queue = r
	}
}

func WithInMemoryQueue(size int) Option {
	repo, err := memory.New(size)
	if err != nil {
		panic(err)
	}
	return WithQueueRepository(repo)
}

func New(opts ...Option) Service {
	s := Service{}
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

func (s *Service) QueueLength(ctx context.Context) (int, error) {
	return s.queue.Len(ctx)
}

// SubmitNewOrder creates a new order and attempts to add the new order to the processing queue.
// The operation is atomic: if either of the two operations fail,
// the order is not added neither to the queue nor the repository
func (s *Service) SubmitNewOrder(ctx context.Context, orderNumber string, userID int) (models.Order, error) {
	return s.OrderService.UploadOrder(
		ctx, orderNumber, userID,
		func(o models.Order, _ pgx.Tx) error {
			return s.queue.Push(ctx, o.Number)
		},
	)
}

func (s *Service) ProcessNextOrder(ctx context.Context) <-chan struct{} {
	orderNumber, err := s.queue.Pop(ctx)
	if err != nil {
		// queue is currently empty, wait a bit
		if errors.Is(err, queue.ErrQueueIsEmpty) {
			log.Debug().Msg("Accrual order queue is empty")
			return wait(ctx, SleepOnEmptyQueue)
		}
		log.Error().Err(err).Str("order", orderNumber).Msg("Unable to retrieve order from queue")
		return wait(ctx, SleepOnError)
	}

	log.Info().Str("order", orderNumber).Msg("Checking order in accrual system")
	orderStatus, err := s.AccrualService.CheckOrder(orderNumber)

	if err != nil {
		// try to put back order to the queue, unless the error was successfully handled
		customWait, handleErr := s.handleProcessingError(ctx, err, orderNumber)
		if handleErr != nil && !errors.Is(handleErr, ErrFailedOrderIsHandled) {
			s.maybeRequeueOrder(ctx, orderNumber)
		}
		// unless the accrual system wants us to wait for specific duration,
		// use the standard timer
		if customWait != nil {
			return customWait
		}
		return wait(ctx, SleepOnError)
	}

	if handleErr := s.handleProcessingResult(ctx, orderNumber, orderStatus); handleErr != nil {
		log.Warn().
			Err(handleErr).Str("order", orderNumber).Str("status", orderStatus.Status).
			Msg("Failed to handle checked order")
		// return the order to the queue, so it will be checked later
		// better luck next time
		s.maybeRequeueOrder(ctx, orderNumber)
	}

	return wait(ctx, SleepOnFinishedRun)
}

func (s *Service) handleProcessingError(ctx context.Context, err error, orderNumber string) (<-chan struct{}, error) {
	var tooManyReqs *accrual.TooManyRequestError
	// for some reason, accrual system does not know anything about this order
	if errors.Is(err, accrual.ErrOrderNotFound) {
		log.Warn().Str("order", orderNumber).Msg("Order could not be found in accrual system")
		// We mark it invalid and never return to this order again, unless there is a problem saving the status
		updErr := s.OrderService.UpdateOrder(ctx, orderNumber, models.OrderStatusInvalid, decimal.NewFromInt(0))
		if updErr != nil {
			log.Error().
				Err(updErr).Str("order", orderNumber).
				Msgf("Failed to mark unknown order invalid")
			return nil, updErr
		}
		return nil, ErrFailedOrderIsHandled
	}
	// accrual system is busy, gotta wait some time as reported with the Retry-After header value
	if errors.As(err, &tooManyReqs) {
		log.Info().
			Err(err).Str("order", orderNumber).Uint("wait", tooManyReqs.RetryAfter).
			Msg("accrual system is busy")
		return wait(ctx, time.Second*time.Duration(tooManyReqs.RetryAfter)), tooManyReqs
	}
	log.Error().Err(err).Str("order", orderNumber).Msg("Failed to check order status at accrual system")
	return nil, err
}

func (s *Service) handleProcessingResult(ctx context.Context, orderNumber string, os accrual.OrderStatus) error {
	logOrderStatus := log.Info().Str("order", orderNumber).Str("status", os.Status)
	switch os.Status {
	case "INVALID":
		logOrderStatus.Msg("Order is not eligible for accrual")
		err := s.OrderService.UpdateOrder(ctx, orderNumber, models.OrderStatusInvalid, decimal.NewFromInt(0))
		if err != nil {
			return err
		}
	case "PROCESSED":
		logOrderStatus.Stringer("points", os.Accrual).Msg("Points accrued for order")
		if err := s.OrderService.UpdateOrder(ctx, orderNumber, models.OrderStatusProcessed, os.Accrual); err != nil {
			return err
		}
	default:
		// other statuses are not finial, so we put back the order into the queue
		logOrderStatus.Msg("Order is not processed yet")
		return ErrOrderIsNotProcessedYet
	}
	return nil
}

func (s *Service) maybeRequeueOrder(ctx context.Context, orderNumber string) {
	log.Info().Str("order", orderNumber).Msg("Returning order to queue")
	if err := s.queue.Push(ctx, orderNumber); err != nil {
		log.Error().
			Err(err).
			Str("order", orderNumber).
			Msgf("Unable to return order to queue")
	}
}

// wait returns a context-interruptable timer in the form of a receive-only channel
func wait(ctx context.Context, dur time.Duration) <-chan struct{} {
	waitCh := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			close(waitCh)
		case <-time.After(dur):
			waitCh <- struct{}{}
		}
	}()
	return waitCh
}
