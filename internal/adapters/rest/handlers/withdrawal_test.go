package handlers_test

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sergeii/practikum-go-gophermart/internal/pkg/testutils"
)

type requestWithdrawalReqSchema struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type requestWithdrawalRespSchema struct {
	Result struct {
		ID          int       `json:"id"`
		Order       string    `json:"order"`
		Sum         float64   `json:"sum"`
		ProcessedAt time.Time `json:"processed_at"` // nolint: tagliatelle
	} `json:"result"`
}

type listWithdrawalItemSchema struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"` // nolint: tagliatelle
}

func TestHandler_RequestWithdrawal_OK(t *testing.T) {
	ctx := context.TODO()
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	before := time.Now()
	u, _ := app.UserService.RegisterNewUser(ctx, "shopper", "secret")
	err := app.UserService.AccruePoints(ctx, u.ID, decimal.RequireFromString("100"))
	require.NoError(t, err)

	withdrawals := []struct {
		order string
		sum   float64
	}{
		{"4561261212345467", 5.99}, {"49927398716", 50.01},
	}

	for _, item := range withdrawals {
		var respJSON requestWithdrawalRespSchema
		resp, _ := testutils.DoTestRequest(
			t, ts, http.MethodPost, "/api/user/balance/withdraw",
			testutils.JSONReader(requestWithdrawalReqSchema{item.order, item.sum}),
			testutils.WithUser(u, app),
			testutils.MustBindJSON(&respJSON),
		)
		assert.Equal(t, 200, resp.StatusCode)
		assert.True(t, respJSON.Result.ID > 0)
		assert.Equal(t, item.order, respJSON.Result.Order)
		assert.Equal(t, item.sum, respJSON.Result.Sum)
		assert.True(t, respJSON.Result.ProcessedAt.After(before))
		assert.True(t, respJSON.Result.ProcessedAt.Before(time.Now()))
	}

	balance, _ := app.UserService.GetBalance(ctx, u.ID)
	assert.Equal(t, "44", balance.Current.String())
	assert.Equal(t, "56", balance.Withdrawn.String())

	userWithdrawals, err := app.WithdrawalService.GetUserWithdrawals(ctx, u.ID)
	require.NoError(t, err)

	assert.Len(t, userWithdrawals, 2)
}

func TestHandler_RequestWithdrawal_Validation(t *testing.T) {
	tests := []struct {
		name       string
		number     string
		sum        float64
		want       bool
		wantStatus int
	}{
		{
			"positive case",
			"49927398716",
			10,
			true,
			200,
		},
		{
			"positive case - full sum",
			"49927398716",
			100,
			true,
			200,
		},
		{
			"order already withdrawn",
			"4561261212345467",
			10,
			false,
			409,
		},
		{
			"invalid number format",
			"11111",
			10,
			false,
			422,
		},
		{
			"not enough points",
			"49927398716",
			100.5,
			false,
			402,
		},
		{
			"zero sum",
			"49927398716",
			0,
			false,
			422,
		},
		{
			"negative sum",
			"49927398716",
			-10,
			false,
			422,
		},
	}

	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			ctx := context.TODO()
			ts, app, cancel := testutils.PrepareTestServer()
			defer cancel()

			hundred := decimal.RequireFromString("100")
			ten := decimal.RequireFromString("10")

			u, _ := app.UserService.RegisterNewUser(ctx, "shopper", "secret")
			other, _ := app.UserService.RegisterNewUser(ctx, "other", "secret_too")
			err := app.UserService.AccruePoints(ctx, u.ID, hundred)
			require.NoError(t, err)
			err = app.UserService.AccruePoints(ctx, other.ID, ten)
			require.NoError(t, err)
			_, err = app.WithdrawalService.RequestWithdrawal(ctx, "4561261212345467", other.ID, ten)
			require.NoError(t, err)

			var respJSON requestWithdrawalRespSchema
			resp, _ := testutils.DoTestRequest(
				t, ts, http.MethodPost, "/api/user/balance/withdraw",
				testutils.JSONReader(requestWithdrawalReqSchema{tt.number, tt.sum}),
				testutils.WithUser(u, app),
				testutils.MustBindJSON(&respJSON),
			)
			balance, _ := app.UserService.GetBalance(ctx, u.ID)
			userWithdrawals, _ := app.WithdrawalService.GetUserWithdrawals(ctx, u.ID)

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.want {
				withdrawn := decimal.NewFromFloat(tt.sum)
				assert.True(t, respJSON.Result.ID > 0)
				assert.Equal(t, withdrawn.String(), balance.Withdrawn.String())
				assert.Equal(t, hundred.Sub(withdrawn).String(), balance.Current.String())
				assert.Len(t, userWithdrawals, 1)
			} else {
				assert.Equal(t, 0, respJSON.Result.ID)
				assert.Equal(t, "0", balance.Withdrawn.String())
				assert.Equal(t, "100", balance.Current.String())
				assert.Len(t, userWithdrawals, 0)
			}
		})
	}
}

func TestHandler_RequestWithdrawal_NotEnoughPointsRace(t *testing.T) {
	ctx := context.TODO()
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	u, _ := app.UserService.RegisterNewUser(ctx, "shopper", "secret")
	err := app.UserService.AccruePoints(ctx, u.ID, decimal.RequireFromString("10"))
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	var errCount int64
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			var respJSON requestWithdrawalRespSchema
			resp, _ := testutils.DoTestRequest(
				t, ts, http.MethodPost, "/api/user/balance/withdraw",
				testutils.JSONReader(requestWithdrawalReqSchema{testutils.NewLuhnNumber(16), 3.5}),
				testutils.WithUser(u, app),
				testutils.MustBindJSON(&respJSON),
			)
			if resp.StatusCode != 200 {
				assert.Equal(t, 402, resp.StatusCode)
				atomic.AddInt64(&errCount, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	balance, _ := app.UserService.GetBalance(ctx, u.ID)
	userWithdrawals, _ := app.WithdrawalService.GetUserWithdrawals(ctx, u.ID)
	assert.Equal(t, "7", balance.Withdrawn.String())
	assert.Equal(t, "3", balance.Current.String())
	assert.Len(t, userWithdrawals, 2)
	assert.Equal(t, 3, int(errCount))
}

func TestHandler_RequestWithdrawal_LuhnValidation(t *testing.T) {
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
			ctx := context.TODO()
			ts, app, cancel := testutils.PrepareTestServer()
			defer cancel()
			u, _ := app.UserService.RegisterNewUser(ctx, "shopper", "secret")
			err := app.UserService.AccruePoints(ctx, u.ID, decimal.RequireFromString("100"))
			require.NoError(t, err)

			var respJSON requestWithdrawalRespSchema
			resp, _ := testutils.DoTestRequest(
				t, ts, http.MethodPost, "/api/user/balance/withdraw",
				testutils.JSONReader(requestWithdrawalReqSchema{tt.number, 1}),
				testutils.WithUser(u, app),
				testutils.MustBindJSON(&respJSON),
			)
			balance, _ := app.UserService.GetBalance(ctx, u.ID)
			if tt.want {
				assert.Equal(t, 200, resp.StatusCode)
				assert.True(t, respJSON.Result.ID > 0)
				assert.Equal(t, "1", balance.Withdrawn.String())
			} else {
				assert.Equal(t, 422, resp.StatusCode)
				assert.Equal(t, 0, respJSON.Result.ID)
				assert.Equal(t, "0", balance.Withdrawn.String())
			}
		})
	}
}

func TestHandler_RequestWithdrawal_RequiresAuth(t *testing.T) {
	ts, _, cancel := testutils.PrepareTestServer()
	defer cancel()

	resp, _ := testutils.DoTestRequest(t, ts, http.MethodPost, "/api/user/balance/withdraw", nil)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestHandler_ListUserWithdrawals_OK(t *testing.T) {
	ctx := context.TODO()
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	u, _ := app.UserService.RegisterNewUser(ctx, "shopper", "secret")
	other, _ := app.UserService.RegisterNewUser(ctx, "other", "secret_too")
	app.UserService.AccruePoints(ctx, u.ID, decimal.RequireFromString("10"))      // nolint:errcheck
	app.UserService.AccruePoints(ctx, other.ID, decimal.RequireFromString("100")) // nolint:errcheck

	app.WithdrawalService.RequestWithdrawal(ctx, "1234567812345670", u.ID, decimal.NewFromFloat(3.5)) // nolint:errcheck
	app.WithdrawalService.RequestWithdrawal(ctx, "4561261212345467", other.ID, decimal.NewFromInt(1)) // nolint:errcheck
	app.WithdrawalService.RequestWithdrawal(ctx, "2538566283278270", u.ID, decimal.NewFromFloat(2.1)) // nolint:errcheck

	uItems := make([]listWithdrawalItemSchema, 0)
	resp, _ := testutils.DoTestRequest(
		t, ts, http.MethodGet, "/api/user/balance/withdrawals", nil,
		testutils.WithUser(u, app),
		testutils.MustBindJSON(&uItems),
	)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Len(t, uItems, 2)
	assert.Equal(t, "1234567812345670", uItems[0].Order)
	assert.Equal(t, 3.5, uItems[0].Sum)
	assert.Equal(t, "2538566283278270", uItems[1].Order)
	assert.Equal(t, 2.1, uItems[1].Sum)

	oItems := make([]listWithdrawalItemSchema, 0)
	resp, _ = testutils.DoTestRequest(
		t, ts, http.MethodGet, "/api/user/balance/withdrawals", nil,
		testutils.WithUser(other, app),
		testutils.MustBindJSON(&oItems),
	)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Len(t, oItems, 1)
	assert.Equal(t, "4561261212345467", oItems[0].Order)
	assert.Equal(t, 1.0, oItems[0].Sum)
}

func TestHandler_ListUserWithdrawals_NoWithdrawalsForUser(t *testing.T) {
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")

	resp, _ := testutils.DoTestRequest(
		t, ts, http.MethodGet, "/api/user/balance/withdrawals", nil,
		testutils.WithUser(u, app),
	)
	assert.Equal(t, 204, resp.StatusCode)
}

func TestHandler_ListUserWithdrawals_RequiresAuth(t *testing.T) {
	ts, _, cancel := testutils.PrepareTestServer()
	defer cancel()
	resp, _ := testutils.DoTestRequest(t, ts, http.MethodGet, "/api/user/balance/withdrawals", nil)
	assert.Equal(t, 401, resp.StatusCode)
}
