package log

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/observability/diagnostics"
	"github.com/stretchr/testify/require"
)

func TestAccessLogURLCredentials(t *testing.T) {
	for _, tc := range []struct {
		name, target, secret string
		fields               diagnostics.Fields
	}{
		{"key", "http://example.test/path?key=private-key&view=full", "private-key", diagnostics.Fields{}},
		{"camel case", "http://example.test/path?apiKey=private-key&view=full", "private-key", diagnostics.Fields{}},
		{"userinfo", "http://user:private-key@example.test/path?view=full", "private-key", diagnostics.Fields{}},
		{"repeated", "http://example.test/path?token=private-key&token=second&view=full", "private-key", diagnostics.Fields{}},
		{"escaped", "http://example.test/path?ticket=private%2Dkey&view=full", "private-key", diagnostics.Fields{Query: []string{"ticket"}}},
		{"custom", "http://example.test/path?ticket=private-key&view=full", "private-key", diagnostics.Fields{Query: []string{"ticket"}}},
		{"path", "http://example.test/private-key?view=full", "private-key", diagnostics.Fields{Path: true}},
	} {
		for _, transport := range []string{"server", "client", "client failure", "client error status"} {
			t.Run(tc.name+"/"+transport, func(t *testing.T) {
				var output bytes.Buffer
				ctx := newTestContext(&output)
				request := httptest.NewRequest(http.MethodGet, tc.target, nil)
				original := request.URL.String()
				calls := 0
				var serverOptions []HTTPLogOption
				var clientOptions []HTTPClientLogOption
				if len(tc.fields.Query) > 0 || tc.fields.Path {
					serverOptions = append(serverOptions, WithURLCredentials(tc.fields))
					clientOptions = append(clientOptions, WithClientURLCredentials(tc.fields))
				}
				if transport == "server" {
					HTTP(ctx, serverOptions...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls++
						require.Equal(t, original, r.URL.String())
						w.WriteHeader(http.StatusOK)
					})).ServeHTTP(httptest.NewRecorder(), request)
					require.Contains(t, output.String(), "msg=start")
					require.Contains(t, output.String(), "msg=end")
				} else {
					failure := errors.New("connection failed")
					client := Client(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						require.Equal(t, original, r.URL.String())
						if transport == "client failure" {
							return nil, failure
						}
						if transport == "client error status" {
							return &http.Response{StatusCode: http.StatusBadRequest, Status: "400 Bad Request", Body: http.NoBody}, nil
						}
						return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: http.NoBody}, nil
					}), clientOptions...)
					response, err := client.RoundTrip(request.WithContext(ctx))
					if transport == "client failure" {
						require.ErrorIs(t, err, failure)
					} else {
						require.NoError(t, err)
						require.NoError(t, response.Body.Close())
					}
				}
				require.Equal(t, 1, calls)
				require.Equal(t, original, request.URL.String())
				require.Contains(t, output.String(), "view=full")
				require.NotContains(t, output.String(), tc.secret)
				require.Contains(t, output.String(), "REDACTED")
				if tc.name == "repeated" {
					require.NotContains(t, output.String(), "second")
				}
				if tc.name == "userinfo" {
					require.NotContains(t, output.String(), "user:")
				}
			})
		}
	}
}

func TestAccessLogCredentialOptionLifetime(t *testing.T) {
	first := diagnostics.Fields{Query: []string{"first"}}
	second := diagnostics.Fields{Query: []string{"second"}}
	serverFirst, serverSecond := WithURLCredentials(first), WithURLCredentials(second)
	clientFirst, clientSecond := WithClientURLCredentials(first), WithClientURLCredentials(second)
	first.Query[0], second.Query[0] = "ordinary", "ordinary"
	for attempt := range 2 {
		t.Run(fmt.Sprint(attempt), func(t *testing.T) {
			for _, target := range []string{
				"http://example.test/path?first=private-one&second=private-two&ordinary=visible",
				"http://example.test/path?first=private-one;bad&ordinary=visible",
			} {
				for _, mode := range []string{"server", "client"} {
					var output bytes.Buffer
					ctx := newTestContext(&output)
					req := httptest.NewRequest(http.MethodGet, target, nil)
					if mode == "server" {
						HTTP(ctx, serverFirst, serverSecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							require.Equal(t, target, r.URL.String())
							w.WriteHeader(http.StatusNoContent)
						})).ServeHTTP(httptest.NewRecorder(), req)
					} else {
						resp, err := Client(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
							require.Equal(t, target, r.URL.String())
							return &http.Response{StatusCode: http.StatusNoContent, Status: "204 No Content", Body: http.NoBody}, nil
						}), clientFirst, clientSecond).RoundTrip(req.WithContext(ctx))
						require.NoError(t, err)
						require.NoError(t, resp.Body.Close())
					}
					require.NotContains(t, output.String(), "private-one")
					require.NotContains(t, output.String(), "private-two")
					if strings.Contains(target, ";bad") {
						require.Contains(t, output.String(), "invalid")
					} else {
						require.Contains(t, output.String(), "ordinary=visible")
					}
					require.Equal(t, target, req.URL.String())
				}
			}
		})
	}
}
