package db_test

import (
	"context"
	"testing"
	"time"

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
	o, err := repo.Add(context.TODO(), models.NewOrder(u, "1234567812345670"))
	require.NoError(t, err)
	assert.True(t, o.ID > 0)
	assert.Equal(t, "1234567812345670", o.Number)
	assert.True(t, o.UploadedAt.After(before))
	assert.Equal(t, u.ID, o.User.ID)
}

func TestOrdersRepository_Add_ErrorOnDuplicate(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	u, err := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	require.NoError(t, err)

	repo := odb.New(pgpool)
	o, err := repo.Add(context.TODO(), models.NewOrder(u, "1234567812345670"))
	require.NoError(t, err)
	assert.True(t, o.ID > 0)

	o, err = repo.Add(context.TODO(), models.NewOrder(u, "1234567812345670"))
	require.ErrorIs(t, err, orders.ErrOrderAlreadyExists)
	assert.Equal(t, 0, o.ID)
}

func TestOrdersRepository_Add_ErrorOnForeignKeyMissing(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	u := models.User{ID: 9999999, Login: "happycustomer", Password: "str0ng"}

	repo := odb.New(pgpool)
	o, err := repo.Add(context.TODO(), models.NewOrder(u, "1234567812345670"))
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
	newOrder := models.NewOrder(u, "1234567812345670")
	newOrder.Status = "foo"
	o, err := repo.Add(context.TODO(), newOrder)
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
		_, err = repo.Add(context.TODO(), models.NewOrder(u, number))
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
}

func TestRepository_GetListForUser_NoErrorForUnknownUser(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	repo := odb.New(pgpool)
	userOrders, err := repo.GetListForUser(context.TODO(), 9999999)
	require.NoError(t, err)
	assert.Len(t, userOrders, 0)
}
