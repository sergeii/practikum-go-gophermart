package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sergeii/practikum-go-gophermart/internal/core/orders"
	odb "github.com/sergeii/practikum-go-gophermart/internal/core/orders/db"
	udb "github.com/sergeii/practikum-go-gophermart/internal/core/users/db"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/pkg/testutils"
)

func TestOrdersRepository_Add_OK(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	u, err := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	require.NoError(t, err)

	before := time.Now()
	repo := odb.New(pgpool)
	o, err := repo.Add(context.TODO(), models.NewCandidateOrder("1234567812345670", u.ID), orders.AddNoop)
	require.NoError(t, err)
	assert.True(t, o.ID > 0)
	assert.Equal(t, "1234567812345670", o.Number)
	assert.True(t, !o.UploadedAt.Before(before))
	assert.Equal(t, u.ID, o.User.ID)
}

func TestOrdersRepository_Add_ErrorOnDuplicate(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	u, err := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	require.NoError(t, err)

	repo := odb.New(pgpool)
	o, err := repo.Add(context.TODO(), models.NewCandidateOrder("1234567812345670", u.ID), orders.AddNoop)
	require.NoError(t, err)
	assert.True(t, o.ID > 0)

	o, err = repo.Add(context.TODO(), models.NewCandidateOrder("1234567812345670", u.ID), orders.AddNoop)
	require.ErrorIs(t, err, orders.ErrOrderAlreadyExists)
	assert.Equal(t, 0, o.ID)
}

func TestOrdersRepository_Add_ErrorOnForeignKeyMissing(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()
	repo := odb.New(pgpool)
	o, err := repo.Add(context.TODO(), models.NewCandidateOrder("1234567812345670", 9999999), orders.AddNoop)
	require.Error(t, err)
	require.ErrorContains(t, err, `violates foreign key constraint "orders_user_id_fk_users"`)
	assert.Equal(t, 0, o.ID)
}

func TestOrdersRepository_Add_ErrorOnInvalidStatus(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	u, err := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	require.NoError(t, err)

	repo := odb.New(pgpool)
	newOrder := models.NewCandidateOrder("1234567812345670", u.ID)
	newOrder.Status = "foo"
	o, err := repo.Add(context.TODO(), newOrder, orders.AddNoop)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid input value for enum order_status")
	assert.Equal(t, 0, o.ID)
}

func TestRepository_GetListForUser_OK(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	before := time.Now()

	users := udb.New(pgpool)
	u, err := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	require.NoError(t, err)

	repo := odb.New(pgpool)
	for _, number := range []string{"1234567812345670", "4561261212345467", "49927398716"} {
		_, err = repo.Add(context.TODO(), models.NewCandidateOrder(number, u.ID), orders.AddNoop)
		require.NoError(t, err)
	}

	userOrders, err := repo.GetListForUser(context.TODO(), u.ID)
	require.NoError(t, err)
	assert.Len(t, userOrders, 3)
	for _, o := range userOrders {
		assert.Equal(t, models.OrderStatusNew, o.Status)
		assert.Equal(t, u.ID, o.User.ID)
		assert.True(t, o.UploadedAt.After(before))
	}
	assert.Equal(t, "1234567812345670", userOrders[0].Number)
	assert.Equal(t, "4561261212345467", userOrders[1].Number)
	assert.Equal(t, "49927398716", userOrders[2].Number)
}

func TestRepository_GetListForUser_NoErrorForUnknownUser(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := odb.New(pgpool)
	userOrders, err := repo.GetListForUser(context.TODO(), 9999999)
	require.NoError(t, err)
	assert.Len(t, userOrders, 0)
}

func TestRepository_UpdateStatus_OK(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	u, err := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	require.NoError(t, err)

	repo := odb.New(pgpool)
	o, _ := repo.Add(context.TODO(), models.NewCandidateOrder("1234567812345670", u.ID), orders.AddNoop)
	other, _ := repo.Add(context.TODO(), models.NewCandidateOrder("4561261212345467", u.ID), orders.AddNoop)

	err = repo.UpdateStatus(context.TODO(), o.ID, models.OrderStatusProcessed, decimal.RequireFromString("100.5"))
	require.NoError(t, err)

	o2, _ := repo.GetByNumber(context.TODO(), "1234567812345670")
	assert.Equal(t, models.OrderStatusProcessed, o2.Status)
	assert.Equal(t, "100.5", o2.Accrual.String())
	assert.Equal(t, u.ID, o2.User.ID)                 // does not change
	assert.Equal(t, o.ID, o2.ID)                      // does not change
	assert.True(t, o2.UploadedAt.Equal(o.UploadedAt)) // does not change

	// other orders are not affected
	other2, _ := repo.GetByNumber(context.TODO(), "4561261212345467")
	assert.Equal(t, other.ID, other2.ID)
	assert.Equal(t, models.OrderStatusNew, other2.Status)
	assert.Equal(t, decimal.NewFromInt(0), other2.Accrual)
}

func TestRepository_UpdateStatus_Errors(t *testing.T) {
	tests := []struct {
		name    string
		status  models.OrderStatus
		accrual decimal.Decimal
		wantErr bool
	}{
		{
			"positive case",
			models.OrderStatusProcessed,
			decimal.NewFromInt(100),
			false,
		},
		{
			"invalid order status",
			models.OrderStatus("foo"),
			decimal.NewFromInt(100),
			true,
		},
		{
			"negative accrual value",
			models.OrderStatusProcessed,
			decimal.NewFromInt(-100),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgpool, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			users := udb.New(pgpool)
			u, err := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
			require.NoError(t, err)

			repo := odb.New(pgpool)
			o, err := repo.Add(context.TODO(), models.NewCandidateOrder("1234567812345670", u.ID), orders.AddNoop)
			require.NoError(t, err)

			err = repo.UpdateStatus(context.TODO(), o.ID, tt.status, tt.accrual)
			o2, _ := repo.GetByNumber(context.TODO(), "1234567812345670")
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, models.OrderStatusNew, o2.Status)
				assert.Equal(t, decimal.NewFromInt(0), o2.Accrual)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.status, o2.Status)
				assert.True(t, tt.accrual.Equal(o2.Accrual))
			}
		})
	}
}
