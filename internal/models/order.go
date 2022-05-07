package models

import "time"

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
	Accrual    float64
	UploadedAt time.Time
}

func NewOrder(number string, userID int) Order {
	return Order{
		User:       User{ID: userID},
		Number:     number,
		Status:     OrderStatusNew,
		UploadedAt: time.Now(),
	}
}
