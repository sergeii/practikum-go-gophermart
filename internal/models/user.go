package models

import "github.com/shopspring/decimal"

type UserBalance struct {
	Current   decimal.Decimal
	Withdrawn decimal.Decimal
}

type User struct {
	ID       int
	Login    string
	Password string
	Balance  UserBalance
}
