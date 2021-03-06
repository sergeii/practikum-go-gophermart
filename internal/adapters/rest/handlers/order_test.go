package handlers_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sergeii/practikum-go-gophermart/cmd/gophermart/config"
	"github.com/sergeii/practikum-go-gophermart/internal/core/orders"
	"github.com/sergeii/practikum-go-gophermart/internal/testutils"
)

type uploadOrderRespSchema struct {
	Result struct {
		ID         int       `json:"id"`
		Status     string    `json:"status"`
		Number     string    `json:"number"`
		UploadedAt time.Time `json:"uploaded_at"` // nolint: tagliatelle
	} `json:"result"`
}

type uploadOrderErrorSchema struct {
	Error string `json:"error"`
}

type listOrderItemSchema struct {
	Status     string    `json:"status"`
	Number     string    `json:"number"`
	Accrual    float64   `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"` // nolint: tagliatelle
}

func TestHandler_UploadOrder_OK(t *testing.T) {
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")
	before := time.Now()
	var respJSON uploadOrderRespSchema
	resp, _ := testutils.DoTestRequest(
		ts, http.MethodPost,
		"/api/user/orders", strings.NewReader("1234567812345670"),
		testutils.WithUser(u, app),
		testutils.MustBindJSON(&respJSON),
	)
	resp.Body.Close()
	assert.Equal(t, 202, resp.StatusCode)
	assert.Equal(t, "NEW", respJSON.Result.Status)
	assert.Equal(t, "1234567812345670", respJSON.Result.Number)
	assert.True(t, respJSON.Result.UploadedAt.After(before))
	assert.True(t, respJSON.Result.UploadedAt.Before(time.Now()))

	// order is added to the processing queue
	qLen, _ := app.OrderService.ProcessingLength(context.TODO())
	assert.Equal(t, 1, qLen)

	// duplicate request is handled
	resp, respBody := testutils.DoTestRequest(
		ts, http.MethodPost,
		"/api/user/orders", strings.NewReader("1234567812345670"),
		testutils.WithUser(u, app),
	)
	resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "", respBody)

	// duplicate order is NOT added to the processing queue
	qLen, _ = app.OrderService.ProcessingLength(context.TODO())
	assert.Equal(t, 1, qLen)
}

func TestHandler_UploadOrder_Validation(t *testing.T) {
	tests := []struct {
		name       string
		number     string
		want       bool
		wantStatus int
	}{
		{
			"valid luhn number",
			"79927398713",
			true,
			202,
		},
		{
			"invalid luhn number",
			"79927398714",
			false,
			422,
		},
		{
			"not a numeric number",
			"foo",
			false,
			422,
		},
		{
			"numeric number with mixed letters",
			"79927398713foo",
			false,
			422,
		},
		{
			"empty body",
			"",
			false,
			400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, app, cancel := testutils.PrepareTestServer()
			defer cancel()
			u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")
			resp, _ := testutils.DoTestRequest(
				ts, http.MethodPost,
				"/api/user/orders", strings.NewReader(tt.number),
				testutils.WithUser(u, app),
			)
			resp.Body.Close()
			userOrders, _ := app.OrderService.GetUserOrders(context.TODO(), u.ID)
			qLen, _ := app.OrderService.ProcessingLength(context.TODO())
			if tt.want {
				assert.Len(t, userOrders, 1)
				assert.Equal(t, 1, qLen)
			} else {
				assert.Len(t, userOrders, 0)
				assert.Equal(t, 0, qLen)
			}
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestHandler_UploadOrder_ErrorOnDuplicate(t *testing.T) {
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	u1, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")
	u2, _ := app.UserService.RegisterNewUser(context.TODO(), "other", "strong")
	resp, _ := testutils.DoTestRequest(
		ts, http.MethodPost,
		"/api/user/orders", strings.NewReader("1234567812345670"),
		testutils.WithUser(u1, app),
	)
	resp.Body.Close()
	assert.Equal(t, 202, resp.StatusCode)

	var respJSON uploadOrderErrorSchema
	resp, _ = testutils.DoTestRequest(
		ts, http.MethodPost,
		"/api/user/orders", strings.NewReader("1234567812345670"),
		testutils.WithUser(u2, app),
		testutils.MustBindJSON(&respJSON),
	)
	resp.Body.Close()
	assert.Equal(t, 409, resp.StatusCode)
	assert.Equal(t, "order has already been uploaded by another user", respJSON.Error)
}

func TestHandler_UploadOrder_LuhnValidation(t *testing.T) {
	tests := []struct {
		number string
		want   bool
	}{
		{
			"79927398713",
			true,
		},
		{
			"79927398714",
			false,
		},
		{
			"4561261212345467",
			true,
		},
		{
			"49927398716",
			true,
		},
		{
			"foo",
			false,
		},
		{
			"79927398713foo",
			false,
		},
		{
			"0",
			true,
		},
		{
			"01",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			ts, app, cancel := testutils.PrepareTestServer()
			defer cancel()
			u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")
			resp, _ := testutils.DoTestRequest(
				ts, http.MethodPost,
				"/api/user/orders", strings.NewReader(tt.number),
				testutils.WithUser(u, app),
			)
			resp.Body.Close()
			userOrders, _ := app.OrderService.GetUserOrders(context.TODO(), u.ID)
			if tt.want {
				assert.Equal(t, 202, resp.StatusCode)
				assert.Len(t, userOrders, 1)
			} else {
				assert.Equal(t, 422, resp.StatusCode)
				assert.Len(t, userOrders, 0)
			}
		})
	}
}

func TestHandler_UploadOrder_ErrorWhenQueueIsFull(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{
			"positive case",
			2,
			false,
		},
		{
			"queue is full",
			1,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, app, cancel := testutils.PrepareTestServer(func(cfg *config.Config) {
				cfg.AccrualQueueSize = tt.size
			})
			defer cancel()
			u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")

			resp, _ := testutils.DoTestRequest(
				ts, http.MethodPost,
				"/api/user/orders", strings.NewReader("1234567812345670"),
				testutils.WithUser(u, app),
			)
			resp.Body.Close()
			assert.Equal(t, 202, resp.StatusCode)

			resp, _ = testutils.DoTestRequest(
				ts, http.MethodPost,
				"/api/user/orders", strings.NewReader("79927398713"),
				testutils.WithUser(u, app),
			)
			resp.Body.Close()

			if tt.wantErr {
				assert.Equal(t, 503, resp.StatusCode)
			} else {
				assert.Equal(t, 202, resp.StatusCode)
			}
		})
	}
}

func TestHandler_UploadOrder_RequiresAuth(t *testing.T) {
	ts, _, cancel := testutils.PrepareTestServer()
	defer cancel()
	resp, _ := testutils.DoTestRequest(ts, http.MethodPost, "/api/user/orders", strings.NewReader("100500"))
	resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}

func TestHandler_ListUserOrders_OK(t *testing.T) {
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	other, _ := app.UserService.RegisterNewUser(context.TODO(), "other", "secret")
	_, err := app.OrderService.SubmitNewOrder(context.TODO(), "79927398713", other.ID)
	require.NoError(t, err)

	u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")
	app.OrderService.SubmitNewOrder(context.TODO(), "4561261212345467", u.ID) // nolint:errcheck
	app.OrderService.SubmitNewOrder(context.TODO(), "49927398716", u.ID)      // nolint:errcheck
	app.OrderService.UpdateOrderStatus(                                       // nolint:errcheck
		context.TODO(), "49927398716",
		orders.OrderStatusProcessed, decimal.RequireFromString("10.1"),
	)

	jsonItems := make([]listOrderItemSchema, 0)
	resp, _ := testutils.DoTestRequest(
		ts, http.MethodGet, "/api/user/orders", nil,
		testutils.WithUser(u, app),
		testutils.MustBindJSON(&jsonItems),
	)
	resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Len(t, jsonItems, 2)
	assert.Equal(t, "NEW", jsonItems[0].Status)
	assert.Equal(t, "4561261212345467", jsonItems[0].Number)
	assert.Equal(t, 0.0, jsonItems[0].Accrual)
	assert.Equal(t, "PROCESSED", jsonItems[1].Status)
	assert.Equal(t, "49927398716", jsonItems[1].Number)
	assert.Equal(t, 10.1, jsonItems[1].Accrual)
}

func TestHandler_ListUserOrders_NoOrdersForUser(t *testing.T) {
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	other, _ := app.UserService.RegisterNewUser(context.TODO(), "other", "secret")
	_, err := app.OrderService.SubmitNewOrder(context.TODO(), "79927398713", other.ID)
	require.NoError(t, err)

	u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")
	resp, _ := testutils.DoTestRequest(
		ts, http.MethodGet, "/api/user/orders", nil,
		testutils.WithUser(u, app),
	)
	resp.Body.Close()
	assert.Equal(t, 204, resp.StatusCode)
}

func TestHandler_ListUserOrders_RequiresAuth(t *testing.T) {
	ts, _, cancel := testutils.PrepareTestServer()
	defer cancel()
	resp, _ := testutils.DoTestRequest(ts, http.MethodGet, "/api/user/orders", nil)
	resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}
