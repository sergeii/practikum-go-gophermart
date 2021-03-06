package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/orders"
	"github.com/sergeii/practikum-go-gophermart/internal/persistence/postgres"
)

type orderRow struct {
	ID         int
	UserID     int
	Number     string
	Status     orders.OrderStatus
	Accrual    decimal.Decimal
	UploadedAt time.Time
}

type Repository struct {
	db *postgres.Database
}

func New(db *postgres.Database) Repository {
	return Repository{db}
}

// Add attempts to insert a new order.
// A new order cannot be added in case of another order having the same number.
// In that case an error is returned
func (r Repository) Add(ctx context.Context, co orders.Order) (orders.Order, error) {
	conn := r.db.Conn(ctx)
	var newOrderID int
	var actualUploadedAt time.Time
	err := conn.
		QueryRow(
			ctx,
			"INSERT INTO orders (uploaded_at, user_id, number, status, accrual) "+
				"VALUES ($1, $2, $3, $4, $5) RETURNING id, uploaded_at",
			co.UploadedAt, co.User.ID, co.Number, co.Status, co.Accrual,
		).
		Scan(&newOrderID, &actualUploadedAt)

	if err != nil {
		log.Error().Err(err).Msg("Failed to add order")
		return orders.Blank, err
	}

	order := orders.NewFromRepo(newOrderID, co.Number, co.User.ID, co.Status, co.Accrual, actualUploadedAt)
	log.Debug().
		Str("number", order.Number).Int("ID", newOrderID).
		Msg("Added new order")

	return order, nil
}

// GetByNumber attempts to find and return an order by its external number
func (r Repository) GetByNumber(ctx context.Context, number string) (orders.Order, error) {
	var row orderRow
	result := r.db.Conn(ctx).QueryRow(
		ctx,
		"SELECT id, user_id, uploaded_at, status, accrual FROM orders WHERE number = $1",
		number,
	)
	if err := result.Scan(&row.ID, &row.UserID, &row.UploadedAt, &row.Status, &row.Accrual); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug().Str("number", number).Msg("Order not found in database")
			return orders.Blank, orders.ErrOrderNotFound
		}
		log.Error().Err(err).Str("number", number).Msg("Failed to retrieve order from database by ID")
		return orders.Blank, err
	}
	return orders.NewFromRepo(row.ID, number, row.UserID, row.Status, row.Accrual, row.UploadedAt), nil
}

// GetListForUser returns a list of orders uploaded by specified user.
// The orders are sorted from the oldest to the newest
func (r Repository) GetListForUser(ctx context.Context, userID int) ([]orders.Order, error) {
	var items []orders.Order
	rows, err := r.db.Conn(ctx).Query(
		ctx,
		"SELECT id, uploaded_at, status, accrual, number, user_id FROM orders "+
			"WHERE user_id = $1 ORDER BY uploaded_at ASC",
		userID,
	)
	if err != nil {
		log.Error().Err(err).Int("userID", userID).Msg("Failed to query orders for user")
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		row := orderRow{}
		err = rows.Scan(&row.ID, &row.UploadedAt, &row.Status, &row.Accrual, &row.Number, &row.UserID)
		if err != nil {
			log.Error().Err(err).Int("userID", userID).Msg("Failed to scan order row")
			return nil, err
		}
		items = append(
			items,
			orders.NewFromRepo(row.ID, row.Number, row.UserID, row.Status, row.Accrual, row.UploadedAt),
		)
	}
	err = rows.Err()
	if err != nil {
		log.Error().Err(err).Int("userID", userID).Msg("Failed to fetch orders for user")
		return nil, err
	}

	return items, nil
}

func (r Repository) Update(ctx context.Context, orderID int, o orders.Order) error {
	return r.db.WithTransaction(ctx, func(txCtx context.Context) error {
		tx := r.db.Conn(txCtx)
		// ensure that we update the order exclusively
		if _, err := tx.Exec(txCtx, "SELECT 1 FROM orders WHERE id = $1 FOR UPDATE NOWAIT", orderID); err != nil {
			log.Error().Err(err).Int("orderID", orderID).Msg("Unable to acquire row lock for order")
			return err
		}
		_, err := tx.Exec(
			txCtx,
			"UPDATE orders SET status = $1, accrual = $2 WHERE id = $3",
			o.Status, o.Accrual, orderID,
		)
		if err != nil {
			return err
		}

		return nil
	})
}
