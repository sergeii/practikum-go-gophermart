package processing_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orepo "github.com/sergeii/practikum-go-gophermart/internal/core/orders"
	odb "github.com/sergeii/practikum-go-gophermart/internal/core/orders/db"
	"github.com/sergeii/practikum-go-gophermart/internal/core/queue"
	udb "github.com/sergeii/practikum-go-gophermart/internal/core/users/db"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/pkg/testutils"
	"github.com/sergeii/practikum-go-gophermart/internal/services/accrual"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
	"github.com/sergeii/practikum-go-gophermart/internal/services/processing"
)

func TestService_SubmitNewOrder_OK(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	u, err := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	require.NoError(t, err)

	orders := odb.New(pgpool)
	os := order.New(orders)
	ps := processing.New(processing.WithOrderService(os), processing.WithInMemoryQueue(10))

	before := time.Now()
	o1, err := ps.SubmitNewOrder(context.TODO(), "1234567812345670", u.ID)
	require.NoError(t, err)
	assert.True(t, o1.ID > 0)
	assert.Equal(t, "1234567812345670", o1.Number)
	assert.True(t, !o1.UploadedAt.Before(before))
	assert.Equal(t, u.ID, o1.User.ID)
	qLen, _ := ps.QueueLength(context.TODO())
	assert.Equal(t, 1, qLen)

	o2, err := ps.SubmitNewOrder(context.TODO(), "4561261212345467", u.ID)
	require.NoError(t, err)
	assert.True(t, o2.ID > o1.ID)
	assert.Equal(t, "4561261212345467", o2.Number)
	assert.True(t, o2.UploadedAt.After(before))
	assert.Equal(t, u.ID, o2.User.ID)
	qLen, _ = ps.QueueLength(context.TODO())
	assert.Equal(t, 2, qLen)
}

func TestService_SubmitNewOrder_Duplicate(t *testing.T) {
	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	user1, _ := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	user2, _ := users.Create(context.TODO(), models.User{Login: "othercustomer", Password: "secr3t"})

	orders := odb.New(pgpool)
	os := order.New(orders)
	ps := processing.New(processing.WithOrderService(os), processing.WithInMemoryQueue(10))

	_, err := ps.SubmitNewOrder(context.TODO(), "1234567812345670", user1.ID)
	require.NoError(t, err)
	qLen, _ := ps.QueueLength(context.TODO())
	assert.Equal(t, 1, qLen)

	_, err = ps.SubmitNewOrder(context.TODO(), "1234567812345670", user1.ID)
	assert.ErrorIs(t, err, order.ErrOrderAlreadyUploaded)

	_, err = ps.SubmitNewOrder(context.TODO(), "1234567812345670", user2.ID)
	assert.ErrorIs(t, err, order.ErrOrderUploadedByAnotherUser)

	qLen, _ = ps.QueueLength(context.TODO())
	assert.Equal(t, 1, qLen)
}

func TestService_ProcessNextOrder_Loop(t *testing.T) {
	type resp struct {
		code int
		body []byte
	}
	responses := map[string]resp{
		"1234567812345670": {
			code: 200,
			body: testutils.MustJSONMarshal(accrual.OrderStatus{
				Number: "79927398713", Status: "PROCESSED", Accrual: decimal.RequireFromString("100.5"),
			}),
		},
		"4561261212345467": {
			code: 200,
			body: testutils.MustJSONMarshal(
				accrual.OrderStatus{Number: "79927398713", Status: "INVALID", Accrual: decimal.NewFromInt(10)},
			),
		},
		"49927398716": {
			code: 204,
			body: nil,
		},
	}
	r := gin.New()
	r.GET("/api/orders/:order", func(c *gin.Context) {
		r := responses[c.Param("order")]
		c.String(r.code, string(r.body))
	})
	ts := httptest.NewServer(r)
	accrualService, err := accrual.New(ts.URL)
	require.NoError(t, err)

	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	user1, _ := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})
	user2, _ := users.Create(context.TODO(), models.User{Login: "othercustomer", Password: "secr3t"})

	orders := odb.New(pgpool)
	os := order.New(orders)
	ps := processing.New(
		processing.WithOrderService(os),
		processing.WithInMemoryQueue(3),
		processing.WithAccrualService(accrualService),
	)

	_, err = ps.SubmitNewOrder(context.TODO(), "1234567812345670", user1.ID)
	assert.NoError(t, err)
	_, err = ps.SubmitNewOrder(context.TODO(), "4561261212345467", user2.ID)
	assert.NoError(t, err)
	_, err = ps.SubmitNewOrder(context.TODO(), "49927398716", user1.ID)
	assert.NoError(t, err)
	_, err = ps.SubmitNewOrder(context.TODO(), "100000000008", user2.ID)
	assert.ErrorIs(t, err, queue.ErrQueueIsFull)

	qLen, _ := ps.QueueLength(context.TODO())
	assert.Equal(t, 3, qLen)

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				wait := ps.ProcessNextOrder(context.TODO())
				<-wait
			}
		}
	}()
	<-time.After(time.Millisecond * 300)
	close(done)

	o1, _ := orders.GetByNumber(context.TODO(), "1234567812345670")
	assert.Equal(t, models.OrderStatusProcessed, o1.Status)
	assert.Equal(t, "100.5", o1.Accrual.String())
	assert.Equal(t, user1.ID, o1.User.ID)

	o2, _ := orders.GetByNumber(context.TODO(), "4561261212345467")
	assert.Equal(t, models.OrderStatusInvalid, o2.Status)
	assert.Equal(t, decimal.Zero.String(), o2.Accrual.String())
	assert.Equal(t, user2.ID, o2.User.ID)

	o3, _ := orders.GetByNumber(context.TODO(), "49927398716")
	assert.Equal(t, models.OrderStatusInvalid, o3.Status)
	assert.Equal(t, decimal.Zero.String(), o3.Accrual.String())
	assert.Equal(t, user1.ID, o3.User.ID)

	_, err = orders.GetByNumber(context.TODO(), "100000000008")
	assert.ErrorIs(t, err, orepo.ErrOrderNotFound)

	qLen, _ = ps.QueueLength(context.TODO())
	assert.Equal(t, 0, qLen)
}

func TestService_ProcessNextOrder_Retry(t *testing.T) {
	done := make(chan struct{})
	retry := 0
	r := gin.New()
	r.GET("/api/orders/:order", func(c *gin.Context) {
		retry++
		if retry == 3 {
			c.JSON(200, accrual.OrderStatus{
				Number: "79927398713", Status: "PROCESSED",
				Accrual: decimal.RequireFromString("100.5"),
			})
			close(done)
		} else {
			c.Header("Retry-After", "1")
			c.Status(429)
		}
	})
	ts := httptest.NewServer(r)
	accrualService, err := accrual.New(ts.URL)
	require.NoError(t, err)

	pgpool, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(pgpool)
	u, _ := users.Create(context.TODO(), models.User{Login: "happycustomer", Password: "str0ng"})

	orders := odb.New(pgpool)
	os := order.New(orders)
	ps := processing.New(
		processing.WithOrderService(os),
		processing.WithInMemoryQueue(10),
		processing.WithAccrualService(accrualService),
	)

	_, err = ps.SubmitNewOrder(context.TODO(), "79927398713", u.ID)
	assert.NoError(t, err)
	qLen, _ := ps.QueueLength(context.TODO())
	assert.Equal(t, 1, qLen)

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				wait := ps.ProcessNextOrder(context.TODO())
				<-wait
			}
		}
	}()
	<-done
	<-time.After(time.Millisecond * 50)

	o, _ := orders.GetByNumber(context.TODO(), "79927398713")
	assert.Equal(t, models.OrderStatusProcessed, o.Status)
	assert.Equal(t, "100.5", o.Accrual.String())
	assert.Equal(t, u.ID, o.User.ID)

	qLen, _ = ps.QueueLength(context.TODO())
	assert.Equal(t, 0, qLen)
}
