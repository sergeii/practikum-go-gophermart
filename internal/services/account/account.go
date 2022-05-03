package account

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"

	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/pkg/security/hasher"
)

var ErrRegisterEmptyPassword = errors.New("cannot register with empty password")
var ErrRegisterLoginIsOccupied = errors.New("cannot register with occupied login")

var ErrAuthenticateEmptyPassword = errors.New("cannot login with empty password")
var ErrAuthenticateInvalidCredentials = errors.New("unable to authenticate user with this login/password")

type Service struct {
	users  users.Repository
	hasher hasher.PasswordHasher
}

type Option func(s *Service)

func WithPasswordHasher(h hasher.PasswordHasher) Option {
	return func(s *Service) {
		s.hasher = h
	}
}

func WithBcryptPasswordHasher() Option {
	return WithPasswordHasher(hasher.NewBcryptPasswordHasher())
}

func New(users users.Repository, opts ...Option) Service {
	s := Service{
		users: users,
		// set defaults
		hasher: hasher.NewNoopPasswordHasher(),
	}
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

// RegisterNewUser attempts to register a new user with the current repository.
// Before saving the user into the repository, the raw password is hashed using the service configured hasher.
// The user is therefore saved with their password hashed
func (s Service) RegisterNewUser(ctx context.Context, login, password string) (models.User, error) {
	// must not register with empty password
	if password == "" {
		return models.User{}, ErrRegisterEmptyPassword
	}
	// store a password hash instead of the plain password
	hashedPassword, err := s.hasher.Hash(password)
	if err != nil {
		log.Debug().Err(err).Str("login", login).Msg("unable to hash password")
		return models.User{}, err
	}

	newUser := models.User{Login: login, Password: hashedPassword}
	u, err := s.users.Create(ctx, newUser)
	if err != nil {
		if errors.Is(err, users.ErrUserLoginIsOccupied) {
			return models.User{}, ErrRegisterLoginIsOccupied
		}
		return models.User{}, err
	}
	return u, nil
}

// Authenticate attempts to log in a user using provided credentials
func (s Service) Authenticate(ctx context.Context, login, password string) (models.User, error) {
	// prevent logging in with an empty password
	if password == "" {
		return models.User{}, ErrAuthenticateEmptyPassword
	}

	user, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFoundInRepo) {
			return models.User{}, ErrAuthenticateInvalidCredentials
		}
		return models.User{}, err
	}

	passwordsMatch, err := s.hasher.Check(password, user.Password)
	if err != nil {
		log.Error().Err(err).Str("login", login).Msg("unable to check password")
	} else if !passwordsMatch {
		log.Debug().Str("login", login).Msg("password does not match")
		return models.User{}, ErrAuthenticateInvalidCredentials
	}

	return user, nil
}
