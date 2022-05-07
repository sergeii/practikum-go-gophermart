package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)

type Order struct {
	ID         int
	User       User
	Number     string
	Status     OrderStatus
	Accrual    decimal.Decimal
	UploadedAt time.Time
}

func NewCandidateOrder(number string, userID int) Order {
	return Order{
		User:       User{ID: userID},
		Number:     number,
		Status:     OrderStatusNew,
		UploadedAt: time.Now(),
	}
}

func NewAcceptedOrder(
	id int, number string, userID int, status OrderStatus, accrual decimal.Decimal, uploadedAt time.Time,
) Order {
	return Order{
		ID:         id,
		User:       User{ID: userID},
		Number:     number,
		Status:     status,
		Accrual:    accrual,
		UploadedAt: uploadedAt,
	}
}
