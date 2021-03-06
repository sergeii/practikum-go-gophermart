package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
	"github.com/sergeii/practikum-go-gophermart/internal/persistence/postgres"
)

type Repository struct {
	db *postgres.Database
}

func New(db *postgres.Database) Repository {
	return Repository{db}
}

// Create attempts to insert a new user into the users table.
// User logins are unique and case-insensitive.
// Therefore, we can't allow the table to contain 2 logins "foobar" and "FooBar" simultaneously.
// This is forced on the database level with a constraint.
// Attempts to create a user with duplicate login would end with a user.ErrLoginIsAlreadyUsed error
// which must be handled by the calling code
func (r Repository) Create(ctx context.Context, u users.User) (users.User, error) {
	conn := r.db.Conn(ctx)
	// force login to lower case
	login := strings.ToLower(u.Login)
	var newUserID int
	err := conn.
		QueryRow(ctx, "INSERT INTO users (login, password) values ($1, $2) RETURNING id", login, u.Password).
		Scan(&newUserID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create new user")
		return users.Blank, err
	}

	log.Debug().Str("login", u.Login).Int("ID", newUserID).Msg("Created new user")
	return users.NewFromRepo(newUserID, login, u.Password, decimal.Zero, decimal.Zero), nil
}

// GetByID attempts to retrieve a user by their ID
// Returns a users.User instance for the found user, or an error in case of a missing user with the given ID
func (r Repository) GetByID(ctx context.Context, id int) (users.User, error) {
	var u users.User
	row := r.db.Conn(ctx).QueryRow(
		ctx,
		"SELECT id, login, password, balance_current, balance_withdrawn FROM users WHERE id = $1",
		id,
	)
	if err := row.Scan(&u.ID, &u.Login, &u.Password, &u.Balance.Current, &u.Balance.Withdrawn); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug().Int("ID", id).Msg("User not found")
			return users.Blank, users.ErrUserNotFound
		}
		log.Error().Err(err).Int("ID", id).Msg("Failed to query user by ID")
		return users.Blank, err
	}

	return u, nil
}

// GetByLogin attempts to retrieve a user by their unique login username
// Just like its neighbour GetByID returns a users.User instance for the found user
func (r Repository) GetByLogin(ctx context.Context, login string) (users.User, error) {
	var u users.User
	row := r.db.Conn(ctx).QueryRow(
		ctx,
		"SELECT id, login, password, balance_current, balance_withdrawn FROM users WHERE lower(login) = $1",
		strings.ToLower(login),
	)
	if err := row.Scan(&u.ID, &u.Login, &u.Password, &u.Balance.Current, &u.Balance.Withdrawn); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug().Str("login", login).Msg("User not found")
			return users.Blank, users.ErrUserNotFound
		}
		log.Error().Err(err).Str("login", login).Msg("Failed to query user by login")
		return users.Blank, err
	}
	return u, nil
}

// AccruePoints accrues specified amount of points for specified user
func (r Repository) AccruePoints(ctx context.Context, userID int, points decimal.Decimal) error {
	return r.db.WithTransaction(ctx, func(txCtx context.Context) error {
		var oldCurrent, newCurrent decimal.Decimal
		tx := r.db.Conn(txCtx)
		if err := tx.QueryRow(
			txCtx, "SELECT balance_current FROM users WHERE id = $1 FOR UPDATE", userID,
		).Scan(&oldCurrent); err != nil {
			log.Error().Err(err).Int("userID", userID).Msg("Unable to acquire row lock for user")
			return err
		}
		if err := tx.QueryRow(
			txCtx,
			"UPDATE users SET balance_current = balance_current + $1 WHERE id = $2 RETURNING balance_current",
			points, userID,
		).Scan(&newCurrent); err != nil {
			return err
		}
		log.Info().
			Int("userID", userID).
			Stringer("points", points).
			Stringer("before", oldCurrent).
			Stringer("after", newCurrent).
			Msg("Points accrued for user")
		return nil
	})
}

func (r Repository) WithdrawPoints(ctx context.Context, userID int, points decimal.Decimal) error {
	return r.db.WithTransaction(ctx, func(txCtx context.Context) error {
		var oldCurrent, newCurrent, oldWithdrawn, newWithdrawn decimal.Decimal
		tx := r.db.Conn(txCtx)
		if err := tx.QueryRow(
			txCtx,
			"SELECT balance_current, balance_withdrawn FROM users WHERE id = $1 FOR UPDATE",
			userID,
		).Scan(&oldCurrent, &oldWithdrawn); err != nil {
			log.Error().Err(err).Int("userID", userID).Msg("Unable to acquire row lock for user")
			return err
		}
		// it's impossible to withdraw more points than the user owns
		if oldCurrent.LessThan(points) {
			return users.ErrUserHasInsufficientBalance
		}
		if err := tx.QueryRow(
			txCtx,
			"UPDATE users SET "+
				"balance_current = balance_current - $1, balance_withdrawn = balance_withdrawn + $1 "+
				"WHERE id = $2 RETURNING balance_current, balance_withdrawn",
			points, userID,
		).Scan(&newCurrent, &newWithdrawn); err != nil {
			return err
		}
		log.Info().
			Int("userID", userID).
			Stringer("points", points).
			Stringer("withdrawnBefore", oldWithdrawn).
			Stringer("withdrawnAfter", newWithdrawn).
			Stringer("currentBefore", oldCurrent).
			Stringer("currentAfter", newCurrent).
			Msg("Points withdrawn for user")
		return nil
	})
}
