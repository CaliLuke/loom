package jsonrpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

func TestWebSocketRequestFailureObservation(t *testing.T) {
	tests := []struct {
		name               string
		firstFrame         string
		firstResponse      bool
		firstCode          Code
		firstID            any
		wantReason         loomtransport.Reason
		wantRequestFailure bool
	}{
		{
			name:               "malformed JSON",
			firstFrame:         `{`,
			firstResponse:      true,
			firstCode:          ParseError,
			wantReason:         loomtransport.ReasonInvalidJSONRPCEnvelope,
			wantRequestFailure: true,
		},
		{
			name:               "invalid envelope",
			firstFrame:         `{"jsonrpc":"1.0","method":"echo","id":"bad"}`,
			firstResponse:      true,
			firstCode:          InvalidRequest,
			firstID:            "bad",
			wantReason:         loomtransport.ReasonInvalidJSONRPCEnvelope,
			wantRequestFailure: true,
		},
		{
			name:               "invalid envelope without ID still replies",
			firstFrame:         `{"jsonrpc":"1.0","method":"echo"}`,
			firstResponse:      true,
			firstCode:          InvalidRequest,
			wantReason:         loomtransport.ReasonInvalidJSONRPCEnvelope,
			wantRequestFailure: true,
		},
		{
			name:               "unknown method",
			firstFrame:         `{"jsonrpc":"2.0","method":"missing","id":"bad"}`,
			firstResponse:      true,
			firstCode:          MethodNotFound,
			firstID:            "bad",
			wantReason:         loomtransport.ReasonUnsupportedMethod,
			wantRequestFailure: true,
		},
		{
			name:               "unknown method notification remains suppressed",
			firstFrame:         `{"jsonrpc":"2.0","method":"missing"}`,
			wantReason:         loomtransport.ReasonUnsupportedMethod,
			wantRequestFailure: true,
		},
		{
			name:          "all requests succeed",
			firstFrame:    `{"jsonrpc":"2.0","method":"echo","id":"first"}`,
			firstResponse: true,
			firstID:       "first",
			wantReason:    loomtransport.ReasonOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			events := make(chan loomtransport.Event, 16)
			observer := loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
				events <- event
			})
			requestContext := loomtransport.WithObserver(context.Background(), observer)
			done := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer close(done)
				r.Pattern = "GET /rpc"
				ServeWebSocket(w, r.WithContext(requestContext), WebSocketHandlerSpec{
					Service:  "probe",
					Upgrader: &websocket.Upgrader{},
					Run: func(ctx context.Context, _ context.CancelFunc, _ *http.Request, _ http.ResponseWriter, stream *loomhttp.WebSocketStream) error {
						for range 2 {
							if err := ReceiveWebSocketRequest(ctx, stream,
								func(method string) bool { return method == "echo" },
								func(ctx context.Context, request *RawRequest) error {
									return stream.WriteJSON(ctx, MakeSuccessResponse(request.ID, "ok"))
								},
								func(ctx context.Context, id any, code Code, message string, data any) error {
									return stream.WriteJSON(ctx, MakeErrorResponse(id, code, message, data))
								},
							); err != nil {
								return err
							}
						}
						return nil
					},
					HandleFailure: func(_ context.Context, _ http.ResponseWriter, err error) {
						t.Errorf("unexpected WebSocket handler failure: %v", err)
					},
				})
			}))
			defer server.Close()

			conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			defer func() {
				require.NoError(t, conn.Close())
			}()
			require.NoError(t, conn.SetReadDeadline(time.Now().Add(3*time.Second)))

			require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(test.firstFrame)))
			if test.firstResponse {
				var first Response
				require.NoError(t, conn.ReadJSON(&first))
				if test.wantReason == loomtransport.ReasonOK {
					require.Nil(t, first.Error)
					require.Equal(t, test.firstID, first.ID)
					require.Equal(t, "ok", first.Result)
				} else {
					require.NotNil(t, first.Error)
					require.Equal(t, test.firstCode, first.Error.Code)
					require.Equal(t, test.firstID, first.ID)
				}
			}

			require.NoError(t, conn.WriteJSON(map[string]any{
				"jsonrpc": "2.0",
				"method":  "echo",
				"id":      "barrier",
			}))
			var barrier Response
			require.NoError(t, conn.ReadJSON(&barrier))
			require.Nil(t, barrier.Error)
			require.Equal(t, "barrier", barrier.ID)
			require.Equal(t, "ok", barrier.Result)

			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("WebSocket handler did not finish after the valid barrier request")
			}
			close(events)
			var starts, terminal, opens, closes int
			for event := range events {
				switch event.Kind {
				case loomtransport.EventKindRequestStart:
					starts++
				case loomtransport.EventKindRequestFailure, loomtransport.EventKindRequestFinish:
					terminal++
					if test.wantRequestFailure {
						require.Equal(t, loomtransport.EventKindRequestFailure, event.Kind)
					} else {
						require.Equal(t, loomtransport.EventKindRequestFinish, event.Kind)
					}
					require.Equal(t, test.wantReason, event.Reason)
				case loomtransport.EventKindStreamOpen:
					if event.Reason == loomtransport.ReasonOK && event.Method == "" {
						opens++
					}
				case loomtransport.EventKindStreamClose:
					if event.Reason == loomtransport.ReasonOK && event.Method == "" {
						closes++
					}
				}
			}
			require.Equal(t, 1, starts, "one connection request lifecycle")
			require.Equal(t, 1, terminal, "one connection terminal event")
			require.Equal(t, 1, opens, "one physical stream open")
			require.Equal(t, 1, closes, "one physical stream close")
		})
	}
}
