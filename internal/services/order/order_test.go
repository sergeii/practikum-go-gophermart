package order_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	o, err := svc.UploadOrder(context.TODO(), u, "1234567812345670")
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

	o, err := svc.UploadOrder(context.TODO(), user1, "1234567812345670")
	require.NoError(t, err)
	assert.True(t, o.ID > 0)

	double, err := svc.UploadOrder(context.TODO(), user1, "1234567812345670")
	assert.ErrorIs(t, err, order.ErrOrderAlreadyUploaded)
	assert.Equal(t, 0, double.ID)

	double, err = svc.UploadOrder(context.TODO(), user2, "1234567812345670")
	assert.ErrorIs(t, err, order.ErrOrderUploadedByAnotherUser)
	assert.Equal(t, 0, double.ID)
}
