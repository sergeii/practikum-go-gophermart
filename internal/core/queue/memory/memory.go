package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/sergeii/practikum-go-gophermart/internal/core/queue"
)

type Queue struct {
	queue   []string
	maxSize int
	size    int
	head    int
	tail    int
	mu      sync.Mutex
}

var ErrSizeIsInvalid = errors.New("queue cannot be of this size")

// New initializes a fixed size queue based on the ring buffer algorithm.
// The queue holds order numbers that await processing in the accrual system
func New(size int) (*Queue, error) {
	if size <= 0 {
		return nil, ErrSizeIsInvalid
	}
	q := Queue{
		queue:   make([]string, size),
		maxSize: size,
		head:    0,
		tail:    0,
		size:    0,
	}
	return &q, nil
}

func (q *Queue) Push(ctx context.Context, orderNumber string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.size >= q.maxSize {
		return queue.ErrQueueIsFull
	}
	q.queue[q.tail] = orderNumber
	q.size++
	q.tail = (q.tail + 1) % q.maxSize
	return nil
}

func (q *Queue) Pop(ctx context.Context) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.size == 0 {
		return "", queue.ErrQueueIsEmpty
	}
	orderNumber := q.queue[q.head]
	q.queue[q.head] = ""
	q.head = (q.head + 1) % q.maxSize
	q.size--
	return orderNumber, nil
}

func (q *Queue) Len(ctx context.Context) (int, error) {
	return q.size, nil
}
