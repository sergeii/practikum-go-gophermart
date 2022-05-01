package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/sergeii/practikum-go-gophermart/internal/domain/user/repository/db"
	"github.com/sergeii/practikum-go-gophermart/internal/domain/user/service"
	"github.com/sergeii/practikum-go-gophermart/internal/pkg/testutils"
)

func TestService_RegisterNewUser_OK(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := db.New(pgpool)
	svc := service.New(repo, service.WithBcryptPasswordHasher())

	u, err := svc.RegisterNewUser(context.TODO(), "happy_customer", "sup3rS3cr3t")
	require.NoError(t, err)
	assert.True(t, u.ID > 0)
	assert.Equal(t, "happy_customer", u.Login)
	assert.Equal(t, "$2a$10", u.Password[:6]) // password is hashed

	u, _ = repo.GetByID(context.TODO(), u.ID)
	checkOK := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("sup3rS3cr3t"))
	assert.NoError(t, checkOK)

	checkWrong := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("maybesecret"))
	assert.ErrorIs(t, checkWrong, bcrypt.ErrMismatchedHashAndPassword)
}

func TestService_RegisterNewUser_Errors(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		password string
		wantErr  error
	}{
		{
			"positive case",
			"foo",
			"secret",
			nil,
		},
		{
			"duplicate login",
			"happy_customer",
			"secret",
			service.ErrRegisterLoginIsOccupied,
		},
		{
			"duplicate login in mixed case",
			"Happy_Customer",
			"secret",
			service.ErrRegisterLoginIsOccupied,
		},
		{
			"empty passsword",
			"bar",
			"",
			service.ErrRegisterEmptyPassword,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgpool, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			repo := db.New(pgpool)
			svc := service.New(repo, service.WithBcryptPasswordHasher())

			_, err := svc.RegisterNewUser(context.TODO(), "happy_customer", "sup3rS3cr3t")
			require.NoError(t, err)

			u, err := svc.RegisterNewUser(context.TODO(), tt.login, tt.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, 0, u.ID)
				assert.Equal(t, "", u.Login)
				assert.Equal(t, "", u.Password)
			} else {
				require.NoError(t, err)
				assert.True(t, u.ID > 0)
				assert.Equal(t, tt.login, u.Login)
				assert.Equal(t, "$2a$10", u.Password[:6])
			}
		})
	}
}

func TestService_Authenticate_OK(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := db.New(pgpool)
	svc := service.New(repo, service.WithBcryptPasswordHasher())

	u1, err := svc.RegisterNewUser(context.TODO(), "happy_customer", "sup3rS3cr3t")
	require.NoError(t, err)
	assert.True(t, u1.ID > 0)
	assert.Equal(t, "happy_customer", u1.Login)
	assert.Equal(t, "$2a$10", u1.Password[:6]) // password is hashed

	u2, err := svc.Authenticate(context.TODO(), "happy_customer", "sup3rS3cr3t")
	require.NoError(t, err)
	assert.Equal(t, u1.ID, u2.ID)
}

func TestService_Authenticate_Errors(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		password string
		wantErr  error
	}{
		{
			"positive case",
			"shopper",
			"sup3rS3cr3t",
			nil,
		},
		{
			"unknown user",
			"unknown",
			"sup3rS3cr3t",
			service.ErrAuthenticateInvalidCredentials,
		},
		{
			"empty password",
			"shopper",
			"",
			service.ErrAuthenticateEmptyPassword,
		},
		{
			"invalid password",
			"shopper",
			"guessing",
			service.ErrAuthenticateInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgpool, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			repo := db.New(pgpool)
			svc := service.New(repo, service.WithBcryptPasswordHasher())
			r, err := svc.RegisterNewUser(context.TODO(), "shopper", "sup3rS3cr3t")
			require.NoError(t, err)

			l, err := svc.Authenticate(context.TODO(), tt.login, tt.password)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, 0, l.ID)
				assert.Equal(t, "", l.Login)
			} else {
				require.NoError(t, err)
				assert.Equal(t, r.ID, l.ID)
				assert.Equal(t, tt.login, l.Login)
			}
		})
	}
}
