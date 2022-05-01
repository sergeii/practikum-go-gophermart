package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sergeii/practikum-go-gophermart/internal/domain/user/repository"
	"github.com/sergeii/practikum-go-gophermart/internal/domain/user/repository/db"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/pkg/testutils"
)

func TestUsersRepository_Create_OK(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := db.New(pgpool)
	u, err := repo.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	require.NoError(t, err)
	assert.True(t, u.ID > 0)
	assert.Equal(t, "happycustomer", u.Login)
	assert.Equal(t, "str0ng", u.Password) // user repository is not hashing passwords
}

func TestUsersRepository_Create_ErrorOnDuplicate(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		wantErr bool
	}{
		{
			"positive case #1",
			"other",
			false,
		},
		{
			"positive case #2",
			"another",
			false,
		},
		{
			"same case",
			"foobar",
			true,
		},
		{
			"upper case",
			"FOOBAR",
			true,
		},
		{
			"mixed case",
			"FooBaR",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgpool, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			repo := db.New(pgpool)
			u1, err := repo.Create(context.TODO(), models.User{Login: "foobar", Password: "str0ng"})
			require.NoError(t, err)
			assert.True(t, u1.ID > 0)

			u2, err := repo.Create(context.TODO(), models.User{Login: tt.login, Password: "s3cret"})
			if tt.wantErr {
				require.ErrorIs(t, err, repository.ErrUserLoginIsOccupied)
				assert.Equal(t, 0, u2.ID)
			} else {
				require.NoError(t, err)
				assert.True(t, u2.ID > u1.ID)
			}
		})
	}
}

func TestUsersRepository_Create_FieldsMustNotBeEmpty(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		password string
		wantErr  bool
	}{
		{
			"positive case",
			"foo",
			"secret",
			false,
		},
		{
			"password cannot be empty",
			"foo",
			"",
			true,
		},
		{
			"login cannot be empty",
			"",
			"secret",
			true,
		},
		{
			"login and password cannot be empty",
			"",
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgpool, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			repo := db.New(pgpool)

			u, err := repo.Create(context.TODO(), models.User{Login: tt.login, Password: tt.password})
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, 0, u.ID)
			} else {
				require.NoError(t, err)
				assert.True(t, u.ID > 0)
			}
		})
	}
}

func TestUsersRepository_GetByID(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := db.New(pgpool)
	u, err := repo.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	require.NoError(t, err)
	assert.True(t, u.ID > 0)
	assert.Equal(t, "happycustomer", u.Login)
	assert.Equal(t, "str0ng", u.Password)

	u1, err := repo.GetByID(context.TODO(), u.ID)
	require.NoError(t, err)
	assert.Equal(t, "happycustomer", u1.Login)
	assert.Equal(t, "str0ng", u1.Password)

	u2, err := repo.GetByID(context.TODO(), 999999)
	require.ErrorIs(t, err, repository.ErrUserNotFoundInRepo)
	assert.Equal(t, 0, u2.ID)
	assert.Equal(t, "", u2.Login)
	assert.Equal(t, "", u2.Password)
}

func TestUsersRepository_GetByLogin(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := db.New(pgpool)
	u, err := repo.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	require.NoError(t, err)
	assert.True(t, u.ID > 0)
	assert.Equal(t, "happycustomer", u.Login)
	assert.Equal(t, "str0ng", u.Password)

	u1, err := repo.GetByLogin(context.TODO(), "happycustomer")
	require.NoError(t, err)
	assert.Equal(t, "happycustomer", u1.Login)
	assert.Equal(t, "str0ng", u1.Password)

	u2, err := repo.GetByLogin(context.TODO(), "unknown")
	require.ErrorIs(t, err, repository.ErrUserNotFoundInRepo)
	assert.Equal(t, 0, u2.ID)
	assert.Equal(t, "", u2.Login)
	assert.Equal(t, "", u2.Password)
}
