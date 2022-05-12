package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testutils2 "github.com/sergeii/practikum-go-gophermart/internal/testutils"
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
	ts, app, cancel := testutils2.PrepareTestServer()
	defer cancel()

	var respJSON registerUserRespSchema
	resp, _ := testutils2.DoTestRequest(
		ts, http.MethodPost, "/api/user/register",
		testutils2.JSONReader(registerUserReqSchema{Login: "happy_shopper", Password: "secret"}),
		testutils2.MustBindJSON(&respJSON),
	)
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
			ts, app, cancel := testutils2.PrepareTestServer()
			defer cancel()

			_, err := app.UserService.RegisterNewUser(context.TODO(), "happy_shopper", "super_secret")
			require.NoError(t, err)

			resp, respBody := testutils2.DoTestRequest(
				ts, http.MethodPost, "/api/user/register",
				testutils2.JSONReader(registerUserReqSchema{Login: tt.login, Password: tt.password}),
			)
			resp.Body.Close()
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
	ts, app, cancel := testutils2.PrepareTestServer()
	defer cancel()

	u, err := app.UserService.RegisterNewUser(context.TODO(), "happy_shopper", "super_secret")
	require.NoError(t, err)

	resp, _ := testutils2.DoTestRequest(
		ts, http.MethodPost, "/api/user/login", testutils2.JSONReader(
			loginUserReqSchema{Login: "happy_shopper", Password: "super_secret"},
		),
	)
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
			ts, app, cancel := testutils2.PrepareTestServer()
			defer cancel()

			_, err := app.UserService.RegisterNewUser(context.TODO(), "happy_shopper", "super_secret")
			require.NoError(t, err)

			resp, respBody := testutils2.DoTestRequest(
				ts, http.MethodPost, "/api/user/login",
				testutils2.JSONReader(loginUserReqSchema{Login: tt.login, Password: tt.password}),
			)
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
