package testdata

// ObjectDefaultsHarness exercises mounted HTTP/JSON-RPC default decoding.
const ObjectDefaultsHarness = `package defaults_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	server "example.com/objectdefaults/gen/TRANSPORT/defaults/server"
	svc "example.com/objectdefaults/gen/defaults"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestObjectDefaults(t *testing.T) {
	const rpc = RPC
	var received *svc.SendPayload
	calls := 0
	endpoint := func(_ context.Context, payload any) (any, error) {
		calls++
		received = payload.(*svc.SendPayload)
		return nil, nil
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&svc.Endpoints{Send: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, SERVER_ARGS))
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"absent", "{}", true},
		{"zero", "{\"value\":{\"x\":0,\"label\":\"\"},\"items\":[],\"indexed\":{},\"maybe\":null}", true},
		{"invalid", "{\"value\":{\"x\":-1}}", false},
		{"null", "{\"value\":null}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, body := "/send", tc.body
			if rpc {
				path = "/rpc"
				body = "{\"jsonrpc\":\"2.0\",\"id\":\"1\",\"method\":\"send\",\"params\":" + body + "}"
			}
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			before := calls
			mux.ServeHTTP(response, request)
			if !tc.valid {
				require.Equal(t, before, calls)
				if rpc {
					require.Equal(t, http.StatusOK, response.Code)
					require.Contains(t, response.Body.String(), "error")
				} else {
					require.Equal(t, http.StatusBadRequest, response.Code)
					require.Contains(t, response.Body.String(), "detail")
				}
				return
			}
			require.Equal(t, before+1, calls, response.Body.String())
			if rpc {
				require.Equal(t, http.StatusOK, response.Code)
				require.NotContains(t, response.Body.String(), "error")
			} else {
				require.Equal(t, http.StatusNoContent, response.Code)
			}
			require.NotNil(t, received.Nested)
			require.Equal(t, 0, received.Nested.Child.Count)
			require.Equal(t, "nested", received.Nested.Child.Label)
			require.Equal(t, 5, received.Nested.Items[0].Count)
			require.NotNil(t, received.Value)
			if tc.name == "absent" {
				require.Equal(t, 1, received.Value.Count)
				require.Equal(t, "nested", received.Value.Label)
				require.Equal(t, 2, received.Items[0].Count)
				require.Equal(t, 3, received.Indexed["key"].Count)
				maybe, present := received.Maybe.Value()
				require.True(t, present)
				require.Equal(t, 4, maybe.Count)
			} else {
				require.Equal(t, 0, received.Value.Count)
				require.Equal(t, "", received.Value.Label)
				require.NotNil(t, received.Items)
				require.Empty(t, received.Items)
				require.NotNil(t, received.Indexed)
				require.Empty(t, received.Indexed)
				require.True(t, received.Maybe.IsNull())
			}
		})
	}
}
`
