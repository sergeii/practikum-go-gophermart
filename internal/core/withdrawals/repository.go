package withdrawals

import (
	"context"
	"errors"

	"github.com/sergeii/practikum-go-gophermart/internal/models"
)

var ErrWithdrawalAlreadyRegistered = errors.New("withdrawal for this order has already been registered")
var ErrWithdrawalMustHavePositiveSum = errors.New("negative withdrawal cannot be made")

type Repository interface {
	Add(context.Context, models.Withdrawal) (models.Withdrawal, error)
	GetListForUser(context.Context, int) ([]models.Withdrawal, error)
}
