package db

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/persistence/db"
)

type Repository struct {
	db *db.Database
}

func New(db *db.Database) Repository {
	return Repository{db}
}

// Create attempts to insert a new user into the users table.
// User logins are unique and case-insensitive.
// Therefore, we can't allow the table to contain 2 logins "foobar" and "FooBar" simultaneously.
// This is forced on the database level with a constraint.
// Attempts to create a user with duplicate login would end with a user.ErrLoginIsAlreadyUsed error
// which must be handled by the calling code
func (r Repository) Create(ctx context.Context, u models.User) (models.User, error) {
	conn := r.db.ExecContext(ctx)
	// force login to lower case
	login := strings.ToLower(u.Login)

	var exists bool
	err := conn.
		QueryRow(ctx, "SELECT EXISTS(SELECT id FROM users WHERE lower(login)=$1)", login).
		Scan(&exists)
	if err != nil {
		return models.User{}, err
	} else if exists {
		log.Debug().Str("login", u.Login).Msg("User with same login already exists")
		return models.User{}, users.ErrUserLoginIsOccupied
	}

	var newUserID int
	err = conn.
		QueryRow(ctx, "INSERT INTO users (login, password) values ($1, $2) RETURNING id", login, u.Password).
		Scan(&newUserID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create new user")
		return models.User{}, err
	}

	log.Debug().Str("login", u.Login).Int("ID", newUserID).Msg("Created new user")
	return models.User{
		ID:       newUserID,
		Login:    u.Login,
		Password: u.Password,
	}, nil
}

// GetByID attempts to retrieve a user by their ID
// Returns a models.User instance for the found user, or an error in case of a missing user with the given ID
func (r Repository) GetByID(ctx context.Context, id int) (models.User, error) {
	var user models.User

	row := r.db.ExecContext(ctx).QueryRow(
		ctx,
		"SELECT id, login, password, balance_current, balance_withdrawn FROM users WHERE id = $1",
		id,
	)
	if err := row.Scan(
		&user.ID, &user.Login, &user.Password,
		&user.Balance.Current, &user.Balance.Withdrawn,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug().Int("ID", id).Msg("User not found")
			return models.User{}, users.ErrUserNotFoundInRepo
		}
		log.Error().Err(err).Int("ID", id).Msg("Failed to query user by ID")
		return models.User{}, err
	}

	return user, nil
}

// GetByLogin attempts to retrieve a user by their unique login username
// Just like its neighbour GetByID returns a models.User instance for the found user
func (r Repository) GetByLogin(ctx context.Context, login string) (models.User, error) {
	var user models.User

	row := r.db.ExecContext(ctx).QueryRow(
		ctx,
		"SELECT id, login, password, balance_current, balance_withdrawn FROM users WHERE lower(login) = $1",
		strings.ToLower(login),
	)
	if err := row.Scan(
		&user.ID, &user.Login, &user.Password,
		&user.Balance.Current, &user.Balance.Withdrawn,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug().Str("login", login).Msg("User not found")
			return models.User{}, users.ErrUserNotFoundInRepo
		}
		log.Error().Err(err).Str("login", login).Msg("Failed to query user by login")
		return models.User{}, err
	}

	return user, nil
}

// AccruePoints accrues specified amount of points for specified user
func (r Repository) AccruePoints(ctx context.Context, userID int, points decimal.Decimal) error {
	return r.db.WithTransaction(ctx, func(txCtx context.Context) error {
		var oldCurrent, newCurrent decimal.Decimal
		tx := r.db.ExecContext(txCtx)
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

// WithdrawPoints attempts to withdraw specified amount of points from the user's current balance
// incrementing the amount of withdrawn points and deducting the amount of current points.
// Since you cannot withdraw more than you have, an error is returned if such an attempt is made
func (r Repository) WithdrawPoints(ctx context.Context, userID int, points decimal.Decimal) error {
	return r.db.WithTransaction(ctx, func(txCtx context.Context) error {
		var oldCurrent, newCurrent, oldWithdrawn, newWithdrawn decimal.Decimal
		tx := r.db.ExecContext(txCtx)
		if err := tx.QueryRow(
			txCtx,
			"SELECT balance_current, balance_withdrawn FROM users WHERE id = $1 FOR UPDATE",
			userID,
		).Scan(&oldCurrent, &oldWithdrawn); err != nil {
			log.Error().Err(err).Int("userID", userID).Msg("Unable to acquire row lock for user")
			return err
		}
		// cannot withdraw more points than the user owns
		if oldCurrent.LessThan(points) {
			return users.ErrUserHasInsufficientAccrual
		}
		// cannot withdraw negative sums
		if points.LessThanOrEqual(decimal.Zero) {
			return users.ErrUserCantWithdrawNegativeSum
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
