package db

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(pgpool *pgxpool.Pool) Repository {
	return Repository{db: pgpool}
}

// Create attempts to insert a new user into the users table.
// User logins are unique and case-insensitive.
// Therefore, we can't allow the table to contain 2 logins "foobar" and "FooBar" simultaneously.
// This is forced on the database level with a constraint.
// Attempts to create a user with duplicate login would end with a user.ErrLoginIsAlreadyUsed error
// which must be handled by the calling code
func (r Repository) Create(ctx context.Context, u models.User) (models.User, error) {
	// force login to lower case
	login := strings.ToLower(u.Login)

	var exists bool
	err := r.db.
		QueryRow(ctx, "SELECT EXISTS(SELECT id FROM users WHERE lower(login)=$1)", login).
		Scan(&exists)
	if err != nil {
		return models.User{}, err
	} else if exists {
		log.Debug().Str("login", u.Login).Msg("user with same login already exists")
		return models.User{}, users.ErrUserLoginIsOccupied
	}

	var newUserID int
	err = r.db.
		QueryRow(ctx, "INSERT INTO users (login, password) values ($1, $2) RETURNING id", login, u.Password).
		Scan(&newUserID)
	if err != nil {
		log.Error().Err(err).Msg("failed to create new user")
		return models.User{}, err
	}

	log.Debug().Str("login", u.Login).Int("ID", newUserID).Msg("created new user")
	return models.User{
		ID:       newUserID,
		Login:    u.Login,
		Password: u.Password,
	}, nil
}

// GetByID attempts to retrieve a user by their ID
// Returns a models.User instance for the found user, or an error in case of a missing user with the given ID
func (r Repository) GetByID(ctx context.Context, id int) (models.User, error) {
	var userLogin, userPassword string
	row := r.db.QueryRow(ctx, "SELECT login, password FROM users WHERE id = $1", id)
	if err := row.Scan(&userLogin, &userPassword); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug().Int("ID", id).Msg("user not found")
			return models.User{}, users.ErrUserNotFoundInRepo
		}
		log.Error().Err(err).Int("ID", id).Msg("failed to query user by ID")
		return models.User{}, err
	}

	return models.User{
		ID:       id,
		Login:    userLogin,
		Password: userPassword,
	}, nil
}

// GetByLogin attempts to retrieve a user by their unique login username
// Just like its neighbour GetByID returns a models.User instance for the found user
func (r Repository) GetByLogin(ctx context.Context, login string) (models.User, error) {
	var userID int
	var userLogin, userPassword string
	row := r.db.QueryRow(
		ctx,
		"SELECT id, login, password FROM users WHERE lower(login) = $1",
		strings.ToLower(login),
	)
	if err := row.Scan(&userID, &userLogin, &userPassword); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Debug().Str("login", login).Msg("user not found")
			return models.User{}, users.ErrUserNotFoundInRepo
		}
		log.Error().Err(err).Str("login", login).Msg("failed to query user by login")
		return models.User{}, err
	}

	return models.User{
		ID:       userID,
		Login:    userLogin,
		Password: userPassword,
	}, nil
}
