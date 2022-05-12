package orders

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
)

type OrderStatus string

const (
	OrderStatusNew       OrderStatus = "NEW"
	OrderStatusInvalid   OrderStatus = "INVALID"
	OrderStatusProcessed OrderStatus = "PROCESSED"
)

type Order struct {
	ID         int
	User       users.User
	Number     string
	Status     OrderStatus
	Accrual    decimal.Decimal
	UploadedAt time.Time
}

var Blank Order // nolint: gochecknoglobals

func New(number string, userID int) Order {
	return Order{
		User:       users.NewFromID(userID),
		Number:     number,
		Status:     OrderStatusNew,
		UploadedAt: time.Now(),
	}
}

func NewFromRepo(
	id int, number string, userID int, status OrderStatus, accrual decimal.Decimal, uploadedAt time.Time,
) Order {
	return Order{
		ID:         id,
		User:       users.NewFromID(userID),
		Number:     number,
		Status:     status,
		Accrual:    accrual,
		UploadedAt: uploadedAt,
	}
}
