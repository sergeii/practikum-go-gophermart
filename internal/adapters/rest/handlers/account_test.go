package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sergeii/practikum-go-gophermart/internal/pkg/testutils"
)

type registerUserReqSchema struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type registerUserRespSchema struct {
	Result struct {
		ID    int    `json:"id"`
		Login string `json:"login"`
	} `json:"result"`
}

type registerUserRespErrorSchema struct {
	Error string `json:"error"`
}

func parseAuthSetCookie(resp *http.Response) *http.Cookie {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "auth" {
			return cookie
		}
	}
	return nil
}

func TestHandler_RegisterUser_OK(t *testing.T) {
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	reqBody, _ := json.Marshal(&registerUserReqSchema{Login: "happy_shopper", Password: "secret"}) // nolint:errchkjson
	resp, respBody := testutils.DoTestRequest(t, ts, http.MethodPost, "/api/user/register", bytes.NewReader(reqBody))
	defer resp.Body.Close()
	var respJSON registerUserRespSchema
	json.Unmarshal([]byte(respBody), &respJSON) // nolint:errcheck
	assert.Equal(t, 200, resp.StatusCode)
	assert.True(t, respJSON.Result.ID > 0)
	assert.Equal(t, "happy_shopper", respJSON.Result.Login)

	u, err := app.UserService.Authenticate(context.TODO(), "happy_shopper", "secret")
	require.NoError(t, err)
	assert.Equal(t, respJSON.Result.ID, u.ID)
	assert.Equal(t, "happy_shopper", u.Login)

	cookie := parseAuthSetCookie(resp)
	require.NotNil(t, cookie)
	token, _ := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		return app.Cfg.SecretKey, nil
	})
	require.True(t, token.Valid)
	claims, _ := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "happy_shopper", claims["login"])
	assert.Equal(t, float64(u.ID), claims["id"])
}

func TestHandler_RegisterUser_Validation(t *testing.T) {
	tests := []struct {
		name        string
		login       string
		password    string
		wantSuccess bool
		wantStatus  int
	}{
		{
			"positive case",
			"shopper",
			"secret",
			true,
			200,
		},
		{
			"empty login",
			"",
			"secret",
			false,
			400,
		},
		{
			"empty login - spaces",
			"  ",
			"secret",
			false,
			400,
		},
		{
			"empty password",
			"secret",
			"",
			false,
			400,
		},
		{
			"empty password - spaces",
			"secret",
			"     ",
			false,
			400,
		},
		{
			"empty login and password",
			"",
			"",
			false,
			400,
		},
		{
			"empty login and password - spaces",
			"  ",
			"    ",
			false,
			400,
		},
		{
			"login is occupied",
			"happy_shopper",
			"secret",
			false,
			409,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, app, cancel := testutils.PrepareTestServer()
			defer cancel()

			_, err := app.UserService.RegisterNewUser(context.TODO(), "happy_shopper", "super_secret")
			require.NoError(t, err)

			reqBody, _ := json.Marshal(&registerUserReqSchema{Login: tt.login, Password: tt.password}) // nolint:errchkjson
			resp, respBody := testutils.DoTestRequest(
				t, ts, http.MethodPost, "/api/user/register", bytes.NewReader(reqBody),
			)
			defer resp.Body.Close()
			if tt.wantSuccess {
				assert.Equal(t, 200, resp.StatusCode)
				var respJSON registerUserRespSchema
				json.Unmarshal([]byte(respBody), &respJSON) // nolint:errcheck
				assert.True(t, respJSON.Result.ID > 0)
				assert.Equal(t, tt.login, respJSON.Result.Login)
			} else {
				assert.Equal(t, tt.wantStatus, resp.StatusCode)
				var respJSON registerUserRespErrorSchema
				json.Unmarshal([]byte(respBody), &respJSON) // nolint:errcheck
				assert.True(t, respJSON.Error != "")
			}
		})
	}
}

type loginUserReqSchema struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginUserRespErrorSchema struct {
	Error string `json:"error"`
}

func TestHandler_LoginUser_OK(t *testing.T) {
	ts, app, cancel := testutils.PrepareTestServer()
	defer cancel()

	u, err := app.UserService.RegisterNewUser(context.TODO(), "happy_shopper", "super_secret")
	require.NoError(t, err)

	reqBody, _ := json.Marshal(&loginUserReqSchema{Login: "happy_shopper", Password: "super_secret"}) // nolint:errchkjson
	resp, _ := testutils.DoTestRequest(t, ts, http.MethodPost, "/api/user/login", bytes.NewReader(reqBody))
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	cookie := parseAuthSetCookie(resp)
	require.NotNil(t, cookie)
	token, _ := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		return app.Cfg.SecretKey, nil
	})
	require.True(t, token.Valid)
	claims, _ := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "happy_shopper", claims["login"])
	assert.Equal(t, float64(u.ID), claims["id"])
}

func TestHandler_LoginUser_Validation(t *testing.T) {
	tests := []struct {
		name        string
		login       string
		password    string
		wantSuccess bool
		wantStatus  int
	}{
		{
			"positive case",
			"happy_shopper",
			"super_secret",
			true,
			200,
		},
		{
			"unknown user",
			"angry_shopper",
			"secret",
			false,
			401,
		},
		{
			"invalid password",
			"happy_shopper",
			"guessing",
			false,
			401,
		},
		{
			"empty login",
			"",
			"super_secret",
			false,
			400,
		},
		{
			"empty login - spaces",
			"  ",
			"super_secret",
			false,
			400,
		},
		{
			"empty password",
			"happy_shopper",
			"",
			false,
			400,
		},
		{
			"empty password - spaces",
			"happy_shopper",
			"     ",
			false,
			400,
		},
		{
			"empty login and password",
			"",
			"",
			false,
			400,
		},
		{
			"empty login and password - spaces",
			"  ",
			"    ",
			false,
			400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, app, cancel := testutils.PrepareTestServer()
			defer cancel()

			_, err := app.UserService.RegisterNewUser(context.TODO(), "happy_shopper", "super_secret")
			require.NoError(t, err)

			reqBody, _ := json.Marshal(&loginUserReqSchema{Login: tt.login, Password: tt.password}) // nolint:errchkjson
			resp, respBody := testutils.DoTestRequest(t, ts, http.MethodPost, "/api/user/login", bytes.NewReader(reqBody))
			defer resp.Body.Close()
			if tt.wantSuccess {
				assert.Equal(t, 200, resp.StatusCode)
				cookie := parseAuthSetCookie(resp)
				require.NotNil(t, cookie)
			} else {
				assert.Equal(t, tt.wantStatus, resp.StatusCode)
				var respJSON loginUserRespErrorSchema
				json.Unmarshal([]byte(respBody), &respJSON) // nolint:errcheck
				assert.True(t, respJSON.Error != "")
			}
		})
	}
}

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
			ts, app, cancel := testutils.PrepareTestServer()
			defer cancel()

			current := decimal.RequireFromString(tt.current)
			withdrawn := decimal.RequireFromString(tt.withdrawn)
			currentF64, _ := current.Float64()
			withdrawnF64, _ := withdrawn.Float64()

			u, _ := app.UserService.RegisterNewUser(context.TODO(), "shopper", "secret")
			err := app.UserService.AccruePoints(context.TODO(), u.ID, current.Add(withdrawn))
			require.NoError(t, err)
			err = app.UserService.WithdrawPoints(context.TODO(), u.ID, withdrawn)
			require.NoError(t, err)

			resp, respBody := testutils.DoTestRequest(
				t, ts, http.MethodGet, "/api/user/balance", nil,
				testutils.RequestWithUser(u, app),
			)
			defer resp.Body.Close()
			require.Equal(t, 200, resp.StatusCode)
			var respJSON showBalanceRespSchema
			json.Unmarshal([]byte(respBody), &respJSON) // nolint:errcheck
			assert.Equal(t, currentF64, respJSON.Current)
			assert.Equal(t, withdrawnF64, respJSON.Withdrawn)
		})
	}
}

func TestHandler_ShowUserBalance_RequiresAuth(t *testing.T) {
	ts, _, cancel := testutils.PrepareTestServer()
	defer cancel()
	resp, _ := testutils.DoTestRequest(t, ts, http.MethodGet, "/api/user/balance", nil)
	defer resp.Body.Close()
	assert.Equal(t, 401, resp.StatusCode)
}
