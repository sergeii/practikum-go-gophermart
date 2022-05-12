package account

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
	"github.com/sergeii/practikum-go-gophermart/pkg/security/hasher"
)

var ErrRegisterEmptyPassword = errors.New("cannot register with empty password")
var ErrRegisterLoginOccupied = errors.New("login is occupied by another user")

var ErrAuthenticateEmptyPassword = errors.New("cannot login with empty password")
var ErrAuthenticateInvalidCredentials = errors.New("unable to authenticate user with this login/password")

var ErrWithdrawInvalidSum = errors.New("user can withdraw positive sum only")

type Service struct {
	users  users.Repository
	hasher hasher.PasswordHasher
}

func New(repo users.Repository, hasher hasher.PasswordHasher) Service {
	return Service{
		users:  repo,
		hasher: hasher,
	}
}

// RegisterNewUser attempts to register a new user with the current repository.
// Before saving the user into the repository, the raw password is hashed using the service configured hasher.
// The user is therefore saved with their password hashed
func (s Service) RegisterNewUser(ctx context.Context, login, password string) (users.User, error) {
	// must not register with empty password
	if password == "" {
		return users.Blank, ErrRegisterEmptyPassword
	}
	// check whether a user with this login already exists
	if _, err := s.users.GetByLogin(ctx, login); err == nil {
		return users.Blank, ErrRegisterLoginOccupied
	}
	// store a password hash instead of the plain password
	hashedPassword, err := s.hasher.Hash(password)
	if err != nil {
		log.Debug().Err(err).Str("login", login).Msg("Unable to hash password")
		return users.Blank, err
	}

	newUser := users.New(login, hashedPassword)
	u, err := s.users.Create(ctx, newUser)
	if err != nil {
		return users.Blank, err
	}
	return u, nil
}

// Authenticate attempts to log in a user using provided credentials
func (s Service) Authenticate(ctx context.Context, login, password string) (users.User, error) {
	// prevent logging in with an empty password
	if password == "" {
		return users.Blank, ErrAuthenticateEmptyPassword
	}
	user, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return users.Blank, ErrAuthenticateInvalidCredentials
		}
		return users.Blank, err
	}

	passwordsMatch, err := s.hasher.Check(password, user.Password)
	if err != nil {
		log.Error().Err(err).Str("login", login).Msg("Unable to check password")
	} else if !passwordsMatch {
		log.Debug().Str("login", login).Msg("Password does not match")
		return users.Blank, ErrAuthenticateInvalidCredentials
	}

	return user, nil
}

func (s Service) AccruePoints(ctx context.Context, userID int, points decimal.Decimal) error {
	return s.users.AccruePoints(ctx, userID, points)
}

// WithdrawPoints attempts to withdraw specified amount of points from the user's current balance
// incrementing the amount of withdrawn points and deducting the amount of current points.
// Since you cannot withdraw more than you have, an error is returned if such an attempt is made
func (s Service) WithdrawPoints(ctx context.Context, userID int, points decimal.Decimal) error {
	// cannot withdraw negative sums
	if points.LessThanOrEqual(decimal.Zero) {
		return ErrWithdrawInvalidSum
	}
	return s.users.WithdrawPoints(ctx, userID, points)
}

// GetBalance returns specified user's balance
func (s Service) GetBalance(ctx context.Context, userID int) (users.UserBalance, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return users.Blank.Balance, err
	}
	return u.Balance, nil
}
