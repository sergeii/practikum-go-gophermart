package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sergeii/practikum-go-gophermart/internal/core/orders"
	odb "github.com/sergeii/practikum-go-gophermart/internal/core/orders/postgres"
	urepo "github.com/sergeii/practikum-go-gophermart/internal/core/users"
	udb "github.com/sergeii/practikum-go-gophermart/internal/core/users/postgres"
	"github.com/sergeii/practikum-go-gophermart/internal/testutils"
)

func TestOrdersDatabase_Add_OK(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	u, err := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))
	require.NoError(t, err)

	before := time.Now()
	repo := odb.New(db)
	o, err := repo.Add(context.TODO(), orders.New("1234567812345670", u.ID))
	require.NoError(t, err)
	assert.True(t, o.ID > 0)
	assert.Equal(t, "1234567812345670", o.Number)
	assert.True(t, !o.UploadedAt.Before(before))
	assert.Equal(t, u.ID, o.User.ID)
}

func TestOrdersDatabase_Add_ErrorOnDuplicate(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	u, err := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))
	require.NoError(t, err)

	repo := odb.New(db)
	o, err := repo.Add(context.TODO(), orders.New("1234567812345670", u.ID))
	require.NoError(t, err)
	assert.True(t, o.ID > 0)

	o, err = repo.Add(context.TODO(), orders.New("1234567812345670", u.ID))
	assert.ErrorContains(t, err, "duplicate key value violates unique constraint")
	assert.Equal(t, 0, o.ID)
}

func TestOrdersDatabase_Add_ErrorOnForeignKeyMissing(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := odb.New(db)
	o, err := repo.Add(context.TODO(), orders.New("1234567812345670", 9999999))
	require.Error(t, err)
	require.ErrorContains(t, err, `violates foreign key constraint "orders_user_id_fk_users"`)
	assert.Equal(t, 0, o.ID)
}

func TestOrdersDatabase_Add_ErrorOnInvalidStatus(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	u, err := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))
	require.NoError(t, err)

	repo := odb.New(db)
	newOrder := orders.New("1234567812345670", u.ID)
	newOrder.Status = "foo"
	o, err := repo.Add(context.TODO(), newOrder)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid input value for enum order_status")
	assert.Equal(t, 0, o.ID)
}

func TestOrdersDatabase_GetListForUser_OK(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	before := time.Now()

	users := udb.New(db)
	u, err := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))
	require.NoError(t, err)

	repo := odb.New(db)
	for _, number := range []string{"1234567812345670", "4561261212345467", "49927398716"} {
		_, err = repo.Add(context.TODO(), orders.New(number, u.ID))
		require.NoError(t, err)
	}

	userOrders, err := repo.GetListForUser(context.TODO(), u.ID)
	require.NoError(t, err)
	assert.Len(t, userOrders, 3)
	for _, o := range userOrders {
		assert.Equal(t, orders.OrderStatusNew, o.Status)
		assert.Equal(t, u.ID, o.User.ID)
		assert.True(t, o.UploadedAt.After(before))
	}
	assert.Equal(t, "1234567812345670", userOrders[0].Number)
	assert.Equal(t, "4561261212345467", userOrders[1].Number)
	assert.Equal(t, "49927398716", userOrders[2].Number)
}

func TestOrdersDatabase_GetListForUser_NoErrorForUnknownUser(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := odb.New(db)
	userOrders, err := repo.GetListForUser(context.TODO(), 9999999)
	require.NoError(t, err)
	assert.Len(t, userOrders, 0)
}

func TestOrdersDatabase_UpdateStatus_OK(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	u, err := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))
	require.NoError(t, err)

	repo := odb.New(db)
	o, _ := repo.Add(context.TODO(), orders.New("1234567812345670", u.ID))
	other, _ := repo.Add(context.TODO(), orders.New("4561261212345467", u.ID))

	o.Status = orders.OrderStatusProcessed
	o.Accrual = decimal.RequireFromString("100.5")
	err = repo.Update(context.TODO(), o.ID, o)
	require.NoError(t, err)

	o2, _ := repo.GetByNumber(context.TODO(), "1234567812345670")
	assert.Equal(t, orders.OrderStatusProcessed, o2.Status)
	assert.Equal(t, "100.5", o2.Accrual.String())
	assert.Equal(t, u.ID, o2.User.ID)                 // does not change
	assert.Equal(t, o.ID, o2.ID)                      // does not change
	assert.True(t, o2.UploadedAt.Equal(o.UploadedAt)) // does not change

	// other orders are not affected
	other2, _ := repo.GetByNumber(context.TODO(), "4561261212345467")
	assert.Equal(t, other.ID, other2.ID)
	assert.Equal(t, orders.OrderStatusNew, other2.Status)
	assert.Equal(t, decimal.NewFromInt(0), other2.Accrual)
}

func TestOrdersDatabase_Update_Errors(t *testing.T) {
	tests := []struct {
		name    string
		status  orders.OrderStatus
		accrual decimal.Decimal
		wantErr bool
	}{
		{
			"positive case",
			orders.OrderStatusProcessed,
			decimal.NewFromInt(100),
			false,
		},
		{
			"invalid order status",
			orders.OrderStatus("foo"),
			decimal.NewFromInt(100),
			true,
		},
		{
			"negative accrual value",
			orders.OrderStatusProcessed,
			decimal.NewFromInt(-100),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, db, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			users := udb.New(db)
			u, err := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))
			require.NoError(t, err)

			repo := odb.New(db)
			o, err := repo.Add(context.TODO(), orders.New("1234567812345670", u.ID))
			require.NoError(t, err)

			o.Status = tt.status
			o.Accrual = tt.accrual
			err = repo.Update(context.TODO(), o.ID, o)
			o2, _ := repo.GetByNumber(context.TODO(), "1234567812345670")
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, orders.OrderStatusNew, o2.Status)
				assert.Equal(t, decimal.NewFromInt(0), o2.Accrual)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.status, o2.Status)
				assert.True(t, tt.accrual.Equal(o2.Accrual))
			}
		})
	}
}
