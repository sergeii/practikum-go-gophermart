package postgres_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sergeii/practikum-go-gophermart/internal/core/users"
	udb "github.com/sergeii/practikum-go-gophermart/internal/core/users/postgres"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/testutils"
)

func TestUsersRepository_Create_OK(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := udb.New(db)
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
			_, db, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			repo := udb.New(db)
			u1, err := repo.Create(context.TODO(), models.User{Login: "foobar", Password: "str0ng"})
			require.NoError(t, err)
			assert.True(t, u1.ID > 0)

			u2, err := repo.Create(context.TODO(), models.User{Login: tt.login, Password: "s3cret"})
			if tt.wantErr {
				require.ErrorIs(t, err, users.ErrUserLoginIsOccupied)
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
			_, db, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			repo := udb.New(db)

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
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := udb.New(db)
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
	require.ErrorIs(t, err, users.ErrUserNotFoundInRepo)
	assert.Equal(t, 0, u2.ID)
	assert.Equal(t, "", u2.Login)
	assert.Equal(t, "", u2.Password)
}

func TestUsersRepository_GetByLogin(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := udb.New(db)
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
	require.ErrorIs(t, err, users.ErrUserNotFoundInRepo)
	assert.Equal(t, 0, u2.ID)
	assert.Equal(t, "", u2.Login)
	assert.Equal(t, "", u2.Password)
}

func TestUsersDatabase_AccruePoints_OK(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := udb.New(db)
	u, _ := repo.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})

	err := repo.AccruePoints(context.TODO(), u.ID, decimal.RequireFromString("10.01"))
	assert.NoError(t, err)
	err = repo.AccruePoints(context.TODO(), u.ID, decimal.RequireFromString("9.99"))
	assert.NoError(t, err)

	u, _ = repo.GetByID(context.TODO(), u.ID)
	assert.Equal(t, "20", u.Balance.Current.String())
}

func TestUsersDatabase_AccruePoints_Race(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := udb.New(db)
	u1, _ := repo.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	u2, _ := repo.Create(context.TODO(), models.User{Login: "othercustomer", Password: "secr3t"})

	queue := []struct {
		userID int
		points string
	}{
		{u1.ID, "19.5"},
		{u1.ID, "0.1"},
		{u2.ID, "0"},
		{u1.ID, "104.99"},
		{u2.ID, "18"},
		{u1.ID, "5.9"},
		{u2.ID, "9.99"},
		{u2.ID, "100.99"},
	}
	wg := &sync.WaitGroup{}
	for _, item := range queue {
		wg.Add(1)
		go func(u int, p string) {
			err := repo.AccruePoints(context.TODO(), u, decimal.RequireFromString(p))
			assert.NoError(t, err)
			wg.Done()
		}(item.userID, item.points)
	}
	wg.Wait()

	u1, _ = repo.GetByID(context.TODO(), u1.ID)
	assert.Equal(t, "130.49", u1.Balance.Current.String())

	u2, _ = repo.GetByID(context.TODO(), u2.ID)
	assert.Equal(t, "128.98", u2.Balance.Current.String())
}

func TestUsersDatabase_WithdrawPoints_OK(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := udb.New(db)
	u, _ := repo.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})

	err := repo.AccruePoints(context.TODO(), u.ID, decimal.RequireFromString("20.01"))
	assert.NoError(t, err)
	err = repo.WithdrawPoints(context.TODO(), u.ID, decimal.RequireFromString("10.01"))
	assert.NoError(t, err)
	err = repo.WithdrawPoints(context.TODO(), u.ID, decimal.RequireFromString("9.99"))
	assert.NoError(t, err)

	u, _ = repo.GetByID(context.TODO(), u.ID)
	assert.Equal(t, "0.01", u.Balance.Current.String())
}

func TestUsersDatabase_WithdrawPoints_CannotWithdrawNegativeSum(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := udb.New(db)
	u, _ := repo.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})

	err := repo.AccruePoints(context.TODO(), u.ID, decimal.RequireFromString("50"))
	assert.NoError(t, err)
	err = repo.WithdrawPoints(context.TODO(), u.ID, decimal.RequireFromString("10"))
	assert.NoError(t, err)
	err = repo.WithdrawPoints(context.TODO(), u.ID, decimal.RequireFromString("-10"))
	assert.ErrorIs(t, err, users.ErrUserCantWithdrawNegativeSum)

	u, _ = repo.GetByID(context.TODO(), u.ID)
	assert.Equal(t, "40", u.Balance.Current.String())
}

func TestUsersDatabase_WithdrawPoints_CannotWithdrawMoreThanHave(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := udb.New(db)
	u, _ := repo.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})

	err := repo.AccruePoints(context.TODO(), u.ID, decimal.RequireFromString("20.01"))
	assert.NoError(t, err)
	err = repo.WithdrawPoints(context.TODO(), u.ID, decimal.RequireFromString("10.01"))
	assert.NoError(t, err)
	err = repo.WithdrawPoints(context.TODO(), u.ID, decimal.RequireFromString("9.99"))
	assert.NoError(t, err)
	err = repo.WithdrawPoints(context.TODO(), u.ID, decimal.RequireFromString("0.02"))
	assert.ErrorIs(t, err, users.ErrUserHasInsufficientAccrual)

	u, _ = repo.GetByID(context.TODO(), u.ID)
	assert.Equal(t, "0.01", u.Balance.Current.String())
}

func TestUsersDatabase_WithdrawPoints_CannotWithdrawMoreThanHave_Race(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := udb.New(db)
	u, _ := repo.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})

	err := repo.AccruePoints(context.TODO(), u.ID, decimal.RequireFromString("5.0"))
	assert.NoError(t, err)

	wg := &sync.WaitGroup{}
	var errCount int64
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			err := repo.WithdrawPoints(context.TODO(), u.ID, decimal.RequireFromString("1.5"))
			if err != nil {
				assert.ErrorIs(t, err, users.ErrUserHasInsufficientAccrual)
				atomic.AddInt64(&errCount, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	u, _ = repo.GetByID(context.TODO(), u.ID)
	assert.Equal(t, "0.5", u.Balance.Current.String())
	assert.Equal(t, 7, int(errCount))
}
