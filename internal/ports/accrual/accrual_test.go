package accrual_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	accrual2 "github.com/sergeii/practikum-go-gophermart/internal/ports/accrual"
	"github.com/sergeii/practikum-go-gophermart/pkg/encode"
)

func TestService_CheckOrder(t *testing.T) {
	tests := []struct {
		name       string
		code       int
		body       []byte
		wantStatus string
		wantErr    error
	}{
		{
			"positive case",
			200,
			encode.MustJSONMarshal(accrual2.OrderStatus{
				Number: "79927398713", Status: "PROCESSED", Accrual: decimal.RequireFromString("100.5"),
			}),
			"PROCESSED",
			nil,
		},
		{
			"another positive case",
			200,
			encode.MustJSONMarshal(
				accrual2.OrderStatus{Number: "79927398713", Status: "INVALID", Accrual: decimal.Zero},
			),
			"INVALID",
			nil,
		},
		{
			"order is not registered",
			204,
			nil,
			"",
			accrual2.ErrOrderNotFound,
		},
		{
			"rate limit exceeded without header",
			429,
			nil,
			"",
			accrual2.ErrRespInvalidWaitTime,
		},
		{
			"unexpected body",
			200,
			nil,
			"",
			accrual2.ErrRespInvalidData,
		},
		{
			"unexpected status",
			500,
			nil,
			"",
			accrual2.ErrRespInvalidStatus,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/orders/:order", func(c *gin.Context) {
				c.String(tt.code, string(tt.body))
			})
			ts := httptest.NewServer(r)
			service, err := accrual2.New(ts.URL)
			require.NoError(t, err)
			os, err := service.CheckOrder("79927398713")
			if tt.wantErr != nil {
				assert.Error(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStatus, os.Status)
			}
		})
	}
}

func TestService_CheckOrder_RetryAfter(t *testing.T) {
	tests := []struct {
		name      string
		headers   map[string]string
		wantRetry int
		want      bool
	}{
		{
			"positive case",
			map[string]string{"Retry-After": "60"},
			60,
			true,
		},
		{
			"empty value",
			map[string]string{"Retry-After": ""},
			0,
			false,
		},
		{
			"invalid integer",
			map[string]string{"Retry-After": "omg"},
			0,
			false,
		},
		{
			"negative integer",
			map[string]string{"Retry-After": "-1"},
			0,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/orders/:order", func(c *gin.Context) {
				for hdr, val := range tt.headers {
					c.Header(hdr, val)
				}
				c.Status(429)
			})
			ts := httptest.NewServer(r)
			service, err := accrual2.New(ts.URL)
			require.NoError(t, err)
			_, err = service.CheckOrder("79927398713")
			require.Error(t, err)
			if tt.want {
				tooManyReqs, ok := err.(*accrual2.TooManyRequestError) // nolint: errorlint
				require.True(t, ok)
				assert.Equal(t, uint(tt.wantRetry), tooManyReqs.RetryAfter)
			} else {
				assert.ErrorIs(t, err, accrual2.ErrRespInvalidWaitTime)
			}
		})
	}
}

func TestService_New_Validation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{
			"positive case - with port",
			"http://localhost:8081",
			nil,
		},
		{
			"positive case - with port and trailing slash",
			"http://localhost:8081/",
			nil,
		},
		{
			"empty address",
			"",
			accrual2.ErrConfigInvalidAddress,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := accrual2.New(tt.url)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
