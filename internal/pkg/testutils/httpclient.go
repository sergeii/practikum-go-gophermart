package testutils

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sergeii/practikum-go-gophermart/internal/adapters/rest/middleware/auth"
	"github.com/sergeii/practikum-go-gophermart/internal/application"
	"github.com/sergeii/practikum-go-gophermart/internal/models"
)

type TestRequestOpt func(*http.Request, *http.Response)

func WithUser(u models.User, app *application.App) TestRequestOpt {
	return func(req *http.Request, resp *http.Response) {
		if req != nil {
			Authenticate(req, app, u)
		}
	}
}

func MustBindJSON(v interface{}) TestRequestOpt {
	return func(req *http.Request, resp *http.Response) {
		if resp != nil {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
			if err := json.Unmarshal(body, v); err != nil {
				panic(err)
			}
		}
	}
}

func DoTestRequest(
	t *testing.T, ts *httptest.Server,
	method, path string, body io.Reader,
	opts ...TestRequestOpt,
) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body) // nolint: noctx
	if err != nil {
		panic(err)
	}
	// run options that operate upon request
	for _, opt := range opts {
		opt(req, nil)
	}

	// disable redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// run options that operate upon response
	for _, opt := range opts {
		opt(nil, resp)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

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

func JSONReader(v interface{}) io.Reader {
	jsonBytes, err := json.Marshal(&v)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(jsonBytes)
}
