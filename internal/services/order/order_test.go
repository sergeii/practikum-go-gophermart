package order_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orepo "github.com/sergeii/practikum-go-gophermart/internal/core/orders"
	odb "github.com/sergeii/practikum-go-gophermart/internal/core/orders/db"
	udb "github.com/sergeii/practikum-go-gophermart/internal/core/users/db"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/pkg/testutils"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
)

func TestService_UploadOrder_OK(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	u, err := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	require.NoError(t, err)

	orders := odb.New(pgpool)
	svc := order.New(orders)

	before := time.Now()
	o, err := svc.UploadOrder(context.TODO(), "1234567812345670", u.ID, orepo.AddNoop)
	require.NoError(t, err)
	assert.True(t, o.ID > 0)
	assert.Equal(t, "1234567812345670", o.Number)
	assert.True(t, o.UploadedAt.After(before))
	assert.Equal(t, u.ID, o.User.ID)
}

func TestService_UploadOrder_Duplicate(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	user1, _ := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	user2, _ := users.Create(context.TODO(), models.User{Login: "othercustomer", Password: "secr3t"})

	orders := odb.New(pgpool)
	svc := order.New(orders)

	o, err := svc.UploadOrder(context.TODO(), "1234567812345670", user1.ID, orepo.AddNoop)
	require.NoError(t, err)
	assert.True(t, o.ID > 0)

	double, err := svc.UploadOrder(context.TODO(), "1234567812345670", user1.ID, orepo.AddNoop)
	assert.ErrorIs(t, err, order.ErrOrderAlreadyUploaded)
	assert.Equal(t, 0, double.ID)

	double, err = svc.UploadOrder(context.TODO(), "1234567812345670", user2.ID, orepo.AddNoop)
	assert.ErrorIs(t, err, order.ErrOrderUploadedByAnotherUser)
	assert.Equal(t, 0, double.ID)
}

func TestService_UpdateOrder_OK(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	u, _ := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})

	orders := odb.New(pgpool)
	svc := order.New(orders)

	o, err := svc.UploadOrder(context.TODO(), "1234567812345670", u.ID, orepo.AddNoop)
	require.NoError(t, err)
	assert.Equal(t, models.OrderStatusNew, o.Status)
	assert.Equal(t, 0.0, o.Accrual)

	err = svc.UpdateOrder(context.TODO(), "1234567812345670", models.OrderStatusProcessed, 100.5)
	require.NoError(t, err)

	upd, _ := orders.GetByNumber(context.TODO(), "1234567812345670")
	assert.Equal(t, models.OrderStatusProcessed, upd.Status)
	assert.Equal(t, 100.5, upd.Accrual)
}

func TestService_UpdateOrder_NotFound(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	orders := odb.New(pgpool)
	svc := order.New(orders)
	err := svc.UpdateOrder(context.TODO(), "1234567812345670", models.OrderStatusProcessed, 100.5)
	assert.ErrorIs(t, err, orepo.ErrOrderNotFound)
}

func TestService_UpdateOrder_ConstraintErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  models.OrderStatus
		accrual float64
		wantErr bool
	}{
		{
			"positive case",
			models.OrderStatusProcessed,
			100,
			false,
		},
		{
			"invalid order status",
			models.OrderStatus("foo"),
			100,
			true,
		},
		{
			"negative accrual value",
			models.OrderStatusProcessed,
			-100,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgpool, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			users := udb.New(pgpool)
			u, _ := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})

			orders := odb.New(pgpool)
			svc := order.New(orders)

			o, err := svc.UploadOrder(context.TODO(), "1234567812345670", u.ID, orepo.AddNoop)
			require.NoError(t, err)

			err = svc.UpdateOrder(context.TODO(), "1234567812345670", tt.status, tt.accrual)
			upd, _ := orders.GetByNumber(context.TODO(), "1234567812345670")
			require.Equal(t, o.ID, upd.ID)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, models.OrderStatusNew, upd.Status)
				assert.Equal(t, 0.0, upd.Accrual)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.status, upd.Status)
				assert.Equal(t, tt.accrual, upd.Accrual)
			}
		})
	}
}
