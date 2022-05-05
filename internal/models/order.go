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
