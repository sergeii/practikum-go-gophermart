package withdrawals

import (
	"context"
	"errors"
)

var ErrWithdrawalNotFound = errors.New("withdrawal not found")

type Repository interface {
	Add(context.Context, Withdrawal) (Withdrawal, error)
	GetByNumber(context.Context, string) (Withdrawal, error)
	GetListForUser(context.Context, int) ([]Withdrawal, error)
}
