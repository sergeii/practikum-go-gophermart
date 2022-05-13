package withdrawal

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
	"github.com/sergeii/practikum-go-gophermart/internal/core/withdrawals"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/transactor"
)

var ErrWithdrawalAlreadyRegistered = errors.New("withdrawal for this order has already been registered")
var ErrWithdrawalInvalidSumSum = errors.New("can withdraw positive sum only")

type Service struct {
	withdrawals withdrawals.Repository
	users       users.Repository
	transactor  transactor.Transactor
}

func New(withdrawals withdrawals.Repository, users users.Repository, transactor transactor.Transactor) Service {
	return Service{
		withdrawals: withdrawals,
		users:       users,
		transactor:  transactor,
	}
}

// RequestWithdrawal attempts to withdraw specified sum from the selected user's account.
// A successful withdrawal may succeed in the following scenario only:
// * the user has enough current balance to withdraw from;
// * the specified order number has never been used for withdrawal before;
// * the withdrawn sum is positive.
// In other cases an error is returned
func (s Service) RequestWithdrawal(
	ctx context.Context,
	number string,
	userID int,
	sum decimal.Decimal,
) (withdrawals.Withdrawal, error) {
	// check whether a withdrawal with the same number has already been registered
	if _, err := s.withdrawals.GetByNumber(ctx, number); !errors.Is(err, withdrawals.ErrWithdrawalNotFound) {
		return withdrawals.Blank, ErrWithdrawalAlreadyRegistered
	}
	// must withdraw positive sum only
	if sum.LessThanOrEqual(decimal.Zero) {
		return withdrawals.Blank, ErrWithdrawalInvalidSumSum
	}
	var withdrawal withdrawals.Withdrawal
	err := s.transactor.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.users.WithdrawPoints(txCtx, userID, sum); err != nil {
			log.Warn().
				Err(err).Str("order", number).Int("userID", userID).Stringer("sum", sum).
				Msg("Unable to withdraw requested sum from user balance")
			return err
		}
		w, err := s.withdrawals.Add(txCtx, withdrawals.New(number, userID, sum))
		if err != nil {
			log.Error().
				Err(err).Str("order", w.Number).Int("userID", userID).
				Msg("Failed to add new withdrawal")
			return err
		}
		withdrawal = w
		return nil
	})

	if err != nil {
		return withdrawal, err
	}

	return withdrawal, nil
}

// GetUserWithdrawals returns all successful withdrawals requested by the specified user
func (s Service) GetUserWithdrawals(ctx context.Context, userID int) ([]withdrawals.Withdrawal, error) {
	return s.withdrawals.GetListForUser(ctx, userID)
}
