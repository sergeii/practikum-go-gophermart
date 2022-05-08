package withdrawal_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	urepo "github.com/sergeii/practikum-go-gophermart/internal/core/users"
	udb "github.com/sergeii/practikum-go-gophermart/internal/core/users/db"
	wrepo "github.com/sergeii/practikum-go-gophermart/internal/core/withdrawals"
	wdb "github.com/sergeii/practikum-go-gophermart/internal/core/withdrawals/db"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/pkg/testutils"
	"github.com/sergeii/practikum-go-gophermart/internal/services/withdrawal"
)

func TestWithdrawalService_RequestWithdrawal_OK(t *testing.T) {
	ctx := context.TODO()
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)

	u1, _ := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	err := users.AccruePoints(context.TODO(), u1.ID, decimal.RequireFromString("10"))
	require.NoError(t, err)
	u2, _ := users.Create(context.TODO(), models.User{Login: "shopper", Password: "secr3t"})
	err = users.AccruePoints(context.TODO(), u2.ID, decimal.RequireFromString("100"))
	require.NoError(t, err)

	withdrawals := wdb.New(db)
	ws := withdrawal.New(withdrawals, users, withdrawal.WithTransactor(db))

	before := time.Now()
	w1, err := ws.RequestWithdrawal(ctx, "1234567812345670", u1.ID, decimal.RequireFromString("4.99"))
	require.NoError(t, err)
	assert.True(t, w1.ID > 0)
	assert.Equal(t, "1234567812345670", w1.Number)
	assert.True(t, !w1.ProcessedAt.Before(before))
	assert.Equal(t, u1.ID, w1.User.ID)

	w2, err := ws.RequestWithdrawal(ctx, "4561261212345467", u1.ID, decimal.RequireFromString("5.00"))
	require.NoError(t, err)
	assert.True(t, w2.ID > w1.ID)
	assert.Equal(t, "4561261212345467", w2.Number)
	assert.True(t, !w2.ProcessedAt.Before(before))
	assert.Equal(t, u1.ID, w2.User.ID)

	w3, err := ws.RequestWithdrawal(ctx, "49927398716", u2.ID, decimal.RequireFromString("0.01"))
	require.NoError(t, err)
	assert.True(t, w3.ID > w2.ID)
	assert.Equal(t, "49927398716", w3.Number)
	assert.True(t, !w3.ProcessedAt.Before(before))
	assert.Equal(t, u2.ID, w3.User.ID)

	u1, _ = users.GetByID(ctx, u1.ID)
	assert.Equal(t, "0.01", u1.Balance.Current.String())
	assert.Equal(t, "9.99", u1.Balance.Withdrawn.String())
	u1Items, _ := withdrawals.GetListForUser(ctx, u1.ID)
	assert.Len(t, u1Items, 2)

	u2, _ = users.GetByID(ctx, u2.ID)
	assert.Equal(t, "99.99", u2.Balance.Current.String())
	assert.Equal(t, "0.01", u2.Balance.Withdrawn.String())
	u2Items, _ := withdrawals.GetListForUser(ctx, u2.ID)
	assert.Len(t, u2Items, 1)
}

func TestWithdrawalService_RequestWithdrawal_Duplicate(t *testing.T) {
	ctx := context.TODO()
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)

	u1, _ := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	u2, _ := users.Create(context.TODO(), models.User{Login: "shopper", Password: "secr3t"})

	err := users.AccruePoints(context.TODO(), u1.ID, decimal.RequireFromString("10"))
	require.NoError(t, err)
	err = users.AccruePoints(context.TODO(), u2.ID, decimal.RequireFromString("100"))
	require.NoError(t, err)

	withdrawals := wdb.New(db)
	ws := withdrawal.New(withdrawals, users, withdrawal.WithTransactor(db))

	_, err = ws.RequestWithdrawal(ctx, "1234567812345670", u1.ID, decimal.RequireFromString("4.99"))
	require.NoError(t, err)

	_, err = ws.RequestWithdrawal(ctx, "1234567812345670", u2.ID, decimal.RequireFromString("0.01"))
	require.ErrorIs(t, err, wrepo.ErrWithdrawalAlreadyRegistered)

	u1, _ = users.GetByID(ctx, u1.ID)
	assert.Equal(t, "5.01", u1.Balance.Current.String())
	assert.Equal(t, "4.99", u1.Balance.Withdrawn.String())
	u1Items, _ := withdrawals.GetListForUser(ctx, u1.ID)
	assert.Len(t, u1Items, 1)

	u2, _ = users.GetByID(ctx, u2.ID)
	assert.Equal(t, "100", u2.Balance.Current.String())
	assert.Equal(t, "0", u2.Balance.Withdrawn.String())
	u2Items, _ := withdrawals.GetListForUser(ctx, u2.ID)
	assert.Len(t, u2Items, 0)
}

func TestWithdrawalService_RequestWithdrawal_NegativeSum(t *testing.T) {
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
			ctx := context.TODO()
			_, db, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			users := udb.New(db)

			u, err := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
			require.NoError(t, err)
			err = users.AccruePoints(context.TODO(), u.ID, decimal.RequireFromString("10"))
			require.NoError(t, err)

			withdrawals := wdb.New(db)
			ws := withdrawal.New(withdrawals, users, withdrawal.WithTransactor(db))

			_, err = ws.RequestWithdrawal(ctx, "1234567812345670", u.ID, decimal.RequireFromString(tt.sum))
			require.ErrorIs(t, err, urepo.ErrUserCantWithdrawNegativeSum)

			u, _ = users.GetByID(ctx, u.ID)
			assert.Equal(t, "10", u.Balance.Current.String())
			assert.Equal(t, "0", u.Balance.Withdrawn.String())
			u1Items, _ := withdrawals.GetListForUser(ctx, u.ID)
			assert.Len(t, u1Items, 0)
		})
	}
}

func TestWithdrawalService_RequestWithdrawal_NotEnoughPoints(t *testing.T) {
	tests := []struct {
		name    string
		initial string
	}{
		{
			"have no points at start",
			"0",
		},
		{
			"have very vew points at start",
			"0.01",
		},
		{
			"have some points at start",
			"10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			_, db, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			users := udb.New(db)

			u, err := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
			require.NoError(t, err)
			err = users.AccruePoints(context.TODO(), u.ID, decimal.RequireFromString(tt.initial))
			require.NoError(t, err)

			withdrawals := wdb.New(db)
			ws := withdrawal.New(withdrawals, users, withdrawal.WithTransactor(db))

			_, err = ws.RequestWithdrawal(
				ctx, "1234567812345670", u.ID, decimal.RequireFromString("10.00001"),
			)
			require.ErrorIs(t, err, urepo.ErrUserHasInsufficientAccrual)

			u, _ = users.GetByID(ctx, u.ID)
			assert.Equal(t, tt.initial, u.Balance.Current.String())
			assert.Equal(t, "0", u.Balance.Withdrawn.String())
			u1Items, _ := withdrawals.GetListForUser(ctx, u.ID)
			assert.Len(t, u1Items, 0)
		})
	}
}
