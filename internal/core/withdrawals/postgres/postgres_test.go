package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	urepo "github.com/sergeii/practikum-go-gophermart/internal/core/users"
	udb "github.com/sergeii/practikum-go-gophermart/internal/core/users/postgres"
	"github.com/sergeii/practikum-go-gophermart/internal/core/withdrawals"
	wdb "github.com/sergeii/practikum-go-gophermart/internal/core/withdrawals/postgres"
	"github.com/sergeii/practikum-go-gophermart/internal/testutils"
)

func TestWithdrawalsDatabase_Add_OK(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	u, err := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))
	require.NoError(t, err)

	before := time.Now()
	repo := wdb.New(db)
	w, err := repo.Add(
		context.TODO(),
		withdrawals.New("1234567812345670", u.ID, decimal.RequireFromString("9.99")),
	)
	require.NoError(t, err)
	assert.True(t, w.ID > 0)
	assert.Equal(t, "1234567812345670", w.Number)
	assert.True(t, !w.ProcessedAt.Before(before))
	assert.Equal(t, u.ID, w.User.ID)
}

func TestWithdrawalsDatabase_Add_ErrorOnDuplicate(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	u, err := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))
	require.NoError(t, err)

	before := time.Now()
	repo := wdb.New(db)
	w, err := repo.Add(
		context.TODO(),
		withdrawals.New("1234567812345670", u.ID, decimal.RequireFromString("9.99")),
	)
	require.NoError(t, err)
	assert.True(t, w.ID > 0)
	assert.Equal(t, "1234567812345670", w.Number)
	assert.True(t, !w.ProcessedAt.Before(before))
	assert.Equal(t, u.ID, w.User.ID)

	w2, err := repo.Add(
		context.TODO(),
		withdrawals.New("1234567812345670", u.ID, decimal.RequireFromString("19.99")),
	)
	assert.ErrorContains(t, err, "duplicate key value violates unique constraint")
	assert.Equal(t, 0, w2.ID)
}

func TestWithdrawalsDatabase_Add_ErrorOnForeignKeyMissing(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := wdb.New(db)
	w, err := repo.Add(
		context.TODO(),
		withdrawals.New("1234567812345670", 999999, decimal.RequireFromString("9.99")),
	)
	require.Error(t, err)
	require.ErrorContains(t, err, `violates foreign key constraint "withdrawals_user_id_fk_users"`)
	assert.Equal(t, 0, w.ID)
}

func TestWithdrawalsDatabase_Add_ErrorOnNegativeSum(t *testing.T) {
	tests := []struct {
		name string
		sum  string
	}{
		{
			"zero sum",
			"0",
		},
		{
			"negative sum",
			"-10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, db, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			users := udb.New(db)
			u, _ := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))

			repo := wdb.New(db)
			w, err := repo.Add(
				context.TODO(),
				withdrawals.New("1234567812345670", u.ID, decimal.RequireFromString(tt.sum)),
			)
			assert.ErrorContains(t, err, `violates check constraint "withdrawals_sum_check"`)
			assert.Equal(t, 0, w.ID)
		})
	}
}

func TestWithdrawalsDatabase_GetListForUser_OK(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	before := time.Now()

	users := udb.New(db)
	u, _ := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))

	repo := wdb.New(db)
	for _, number := range []string{"1234567812345670", "4561261212345467", "49927398716"} {
		_, err := repo.Add(
			context.TODO(),
			withdrawals.New(number, u.ID, decimal.RequireFromString("9.99")),
		)
		require.NoError(t, err)
	}

	userWithdrawals, err := repo.GetListForUser(context.TODO(), u.ID)
	require.NoError(t, err)
	assert.Len(t, userWithdrawals, 3)
	for _, o := range userWithdrawals {
		assert.Equal(t, u.ID, o.User.ID)
		assert.True(t, o.ProcessedAt.After(before))
	}
	assert.Equal(t, "1234567812345670", userWithdrawals[0].Number)
	assert.Equal(t, "4561261212345467", userWithdrawals[1].Number)
	assert.Equal(t, "49927398716", userWithdrawals[2].Number)
}

func TestWithdrawalsDatabase_GetListForUser_NoErrorForUnknownUser(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()
	repo := wdb.New(db)
	userWithdrawals, err := repo.GetListForUser(context.TODO(), 9999999)
	require.NoError(t, err)
	assert.Len(t, userWithdrawals, 0)
}
