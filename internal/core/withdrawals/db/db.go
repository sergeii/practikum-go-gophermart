package db

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/withdrawals"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/persistence/db"
)

type withdrawalRow struct {
	ID          int
	UserID      int
	Number      string
	Sum         decimal.Decimal
	ProcessedAt time.Time
}

type Repository struct {
	db *db.Database
}

func New(db *db.Database) Repository {
	return Repository{db}
}

func (r Repository) Add(ctx context.Context, cw models.Withdrawal) (models.Withdrawal, error) {
	conn := r.db.ExecContext(ctx)
	// check whether a withdrawal with the same number has already been registered
	var exists bool
	err := conn.
		QueryRow(ctx, "SELECT EXISTS(SELECT id FROM withdrawals WHERE number=$1)", cw.Number).
		Scan(&exists)
	if err != nil {
		return models.Withdrawal{}, err
	} else if exists {
		log.Debug().
			Str("order", cw.Number).
			Msg("Withdrawal with same order number already exists in database")
		return models.Withdrawal{}, withdrawals.ErrWithdrawalAlreadyRegistered
	}

	// can withdraw a positive sum only
	if cw.Sum.LessThanOrEqual(decimal.Zero) {
		return models.Withdrawal{}, withdrawals.ErrWithdrawalMustHavePositiveSum
	}

	var newWithdrawalID int
	var actualProcessedAt time.Time
	err = conn.
		QueryRow(
			ctx,
			"INSERT INTO withdrawals (processed_at, user_id, number, sum) "+
				"VALUES ($1, $2, $3, $4) RETURNING id, processed_at",
			cw.ProcessedAt, cw.User.ID, cw.Number, cw.Sum,
		).
		Scan(&newWithdrawalID, &actualProcessedAt)

	if err != nil {
		log.Error().Err(err).Msg("Failed to add withdrawal")
		return models.Withdrawal{}, err
	}

	wd := models.NewAcceptedWithdrawal(newWithdrawalID, cw.Number, cw.User.ID, cw.Sum, actualProcessedAt)
	log.Debug().
		Str("number", wd.Number).Int("ID", newWithdrawalID).
		Msg("Registered new withdrawal")

	return wd, nil
}

func (r Repository) GetListForUser(ctx context.Context, userID int) ([]models.Withdrawal, error) {
	rows, err := r.db.ExecContext(ctx).Query(
		ctx,
		"SELECT id, processed_at, sum, number, user_id FROM withdrawals "+
			"WHERE user_id = $1 ORDER BY processed_at ASC",
		userID,
	)
	if err != nil {
		log.Error().Err(err).Int("userID", userID).Msg("Failed to query withdrawals for user")
		return nil, err
	}
	defer rows.Close()

	items := make([]models.Withdrawal, 0)
	for rows.Next() {
		row := withdrawalRow{}
		err = rows.Scan(&row.ID, &row.ProcessedAt, &row.Sum, &row.Number, &row.UserID)
		if err != nil {
			log.Error().Err(err).Int("userID", userID).Msg("Failed to read withdrawals for user")
			return nil, err
		}
		items = append(
			items,
			models.NewAcceptedWithdrawal(row.ID, row.Number, row.UserID, row.Sum, row.ProcessedAt),
		)
	}
	err = rows.Err()
	if err != nil {
		log.Error().Err(err).Int("userID", userID).Msg("Failed to fetch withdrawals for user")
		return nil, err
	}

	return items, nil
}
