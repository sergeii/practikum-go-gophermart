package order_test

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
	odb "github.com/sergeii/practikum-go-gophermart/internal/core/orders/postgres"
	urepo "github.com/sergeii/practikum-go-gophermart/internal/core/users"
	udb "github.com/sergeii/practikum-go-gophermart/internal/core/users/postgres"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/accrual"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/queue"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/queue/memory"
	"github.com/sergeii/practikum-go-gophermart/internal/ports/transactor"
	"github.com/sergeii/practikum-go-gophermart/internal/services/order"
	"github.com/sergeii/practikum-go-gophermart/internal/testutils"
	"github.com/sergeii/practikum-go-gophermart/pkg/encode"
)

func newService(
	orders orepo.Repository,
	users urepo.Repository,
	trans transactor.Transactor,
	queueSize int,
	accrualURL string,
) order.Service {
	q, err := memory.New(queueSize)
	if err != nil {
		panic(err)
	}
	if accrualURL == "" {
		accrualURL = "http://localhost:8081"
	}
	acc, err := accrual.New(accrualURL)
	if err != nil {
		panic(err)
	}
	return order.New(orders, users, trans, q, acc)
}

func TestOrderService_SubmitNewOrder_OK(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	u, err := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))
	require.NoError(t, err)

	orders := odb.New(db)
	os := newService(orders, users, db, 10, "")

	before := time.Now()
	o1, err := os.SubmitNewOrder(context.TODO(), "1234567812345670", u.ID)
	require.NoError(t, err)
	assert.True(t, o1.ID > 0)
	assert.Equal(t, "1234567812345670", o1.Number)
	assert.True(t, !o1.UploadedAt.Before(before))
	assert.Equal(t, u.ID, o1.User.ID)
	qLen, _ := os.ProcessingLength(context.TODO())
	assert.Equal(t, 1, qLen)

	o2, err := os.SubmitNewOrder(context.TODO(), "4561261212345467", u.ID)
	require.NoError(t, err)
	assert.True(t, o2.ID > o1.ID)
	assert.Equal(t, "4561261212345467", o2.Number)
	assert.True(t, o2.UploadedAt.After(before))
	assert.Equal(t, u.ID, o2.User.ID)
	qLen, _ = os.ProcessingLength(context.TODO())
	assert.Equal(t, 2, qLen)
}

func TestOrderService_SubmitNewOrder_Duplicate(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	user1, _ := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))
	user2, _ := users.Create(context.TODO(), urepo.New("othercustomer", "secr3t"))

	orders := odb.New(db)
	os := newService(orders, users, db, 10, "")

	_, err := os.SubmitNewOrder(context.TODO(), "1234567812345670", user1.ID)
	require.NoError(t, err)
	qLen, _ := os.ProcessingLength(context.TODO())
	assert.Equal(t, 1, qLen)

	_, err = os.SubmitNewOrder(context.TODO(), "1234567812345670", user1.ID)
	assert.ErrorIs(t, err, order.ErrOrderAlreadyUploaded)

	_, err = os.SubmitNewOrder(context.TODO(), "1234567812345670", user2.ID)
	assert.ErrorIs(t, err, order.ErrOrderUploadedByAnotherUser)

	qLen, _ = os.ProcessingLength(context.TODO())
	assert.Equal(t, 1, qLen)
}

func TestOrderService_UpdateOrderStatus_OK(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	u, _ := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))

	orders := odb.New(db)
	svc := newService(orders, users, db, 10, "")

	o, err := svc.SubmitNewOrder(context.TODO(), "1234567812345670", u.ID)
	require.NoError(t, err)
	assert.Equal(t, orepo.OrderStatusNew, o.Status)
	assert.True(t, decimal.Zero.Equal(o.Accrual))

	err = svc.UpdateOrderStatus(
		context.TODO(), "1234567812345670",
		orepo.OrderStatusProcessed, decimal.RequireFromString("100.5"),
	)
	require.NoError(t, err)

	upd, _ := orders.GetByNumber(context.TODO(), "1234567812345670")
	assert.Equal(t, orepo.OrderStatusProcessed, upd.Status)
	assert.Equal(t, "100.5", upd.Accrual.String())

	u2, _ := users.GetByID(context.TODO(), u.ID)
	assert.Equal(t, "0", u2.Balance.Current.String())
	assert.Equal(t, "0", u2.Balance.Withdrawn.String())
}

func TestOrderService_UpdateOrderStatus_NotFound(t *testing.T) {
	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	orders := odb.New(db)
	svc := newService(orders, users, db, 10, "")
	err := svc.UpdateOrderStatus(
		context.TODO(), "1234567812345670",
		orepo.OrderStatusProcessed, decimal.RequireFromString("100.5"),
	)
	assert.ErrorIs(t, err, orepo.ErrOrderNotFound)
}

func TestOrderService_UpdateOrderStatus_ConstraintErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  orepo.OrderStatus
		accrual decimal.Decimal
		wantErr bool
	}{
		{
			"positive case",
			orepo.OrderStatusProcessed,
			decimal.NewFromInt(100),
			false,
		},
		{
			"invalid order status",
			orepo.OrderStatus("foo"),
			decimal.NewFromInt(100),
			true,
		},
		{
			"negative accrual value",
			orepo.OrderStatusProcessed,
			decimal.NewFromInt(-100),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, db, cancel := testutils.PrepareTestDatabase()
			defer cancel()

			users := udb.New(db)
			u, _ := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))

			orders := odb.New(db)
			svc := newService(orders, users, db, 10, "")

			o, err := svc.SubmitNewOrder(context.TODO(), "1234567812345670", u.ID)
			require.NoError(t, err)

			err = svc.UpdateOrderStatus(context.TODO(), "1234567812345670", tt.status, tt.accrual)
			upd, _ := orders.GetByNumber(context.TODO(), "1234567812345670")
			require.Equal(t, o.ID, upd.ID)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, orepo.OrderStatusNew, upd.Status)
				assert.True(t, upd.Accrual.IsZero())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.status, upd.Status)
				assert.Equal(t, tt.accrual.String(), upd.Accrual.String())
			}
		})
	}
}

func TestOrderService_ProcessNextOrder_Loop(t *testing.T) {
	type resp struct {
		code int
		body []byte
	}
	responses := map[string]resp{
		"1234567812345670": {
			code: 200,
			body: encode.MustJSONMarshal(accrual.OrderStatus{
				Number: "1234567812345670", Status: "PROCESSED", Accrual: decimal.RequireFromString("100.5"),
			}),
		},
		"4561261212345467": {
			code: 200,
			body: encode.MustJSONMarshal(
				accrual.OrderStatus{
					Number: "4561261212345467", Status: "INVALID", Accrual: decimal.NewFromInt(10),
				},
			),
		},
		"79927398713": {
			code: 200,
			body: encode.MustJSONMarshal(
				accrual.OrderStatus{Number: "79927398713", Status: "PROCESSED", Accrual: decimal.NewFromInt(47)},
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

	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	user1, _ := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))
	user2, _ := users.Create(context.TODO(), urepo.New("othercustomer", "secr3t"))

	orders := odb.New(db)
	os := newService(orders, users, db, 4, ts.URL)

	_, err := os.SubmitNewOrder(context.TODO(), "1234567812345670", user1.ID)
	assert.NoError(t, err)
	_, err = os.SubmitNewOrder(context.TODO(), "4561261212345467", user2.ID)
	assert.NoError(t, err)
	_, err = os.SubmitNewOrder(context.TODO(), "79927398713", user2.ID)
	assert.NoError(t, err)
	_, err = os.SubmitNewOrder(context.TODO(), "49927398716", user1.ID)
	assert.NoError(t, err)
	_, err = os.SubmitNewOrder(context.TODO(), "100000000008", user2.ID)
	assert.ErrorIs(t, err, queue.ErrQueueIsFull)

	qLen, _ := os.ProcessingLength(context.TODO())
	assert.Equal(t, 4, qLen)

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				wait := os.ProcessNextOrder(context.TODO())
				<-wait
			}
		}
	}()
	<-time.After(time.Millisecond * 300)
	close(done)

	o1, _ := orders.GetByNumber(context.TODO(), "1234567812345670")
	assert.Equal(t, orepo.OrderStatusProcessed, o1.Status)
	assert.Equal(t, "100.5", o1.Accrual.String())
	assert.Equal(t, user1.ID, o1.User.ID)

	o2, _ := orders.GetByNumber(context.TODO(), "4561261212345467")
	assert.Equal(t, orepo.OrderStatusInvalid, o2.Status)
	assert.Equal(t, decimal.Zero.String(), o2.Accrual.String())
	assert.Equal(t, user2.ID, o2.User.ID)

	o3, _ := orders.GetByNumber(context.TODO(), "79927398713")
	assert.Equal(t, orepo.OrderStatusProcessed, o3.Status)
	assert.Equal(t, "47", o3.Accrual.String())
	assert.Equal(t, user2.ID, o3.User.ID)

	o4, _ := orders.GetByNumber(context.TODO(), "49927398716")
	assert.Equal(t, orepo.OrderStatusInvalid, o4.Status)
	assert.Equal(t, decimal.Zero.String(), o4.Accrual.String())
	assert.Equal(t, user1.ID, o4.User.ID)

	_, err = orders.GetByNumber(context.TODO(), "100000000008")
	assert.ErrorIs(t, err, orepo.ErrOrderNotFound)

	u1, _ := users.GetByID(context.TODO(), user1.ID)
	assert.Equal(t, "100.5", u1.Balance.Current.String())
	assert.Equal(t, "0", u1.Balance.Withdrawn.String())

	u2, _ := users.GetByID(context.TODO(), user2.ID)
	assert.Equal(t, "47", u2.Balance.Current.String())
	assert.Equal(t, "0", u2.Balance.Withdrawn.String())

	qLen, _ = os.ProcessingLength(context.TODO())
	assert.Equal(t, 0, qLen)
}

func TestOrderService_ProcessNextOrder_Retry(t *testing.T) {
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

	_, db, cancel := testutils.PrepareTestDatabase()
	defer cancel()

	users := udb.New(db)
	u, _ := users.Create(context.TODO(), urepo.New("happycustomer", "str0ng"))

	orders := odb.New(db)
	os := newService(orders, users, db, 3, ts.URL)

	_, err := os.SubmitNewOrder(context.TODO(), "79927398713", u.ID)
	assert.NoError(t, err)
	qLen, _ := os.ProcessingLength(context.TODO())
	assert.Equal(t, 1, qLen)

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				wait := os.ProcessNextOrder(context.TODO())
				<-wait
			}
		}
	}()
	<-done
	<-time.After(time.Millisecond * 50)

	o, _ := orders.GetByNumber(context.TODO(), "79927398713")
	assert.Equal(t, orepo.OrderStatusProcessed, o.Status)
	assert.Equal(t, "100.5", o.Accrual.String())
	assert.Equal(t, u.ID, o.User.ID)

	u, _ = users.GetByID(context.TODO(), u.ID)
	assert.Equal(t, "100.5", u.Balance.Current.String())
	assert.Equal(t, "0", u.Balance.Withdrawn.String())

	qLen, _ = os.ProcessingLength(context.TODO())
	assert.Equal(t, 0, qLen)
}
