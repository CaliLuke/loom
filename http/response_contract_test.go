package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestValidateResponseContract(t *testing.T) {
	tests := []struct {
		name     string
		response *http.Response
		contract ResponseContractCase
		wantErr  string
		contains bool
	}{
		{
			name: "success",
			response: &http.Response{
				StatusCode: http.StatusAccepted,
				Header: http.Header{
					"Content-Type": []string{"application/json; charset=utf-8"},
					"X-Version":    []string{""},
					"Set-Cookie":   []string{"widget_session=abc; Path=/"},
				},
			},
			contract: ResponseContractCase{
				ID:              "widgets.show.success.202",
				Kind:            ResponseContractSuccess,
				StatusCode:      http.StatusAccepted,
				ContentTypes:    []string{"application/json"},
				RequiredHeaders: []string{"X-Version"},
				RequiredCookies: []string{"widget_session"},
			},
		},
		{
			name: "error",
			response: &http.Response{
				StatusCode: http.StatusNotFound,
				Header: http.Header{
					"Content-Type": []string{"application/problem+json"},
					"Loom-Error":   []string{"not_found"},
				},
			},
			contract: ResponseContractCase{
				ID:           "widgets.show.error.not_found.404",
				Kind:         ResponseContractError,
				StatusCode:   http.StatusNotFound,
				ErrorName:    "not_found",
				ContentTypes: []string{"application/problem+json"},
			},
		},
		{
			name:     "nil response",
			contract: ResponseContractCase{ID: "widgets.show.success.200"},
			wantErr:  `response contract "widgets.show.success.200": response is nil`,
		},
		{
			name: "status mismatch",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
			},
			contract: ResponseContractCase{ID: "widgets.show.success.202", StatusCode: http.StatusAccepted},
			wantErr:  `response contract "widgets.show.success.202": status is 200, want 202`,
		},
		{
			name: "malformed content type",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json; charset"}},
			},
			contract: ResponseContractCase{
				ID:           "widgets.show.success.200",
				StatusCode:   http.StatusOK,
				ContentTypes: []string{"application/json"},
			},
			wantErr:  `response contract "widgets.show.success.200": parse Content-Type "application/json; charset"`,
			contains: true,
		},
		{
			name: "content type mismatch",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
			},
			contract: ResponseContractCase{
				ID:           "widgets.show.success.200",
				StatusCode:   http.StatusOK,
				ContentTypes: []string{"application/json; profile=widget"},
			},
			wantErr: `response contract "widgets.show.success.200": Content-Type is "text/plain", want one of [application/json]`,
		},
		{
			name: "wildcard content type",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/pdf"}},
			},
			contract: ResponseContractCase{
				ID:           "files.download.success.200",
				StatusCode:   http.StatusOK,
				ContentTypes: []string{"*/*"},
			},
		},
		{
			name: "type wildcard content type",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/problem+json"}},
			},
			contract: ResponseContractCase{
				ID:           "errors.show.success.200",
				StatusCode:   http.StatusOK,
				ContentTypes: []string{"application/*"},
			},
		},
		{
			name: "error name mismatch",
			response: &http.Response{
				StatusCode: http.StatusNotFound,
				Header: http.Header{
					"Content-Type": []string{"application/problem+json"},
					"Loom-Error":   []string{"gone"},
				},
			},
			contract: ResponseContractCase{
				ID:           "widgets.show.error.not_found.404",
				Kind:         ResponseContractError,
				StatusCode:   http.StatusNotFound,
				ErrorName:    "not_found",
				ContentTypes: []string{"application/problem+json"},
			},
			wantErr: `response contract "widgets.show.error.not_found.404": Loom-Error is "gone", want "not_found"`,
		},
		{
			name: "success carries error name",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Loom-Error": []string{"not_found"}},
			},
			contract: ResponseContractCase{
				ID:         "widgets.show.success.200",
				Kind:       ResponseContractSuccess,
				StatusCode: http.StatusOK,
			},
			wantErr: `response contract "widgets.show.success.200": Loom-Error is "not_found", want empty`,
		},
		{
			name: "missing required header",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
			},
			contract: ResponseContractCase{
				ID:              "widgets.show.success.200",
				StatusCode:      http.StatusOK,
				RequiredHeaders: []string{"X-Version"},
			},
			wantErr: `response contract "widgets.show.success.200": required header "X-Version" is missing`,
		},
		{
			name: "missing required cookie",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
			},
			contract: ResponseContractCase{
				ID:              "widgets.show.success.200",
				StatusCode:      http.StatusOK,
				RequiredCookies: []string{"widget_session"},
			},
			wantErr: `response contract "widgets.show.success.200": required cookie "widget_session" is missing`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateResponseContract(test.response, test.contract)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			if test.contains {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.EqualError(t, err, test.wantErr)
		})
	}
}

func TestValidateSSEResponseContract(t *testing.T) {
	contract := ResponseContractCase{
		ID:           "events.watch.success.200",
		Kind:         ResponseContractSuccess,
		Transport:    ResponseContractSSE,
		StatusCode:   http.StatusOK,
		ContentTypes: []string{"text/event-stream"},
		SSE: &SSEResponseContract{
			Direction:         "server",
			MessageType:       "WatchEvent",
			DataEncoding:      "json",
			IDField:           "id",
			EventField:        "event",
			IDRequired:        true,
			EventTypeRequired: true,
			EventTypes:        []string{"created", "updated"},
			Terminal:          "eof",
		},
	}
	valid := func() *SSEResponseContractObservation {
		return &SSEResponseContractObservation{
			Response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream; charset=utf-8"}},
			},
			Events:        []SSEEvent{{ID: "1", Type: "created", Data: `{"id":"1"}`}},
			TerminalError: io.EOF,
		}
	}

	tests := []struct {
		name    string
		mutate  func(*SSEResponseContractObservation)
		wantErr string
	}{
		{name: "success"},
		{name: "clean nil terminal", mutate: func(observation *SSEResponseContractObservation) {
			observation.TerminalError = nil
		}},
		{name: "missing events", mutate: func(observation *SSEResponseContractObservation) {
			observation.Events = nil
		}, wantErr: "no SSE events were observed"},
		{name: "invalid JSON", mutate: func(observation *SSEResponseContractObservation) {
			observation.Events[0].Data = "not-json"
		}, wantErr: "data is not valid JSON"},
		{name: "missing id", mutate: func(observation *SSEResponseContractObservation) {
			observation.Events[0].ID = ""
		}, wantErr: "is missing an id"},
		{name: "missing event type", mutate: func(observation *SSEResponseContractObservation) {
			observation.Events[0].Type = ""
		}, wantErr: "is missing an event type"},
		{name: "unexpected event type", mutate: func(observation *SSEResponseContractObservation) {
			observation.Events[0].Type = "deleted"
		}, wantErr: `type is "deleted", want one of [created updated]`},
		{name: "terminal failure", mutate: func(observation *SSEResponseContractObservation) {
			observation.TerminalError = errors.New("connection reset")
		}, wantErr: "SSE terminal error, want clean EOF: connection reset"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observation := valid()
			if test.mutate != nil {
				test.mutate(observation)
			}
			err := ValidateSSEResponseContract(observation, contract)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestValidateWebSocketResponseContract(t *testing.T) {
	contract := ResponseContractCase{
		ID:         "events.watch.success.101",
		Kind:       ResponseContractSuccess,
		Transport:  ResponseContractWebSocket,
		StatusCode: http.StatusSwitchingProtocols,
		WebSocket: &WebSocketResponseContract{
			Direction:           "server",
			OutboundMessageType: "WatchStreamingResult",
			HandshakeHeaders: []string{
				"Connection",
				"Sec-WebSocket-Accept",
				"Upgrade",
			},
			Terminal: "normal_close",
		},
	}
	valid := func() *WebSocketResponseContractObservation {
		header := make(http.Header)
		header.Set("Connection", "Upgrade")
		header.Set("Sec-WebSocket-Accept", "key")
		header.Set("Upgrade", "websocket")
		return &WebSocketResponseContractObservation{
			Response: &http.Response{
				StatusCode: http.StatusSwitchingProtocols,
				Header:     header,
			},
			Messages:      []json.RawMessage{json.RawMessage(`{"message":"ready"}`)},
			TerminalError: &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "done"},
		}
	}
	require.ErrorContains(
		t,
		ValidateWebSocketResponseContract(nil, contract),
		"WebSocket observation is nil",
	)

	tests := []struct {
		name    string
		mutate  func(*WebSocketResponseContractObservation, *ResponseContractCase)
		wantErr string
	}{
		{name: "success"},
		{name: "nil observation", mutate: func(observation *WebSocketResponseContractObservation, _ *ResponseContractCase) {
			*observation = WebSocketResponseContractObservation{}
		}, wantErr: "response is nil"},
		{name: "wrong contract transport", mutate: func(_ *WebSocketResponseContractObservation, contract *ResponseContractCase) {
			contract.Transport = ResponseContractHTTP
		}, wantErr: "contract is not a WebSocket response"},
		{name: "missing messages", mutate: func(observation *WebSocketResponseContractObservation, _ *ResponseContractCase) {
			observation.Messages = nil
		}, wantErr: "no WebSocket messages were observed"},
		{name: "missing handshake header", mutate: func(observation *WebSocketResponseContractObservation, _ *ResponseContractCase) {
			observation.Response.Header.Del("Upgrade")
		}, wantErr: `WebSocket handshake header "Upgrade" is missing`},
		{name: "invalid connection header", mutate: func(observation *WebSocketResponseContractObservation, _ *ResponseContractCase) {
			observation.Response.Header.Set("Connection", "keep-alive")
		}, wantErr: `WebSocket handshake header "Connection" does not contain "Upgrade"`},
		{name: "invalid upgrade header", mutate: func(observation *WebSocketResponseContractObservation, _ *ResponseContractCase) {
			observation.Response.Header.Set("Upgrade", "h2c")
		}, wantErr: `WebSocket handshake header "Upgrade" is "h2c", want "websocket"`},
		{name: "empty accept header", mutate: func(observation *WebSocketResponseContractObservation, _ *ResponseContractCase) {
			observation.Response.Header.Set("Sec-WebSocket-Accept", "")
		}, wantErr: `WebSocket handshake header "Sec-WebSocket-Accept" is empty`},
		{name: "invalid JSON", mutate: func(observation *WebSocketResponseContractObservation, _ *ResponseContractCase) {
			observation.Messages[0] = json.RawMessage("not-json")
		}, wantErr: "message 0 is not valid JSON"},
		{name: "abnormal close", mutate: func(observation *WebSocketResponseContractObservation, _ *ResponseContractCase) {
			observation.TerminalError = &websocket.CloseError{Code: websocket.CloseAbnormalClosure}
		}, wantErr: "terminal error, want normal close"},
		{name: "missing close", mutate: func(observation *WebSocketResponseContractObservation, _ *ResponseContractCase) {
			observation.TerminalError = nil
		}, wantErr: "terminal error is nil, want normal close"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observation := valid()
			current := contract
			if test.mutate != nil {
				test.mutate(observation, &current)
			}
			err := ValidateWebSocketResponseContract(observation, current)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestValidateWebSocketResponseContractFinalMessage(t *testing.T) {
	contract := ResponseContractCase{
		ID:         "events.collect.success.101",
		Kind:       ResponseContractSuccess,
		Transport:  ResponseContractWebSocket,
		StatusCode: http.StatusSwitchingProtocols,
		WebSocket: &WebSocketResponseContract{
			Direction:           "client",
			InboundMessageType:  "CollectStreamingPayload",
			OutboundMessageType: "CollectResult",
			Terminal:            "final_message",
		},
	}
	observation := &WebSocketResponseContractObservation{
		Response: &http.Response{
			StatusCode: http.StatusSwitchingProtocols,
			Header:     make(http.Header),
		},
		Messages: []json.RawMessage{json.RawMessage(`{"count":1}`)},
	}
	require.NoError(t, ValidateWebSocketResponseContract(observation, contract))

	observation.Messages = append(observation.Messages, json.RawMessage(`{"count":2}`))
	require.ErrorContains(t, ValidateWebSocketResponseContract(observation, contract), "observed 2 WebSocket messages, want exactly one final message")
}
