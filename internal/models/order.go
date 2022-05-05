package models

import "time"

type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "new"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusInvalid    OrderStatus = "invalid"
	OrderStatusProcessed  OrderStatus = "processed"
)

type Order struct {
	ID         int
	User       User
	Number     string
	Status     OrderStatus
	UploadedAt time.Time
}

func NewOrder(owner User, number string) Order {
	return Order{
		User:       owner,
		Number:     number,
		Status:     OrderStatusNew,
		UploadedAt: time.Now(),
	}
}
