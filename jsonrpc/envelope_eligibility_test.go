package jsonrpc

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

func TestEnvelopeResponseEligibility(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		code     Code
		dispatch bool
		id       any
	}{
		{name: "invalid version without ID", body: `{"jsonrpc":"1.0","method":"echo"}`, code: InvalidRequest},
		{name: "missing method without ID", body: `{"jsonrpc":"2.0"}`, code: InvalidRequest},
		{name: "nonstring method", body: `{"jsonrpc":"2.0","method":1}`, code: InvalidRequest},
		{name: "null envelope", body: `null`, code: InvalidRequest},
		{name: "scalar envelope", body: `true`, code: InvalidRequest},
		{name: "malformed JSON", body: `{"jsonrpc":`, code: ParseError},
		{name: "invalid supplied ID", body: `{"jsonrpc":"2.0","method":"echo","id":{}}`, code: InvalidRequest},
		{name: "valid notification", body: `{"jsonrpc":"2.0","method":"echo"}`, dispatch: true},
		{name: "unknown notification", body: `{"jsonrpc":"2.0","method":"missing"}`},
		{name: "invalid version with ID", body: `{"jsonrpc":"1.0","method":"echo","id":"one"}`, code: InvalidRequest, id: "one"},
		{name: "unknown method with ID", body: `{"jsonrpc":"2.0","method":"missing","id":"one"}`, code: MethodNotFound, id: "one"},
		{name: "explicit null ID", body: `{"jsonrpc":"2.0","method":"missing","id":null}`, code: MethodNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, transport := range []string{"http", "sse", "websocket"} {
				t.Run(transport, func(t *testing.T) {
					if transport == "websocket" {
						outcome := runWebSocketReceive(t, webSocketReceiveCase{client: sendFrames(tc.body, `{"jsonrpc":"2.0","method":"echo","id":"barrier"}`), recvs: 2})
						require.Len(t, outcome.errs, 2)
						for _, err := range outcome.errs {
							require.NoError(t, err)
						}
						if tc.code == 0 {
							require.Empty(t, outcome.errorCodes)
						} else {
							require.Equal(t, []Code{tc.code}, outcome.errorCodes)
							require.Equal(t, []any{tc.id}, outcome.errorIDs)
						}
						wantDispatches := 1
						if tc.dispatch {
							wantDispatches++
						}
						require.Len(t, outcome.dispatched, wantDispatches)
						return
					}
					dispatched := false
					dispatch := func(request *RawRequest) bool {
						if request.Method != "echo" {
							return false
						}
						dispatched = true
						return true
					}
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(tc.body))
					if transport == "http" {
						ServeHTTP(w, r, HTTPHandlerSpec{Decoder: loomhttp.RequestDecoder, Encoder: loomhttp.ResponseEncoder, Dispatch: func(_ context.Context, _ *http.Request, request *RawRequest, _ http.ResponseWriter) (bool, error) {
							return dispatch(request), nil
						}})
					} else {
						ServeSSE(w, r, SSEHandlerSpec{Decoder: loomhttp.RequestDecoder, Dispatch: func(ctx context.Context, r *http.Request, request *RawRequest, w http.ResponseWriter) (bool, bool, error) {
							return dispatch(request), true, nil
						}, SendError: func(ctx context.Context, _ *http.Request, w http.ResponseWriter, id any, code Code, message string, data any) error {
							return loomhttp.ResponseEncoder(ctx, w).Encode(MakeErrorResponse(id, code, message, data))
						}})
					}
					require.Equal(t, tc.dispatch, dispatched)
					if tc.code == 0 {
						require.Empty(t, w.Body.String())
						if transport == "sse" {
							require.Equal(t, http.StatusNoContent, w.Code)
						}
					} else {
						var response RawResponse
						require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response), w.Body.String())
						require.NotNil(t, response.Error)
						require.Equal(t, int(tc.code), response.Error.Code)
						require.Equal(t, tc.id, response.ID)
					}
				})
			}
		})
	}
}
