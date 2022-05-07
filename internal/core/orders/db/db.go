package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/internal/core/orders"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(pgpool *pgxpool.Pool) Repository {
	return Repository{db: pgpool}
}

// Add attempts to insert a new order.
// A new order cannot be added in case of another order having the same number.
// In that case an error is returned
func (r Repository) Add(ctx context.Context, o models.Order, action func(models.Order) error) (models.Order, error) {
	var exists bool
	err := r.db.
		QueryRow(ctx, "SELECT EXISTS(SELECT id FROM orders WHERE number=$1)", o.Number).
		Scan(&exists)
	if err != nil {
		return models.Order{}, err
	} else if exists {
		log.Debug().Str("order", o.Number).Msg("order with same number already exists")
		return models.Order{}, orders.ErrOrderAlreadyExists
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Order{}, err
	}
	defer tx.Rollback(ctx) // nolint: errcheck

	var newOrderID int
	err = tx.
		QueryRow(
			ctx,
			"INSERT INTO orders (uploaded_at, user_id, number, status, accrual) "+
				"VALUES ($1, $2, $3, $4, $5) RETURNING id",
			o.UploadedAt, o.User.ID, o.Number, o.Status, o.Accrual,
		).
		Scan(&newOrderID)
	if err != nil {
		log.Error().Err(err).Msg("failed to add order")
		return models.Order{}, err
	}
	o.ID = newOrderID
	log.Debug().
		Str("number", o.Number).Int("ID", newOrderID).
		Msg("added new order")

	if err = action(o); err != nil {
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			return models.Order{}, errRollback
		}
		return models.Order{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return models.Order{}, err
	}

	return o, nil
}

// GetByNumber attempts to find and return an order by its external number
func (r Repository) GetByNumber(ctx context.Context, number string) (models.Order, error) {
	var orderID, userID int
	var status models.OrderStatus
	var uploadedAt time.Time
	var accrual float64

	row := r.db.QueryRow(
		ctx,
		"SELECT id, user_id, uploaded_at, status, accrual FROM orders WHERE number = $1",
		number,
	)
	if err := row.Scan(&orderID, &userID, &uploadedAt, &status, &accrual); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug().Str("number", number).Msg("Order not found in database")
			return models.Order{}, orders.ErrOrderNotFound
		}
		log.Error().Err(err).Str("number", number).Msg("Failed to retrieve order from database by ID")
		return models.Order{}, err
	}

	return models.Order{
		ID:         orderID,
		User:       models.User{ID: userID},
		UploadedAt: uploadedAt,
		Status:     status,
		Accrual:    accrual,
	}, nil
}

// GetListForUser returns a list of orders uploaded by specified user.
// The orders are sorted from the oldest to the newest
func (r Repository) GetListForUser(ctx context.Context, userID int) ([]models.Order, error) {
	rows, err := r.db.Query(
		ctx,
		"SELECT id, uploaded_at, status, accrual, number, user_id FROM orders "+
			"WHERE user_id = $1 ORDER BY uploaded_at ASC",
		userID,
	)
	if err != nil {
		log.Error().Err(err).Int("userID", userID).Msg("failed to query orders for user")
		return nil, err
	}
	defer rows.Close()

	items := make([]models.Order, 0)
	for rows.Next() {
		item := models.Order{}
		err = rows.Scan(&item.ID, &item.UploadedAt, &item.Status, &item.Accrual, &item.Number, &item.User.ID)
		if err != nil {
			log.Error().Err(err).Int("userID", userID).Msg("failed to scan order row")
			return nil, err
		}
		items = append(items, item)
	}
	err = rows.Err()
	if err != nil {
		log.Error().Err(err).Int("userID", userID).Msg("failed to fetch orders for user")
		return nil, err
	}

	return items, nil
}

func (r Repository) UpdateStatus(ctx context.Context, orderID int, status models.OrderStatus, accrual float64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // nolint: errcheck

	// ensure that we update the order exclusively
	if _, err = tx.Exec(ctx, "SELECT 1 FROM orders WHERE id = $1 FOR UPDATE", orderID); err != nil {
		log.Error().Err(err).Int("orderID", orderID).Msg("Unable to acquire row lock for order")
		return err
	}

	if _, err = tx.Exec(
		ctx,
		"UPDATE orders SET status = $1, accrual = $2 WHERE id = $3",
		status, accrual, orderID,
	); err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}
