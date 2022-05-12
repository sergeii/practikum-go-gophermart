package withdrawal

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
	"github.com/sergeii/practikum-go-gophermart/internal/core/withdrawals"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/transactor"
)

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
) (models.Withdrawal, error) {
	var withdrawal models.Withdrawal

	err := s.transactor.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.users.WithdrawPoints(txCtx, userID, sum); err != nil {
			log.Warn().
				Err(err).Str("order", number).Int("userID", userID).Stringer("sum", sum).
				Msg("Unable to withdraw requested sum from user balance")
			return err
		}
		w, err := s.withdrawals.Add(txCtx, models.NewCandidateWithdrawal(number, userID, sum))
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
func (s Service) GetUserWithdrawals(ctx context.Context, userID int) ([]models.Withdrawal, error) {
	return s.withdrawals.GetListForUser(ctx, userID)
}
