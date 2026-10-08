package ticktock_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"example.com/http-ticktock/gen/clock"
	clockclient "example.com/http-ticktock/gen/http/clock/client"
	loomhttp "github.com/CaliLuke/loom/http"
)

type (
	sseResponseBody struct {
		io.Reader
		closeErr error
		closes   int
	}
	sseResponseDoer struct {
		response *http.Response
	}
)

func TestGeneratedSSEResponseOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, wantError string
		status                       int
		guarded                      bool
	}{
		{"unexpected status", "application/json", "unexpected status", http.StatusBadGateway, false},
		{"unexpected content type", "application/json", "unexpected content type", http.StatusOK, false},
		{"typed endpoint content type", "application/json", "unexpected content type", http.StatusOK, true},
		{"typed endpoint rejection", "application/problem+json", "denied", http.StatusUnauthorized, true},
		{"stream", "text/event-stream", "", http.StatusOK, false},
		{"stream without content type", "", "", http.StatusOK, false},
	} {
		for _, closeFails := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "/close succeeds", true: "/close fails"}[closeFails], func(t *testing.T) {
				body := &sseResponseBody{Reader: strings.NewReader(`{"type":"about:blank","title":"Unauthorized","status":401,"detail":"denied","instance":"test","code":"unauthorized"}`)}
				if closeFails {
					body.closeErr = errors.New("close failed")
				}
				response := &http.Response{StatusCode: tc.status, Body: body, Header: make(http.Header)}
				response.Header.Set("Content-Type", tc.contentType)
				client := clockclient.NewClient("http", "example.test", sseResponseDoer{response}, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
				endpoint := client.Tick()
				var payload any
				if tc.guarded {
					endpoint = client.Guarded()
					payload = &clock.GuardedPayload{}
				}
				result, err := endpoint(t.Context(), payload)
				if tc.wantError != "" {
					require.Nil(t, result)
					require.ErrorContains(t, err, tc.wantError)
					if closeFails {
						require.ErrorIs(t, err, body.closeErr)
					}
					require.Equal(t, 1, body.closes)
					return
				}
				require.NoError(t, err)
				require.Zero(t, body.closes, "successful handshake transfers the open body")
				stream, ok := result.(io.Closer)
				require.True(t, ok)
				err = stream.Close()
				if closeFails {
					require.ErrorIs(t, err, body.closeErr)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, 1, body.closes)
			})
		}
	}
}

func (b *sseResponseBody) Close() error {
	b.closes++
	return b.closeErr
}

func (d sseResponseDoer) Do(*http.Request) (*http.Response, error) {
	return d.response, nil
}
