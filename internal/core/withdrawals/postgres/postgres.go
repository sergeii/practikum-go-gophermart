package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/withdrawals"
	"github.com/sergeii/practikum-go-gophermart/internal/persistence/postgres"
)

type withdrawalRow struct {
	ID          int
	UserID      int
	Number      string
	Sum         decimal.Decimal
	ProcessedAt time.Time
}

type Repository struct {
	db *postgres.Database
}

func New(db *postgres.Database) Repository {
	return Repository{db}
}

func (r Repository) Add(ctx context.Context, cw withdrawals.Withdrawal) (withdrawals.Withdrawal, error) {
	conn := r.db.Conn(ctx)
	var newWithdrawalID int
	var actualProcessedAt time.Time
	err := conn.
		QueryRow(
			ctx,
			"INSERT INTO withdrawals (processed_at, user_id, number, sum) "+
				"VALUES ($1, $2, $3, $4) RETURNING id, processed_at",
			cw.ProcessedAt, cw.User.ID, cw.Number, cw.Sum,
		).
		Scan(&newWithdrawalID, &actualProcessedAt)

	if err != nil {
		log.Error().Err(err).Msg("Failed to add withdrawal")
		return withdrawals.Blank, err
	}

	wd := withdrawals.NewFromRepo(newWithdrawalID, cw.Number, cw.User.ID, cw.Sum, actualProcessedAt)
	log.Debug().
		Str("number", wd.Number).Int("ID", newWithdrawalID).
		Msg("Registered new withdrawal")

	return wd, nil
}

// GetByNumber attempts to find a withdrawal by an order number associated with it
func (r Repository) GetByNumber(ctx context.Context, number string) (withdrawals.Withdrawal, error) {
	var row withdrawalRow
	result := r.db.Conn(ctx).QueryRow(
		ctx,
		"SELECT id, user_id, sum, processed_at FROM withdrawals WHERE number = $1",
		number,
	)
	if err := result.Scan(&row.ID, &row.UserID, &row.Sum, &row.ProcessedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug().Str("number", number).Msg("Withdrawal not found in database")
			return withdrawals.Blank, withdrawals.ErrWithdrawalNotFound
		}
		log.Error().Err(err).Str("number", number).Msg("Failed to to retrieve withdrawal by ID")
		return withdrawals.Blank, err
	}
	return withdrawals.NewFromRepo(row.ID, number, row.UserID, row.Sum, row.ProcessedAt), nil
}

func (r Repository) GetListForUser(ctx context.Context, userID int) ([]withdrawals.Withdrawal, error) {
	var items []withdrawals.Withdrawal
	rows, err := r.db.Conn(ctx).Query(
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

	for rows.Next() {
		row := withdrawalRow{}
		err = rows.Scan(&row.ID, &row.ProcessedAt, &row.Sum, &row.Number, &row.UserID)
		if err != nil {
			log.Error().Err(err).Int("userID", userID).Msg("Failed to read withdrawals for user")
			return nil, err
		}
		items = append(
			items,
			withdrawals.NewFromRepo(row.ID, row.Number, row.UserID, row.Sum, row.ProcessedAt),
		)
	}
	err = rows.Err()
	if err != nil {
		log.Error().Err(err).Int("userID", userID).Msg("Failed to fetch withdrawals for user")
		return nil, err
	}

	return items, nil
}
