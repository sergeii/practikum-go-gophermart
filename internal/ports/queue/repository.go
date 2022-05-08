package queue

import (
	"context"
	"errors"
)

var ErrQueueIsFull = errors.New("accrual queue is full")
var ErrQueueIsEmpty = errors.New("accrual queue is empty")

type Repository interface {
	Push(context.Context, string) error
	Pop(context.Context) (string, error)
	Len(ctx context.Context) (int, error)
}
