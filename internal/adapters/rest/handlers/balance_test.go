package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testutils2 "github.com/sergeii/practikum-go-gophermart/internal/testutils"
	"github.com/sergeii/practikum-go-gophermart/pkg/encode"
)

type showBalanceRespSchema struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

func TestHandler_ShowUserBalance_OK(t *testing.T) {
	tests := []struct {
		name      string
		current   string
		withdrawn string
	}{
		{
			"zero points of each",
			"0",
			"0",
		},
		{
			"have current but no withdrawn",
			"100500.1",
			"0",
		},
		{
			"have a bit of both",
			"2048.1",
			"2022.91",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := decimal.RequireFromString(tt.current)
			withdrawn := decimal.RequireFromString(tt.withdrawn)
			accrued := current.Add(withdrawn)

			ts, app, cancel := testutils2.PrepareTestServer()
			defer cancel()

			u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")
			if !accrued.IsZero() {
				err := app.UserService.AccruePoints(context.TODO(), u.ID, accrued)
				require.NoError(t, err)
			}
			if !withdrawn.IsZero() {
				err := app.UserService.WithdrawPoints(context.TODO(), u.ID, withdrawn)
				require.NoError(t, err)
			}

			var respJSON showBalanceRespSchema
			resp, _ := testutils2.DoTestRequest(
				ts, http.MethodGet, "/api/user/balance", nil,
				testutils2.WithUser(u, app),
				testutils2.MustBindJSON(&respJSON),
			)
			resp.Body.Close()
			require.Equal(t, 200, resp.StatusCode)
			assert.Equal(t, encode.DecimalToFloat(current), respJSON.Current)
			assert.Equal(t, encode.DecimalToFloat(withdrawn), respJSON.Withdrawn)
		})
	}
}

func TestHandler_ShowUserBalance_RequiresAuth(t *testing.T) {
	ts, _, cancel := testutils2.PrepareTestServer()
	defer cancel()
	resp, _ := testutils2.DoTestRequest(ts, http.MethodGet, "/api/user/balance", nil)
	resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}
