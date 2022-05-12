package users

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

var Blank User // nolint: gochecknoglobals

func New(login, password string) User {
	return User{
		Login:    login,
		Password: password,
	}
}

func NewFromRepo(id int, login, password string, accrued, withdrawn decimal.Decimal) User {
	return User{
		ID:       id,
		Login:    login,
		Password: password,
		Balance: UserBalance{
			Current:   accrued,
			Withdrawn: withdrawn,
		},
	}
}

func NewFromID(id int) User {
	return User{ID: id}
}
