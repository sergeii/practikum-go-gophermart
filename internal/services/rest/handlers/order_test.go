package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sergeii/practikum-go-gophermart/internal/pkg/testutils"
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
	UploadedAt time.Time `json:"uploaded_at"` // nolint: tagliatelle
}

func TestHandler_UploadOrder_OK(t *testing.T) {
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")
	before := time.Now()
	resp, respBody := testutils.DoTestRequest(
		t, ts, http.MethodPost,
		"/api/user/orders", strings.NewReader("1234567812345670"),
		testutils.RequestWithUser(u, app),
	)
	defer resp.Body.Close()
	assert.Equal(t, 202, resp.StatusCode)
	var respJSON uploadOrderRespSchema
	json.Unmarshal([]byte(respBody), &respJSON) // nolint:errcheck
	assert.Equal(t, "NEW", respJSON.Result.Status)
	assert.Equal(t, "1234567812345670", respJSON.Result.Number)
	assert.True(t, respJSON.Result.UploadedAt.After(before))
	assert.True(t, respJSON.Result.UploadedAt.Before(time.Now()))

	resp, respBody = testutils.DoTestRequest(
		t, ts, http.MethodPost,
		"/api/user/orders", strings.NewReader("1234567812345670"),
		testutils.RequestWithUser(u, app),
	)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "", respBody)
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
				t, ts, http.MethodPost,
				"/api/user/orders", strings.NewReader(tt.number),
				testutils.RequestWithUser(u, app),
			)
			defer resp.Body.Close()
			userOrders, _ := app.OrderService.GetUserOrders(context.TODO(), u.ID)
			if tt.want {
				assert.Len(t, userOrders, 1)
			} else {
				assert.Len(t, userOrders, 0)
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
		t, ts, http.MethodPost,
		"/api/user/orders", strings.NewReader("1234567812345670"),
		testutils.RequestWithUser(u1, app),
	)
	defer resp.Body.Close()
	assert.Equal(t, 202, resp.StatusCode)

	resp, respBody := testutils.DoTestRequest(
		t, ts, http.MethodPost,
		"/api/user/orders", strings.NewReader("1234567812345670"),
		testutils.RequestWithUser(u2, app),
	)
	defer resp.Body.Close()
	assert.Equal(t, 409, resp.StatusCode)
	var respJSON uploadOrderErrorSchema
	json.Unmarshal([]byte(respBody), &respJSON) // nolint:errcheck
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
				t, ts, http.MethodPost,
				"/api/user/orders", strings.NewReader(tt.number),
				testutils.RequestWithUser(u, app),
			)
			defer resp.Body.Close()
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

func TestHandler_UploadOrder_RequiresAuth(t *testing.T) {
	ts, _, cancel := testutils.PrepareTestServer()
	defer cancel()
	resp, _ := testutils.DoTestRequest(
		t, ts, http.MethodPost,
		"/api/user/orders", strings.NewReader("100500"),
	)
	defer resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}

func TestHandler_ListUserOrders_OK(t *testing.T) {
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	other, _ := app.UserService.RegisterNewUser(context.TODO(), "other", "secret")
	_, err := app.OrderService.UploadOrder(context.TODO(), other, "79927398713")
	require.NoError(t, err)

	u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")
	app.OrderService.UploadOrder(context.TODO(), u, "4561261212345467") // nolint:errcheck
	app.OrderService.UploadOrder(context.TODO(), u, "49927398716")      // nolint:errcheck
	resp, body := testutils.DoTestRequest(
		t, ts, http.MethodGet, "/api/user/orders", nil,
		testutils.RequestWithUser(u, app),
	)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	jsonItems := make([]listOrderItemSchema, 0)
	json.Unmarshal([]byte(body), &jsonItems) // nolint:errcheck
	assert.Len(t, jsonItems, 2)
	assert.Equal(t, "NEW", jsonItems[0].Status)
	assert.Equal(t, "4561261212345467", jsonItems[0].Number)
	assert.Equal(t, "NEW", jsonItems[1].Status)
	assert.Equal(t, "49927398716", jsonItems[1].Number)
}

func TestHandler_ListUserOrders_NoOrdersForUser(t *testing.T) {
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	other, _ := app.UserService.RegisterNewUser(context.TODO(), "other", "secret")
	_, err := app.OrderService.UploadOrder(context.TODO(), other, "79927398713")
	require.NoError(t, err)

	u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")
	resp, _ := testutils.DoTestRequest(
		t, ts, http.MethodGet, "/api/user/orders", nil,
		testutils.RequestWithUser(u, app),
	)
	defer resp.Body.Close()
	assert.Equal(t, 204, resp.StatusCode)
}

func TestHandler_ListUserOrders_RequiresAuth(t *testing.T) {
	ts, _, cancel := testutils.PrepareTestServer()
	defer cancel()
	resp, _ := testutils.DoTestRequest(t, ts, http.MethodGet, "/api/user/orders", nil)
	defer resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}
