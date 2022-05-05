package server_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sergeii/practikum-go-gophermart/pkg/http/server"
)

func reverseBytes(s []byte) []byte {
	i := 0
	j := len(s) - 1
	for i < j {
		s[i], s[j] = s[j], s[i]
		i++
		j--
	}
	return s
}

func TestHTTPServerListenAndServe(t *testing.T) {
	ready := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svr, err := server.New(
		"localhost:0", // 0 - listen an any available port
		server.WithHandler(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			rw.WriteHeader(http.StatusTeapot)
			rw.Write(reverseBytes(body)) // nolint:errcheck
		})),
		server.WithReadySignal(func() {
			ready <- struct{}{}
		}),
	)
	defer svr.Stop() // nolint: errcheck
	require.NoError(t, err)

	go func() {
		svr.ListenAndServe(ctx) // nolint: errcheck
	}()
	// wait for the server to start
	<-ready

	svrAddr := fmt.Sprintf("http://%s", svr.ListenAddr())
	resp, err := http.Post(svrAddr, "application/octet-stream", strings.NewReader("Hello World!")) // nolint: gosec
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 418, resp.StatusCode)
	respBody, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "!dlroW olleH", string(respBody))
}
