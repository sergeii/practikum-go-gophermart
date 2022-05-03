package testutils

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sergeii/practikum-go-gophermart/internal/application"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
	"github.com/sergeii/practikum-go-gophermart/internal/services/rest/middleware/auth"
)

type TestRequestOpt func(r *http.Request)

func RequestWithUser(u models.User, app *application.App) TestRequestOpt {
	return func(r *http.Request) {
		Authenticate(r, app, u)
	}
}

func DoTestRequest(
	t *testing.T, ts *httptest.Server,
	method, path string, body io.Reader,
	opts ...TestRequestOpt,
) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	require.NoError(t, err)

	for _, opt := range opts {
		opt(req)
	}

	// disable redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	require.NoError(t, err)

	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func Authenticate(r *http.Request, app *application.App, u models.User) *http.Cookie {
	jwtToken, err := auth.GenerateAuthTokenCookie(u, app.Cfg.SecretKey)
	if err != nil {
		panic(err)
	}
	cookie := &http.Cookie{
		Name:     auth.CookieName,
		Value:    jwtToken,
		Path:     "/",
		Expires:  time.Now().Add(auth.CookieAge),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if r != nil {
		r.AddCookie(cookie)
	}
	return cookie
}
