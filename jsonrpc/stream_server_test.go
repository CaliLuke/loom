package jsonrpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func (failingReadCloser) Close() error {
	return nil
}

func TestServeSSEOwnsValidationAndNotificationSuppression(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		matched       bool
		unary         bool
		wantDispatch  int
		wantErrorCode Code
		wantStatus    int
	}{
		{name: "valid", body: `{"jsonrpc":"2.0","method":"watch","id":1}`, matched: true, wantDispatch: 1},
		{name: "invalid version", body: `{"jsonrpc":"1.0","method":"watch","id":1}`, wantErrorCode: InvalidRequest},
		{name: "unknown notification", body: `{"jsonrpc":"2.0","method":"missing"}`, wantDispatch: 1, wantStatus: http.StatusNoContent},
		{name: "unary notification", body: `{"jsonrpc":"2.0","method":"watch"}`, matched: true, unary: true, wantDispatch: 1, wantStatus: http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var dispatched int
			var errorCode Code
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ServeSSE(w, r, SSEHandlerSpec{
					Service: "clock",
					Decoder: loomhttp.RequestDecoder,
					Dispatch: func(context.Context, *http.Request, *RawRequest, http.ResponseWriter) (bool, bool, error) {
						dispatched++
						return test.matched, test.unary, nil
					},
					SendError: func(_ context.Context, _ *http.Request, _ http.ResponseWriter, _ any, code Code, _ string, _ any) error {
						errorCode = code
						return nil
					},
				})
			})
			request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(test.body))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			require.Equal(t, test.wantDispatch, dispatched)
			require.Equal(t, test.wantErrorCode, errorCode)
			if test.wantStatus != 0 {
				require.Equal(t, test.wantStatus, response.Code)
			}
		})
	}
}

func TestServeMixedOwnsTransportNegotiation(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		accept     string
		body       string
		wantHTTP   int
		wantSSE    int
		wantStatus int
	}{
		{name: "unary", method: http.MethodPost, body: `{"jsonrpc":"2.0","method":"tick","id":1}`, wantHTTP: 1},
		{name: "SSE", method: http.MethodPost, accept: "text/event-stream", body: `  {"jsonrpc":"2.0","method":"tick","id":1}`, wantSSE: 1},
		{name: "batch stays HTTP", method: http.MethodPost, accept: "text/event-stream", body: ` [{"jsonrpc":"2.0","method":"tick","id":1}]`, wantHTTP: 1},
		{name: "GET without listener", method: http.MethodGet, wantStatus: http.StatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var httpCalls int
			var sseCalls int
			spec := MixedHandlerSpec{
				HTTP: HTTPHandlerSpec{
					Service: "clock",
					Decoder: loomhttp.RequestDecoder,
					Encoder: loomhttp.ResponseEncoder,
					Dispatch: func(context.Context, *http.Request, *RawRequest, http.ResponseWriter) (bool, error) {
						httpCalls++
						return true, nil
					},
				},
				SSE: SSEHandlerSpec{
					Service: "clock",
					Decoder: loomhttp.RequestDecoder,
					Dispatch: func(context.Context, *http.Request, *RawRequest, http.ResponseWriter) (bool, bool, error) {
						sseCalls++
						return true, false, nil
					},
					SendError: func(context.Context, *http.Request, http.ResponseWriter, any, Code, string, any) error {
						return nil
					},
				},
			}
			request := httptest.NewRequest(test.method, "/rpc", strings.NewReader(test.body))
			request.Header.Set("Accept", test.accept)
			response := httptest.NewRecorder()

			ServeMixed(response, request, spec)

			require.Equal(t, test.wantHTTP, httpCalls)
			require.Equal(t, test.wantSSE, sseCalls)
			if test.wantStatus != 0 {
				require.Equal(t, test.wantStatus, response.Code)
			}
		})
	}
}

func TestServeMixedStopsAfterNegotiationReadFailure(t *testing.T) {
	var failures int
	var dispatches int
	spec := MixedHandlerSpec{
		HTTP: HTTPHandlerSpec{
			HandleFailure: func(_ context.Context, _ http.ResponseWriter, err error) {
				failures++
				require.ErrorContains(t, err, "failed to read request body")
			},
		},
		SSE: SSEHandlerSpec{
			Dispatch: func(context.Context, *http.Request, *RawRequest, http.ResponseWriter) (bool, bool, error) {
				dispatches++
				return true, false, nil
			},
		},
	}
	request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(""))
	request.Header.Set("Accept", "text/event-stream")
	request.Body = failingReadCloser{}

	ServeMixed(httptest.NewRecorder(), request, spec)

	require.Equal(t, 1, failures)
	require.Zero(t, dispatches)
}

func TestCompleteStreamOwnsFinalResponseSuppression(t *testing.T) {
	tests := []struct {
		name       string
		hasID      bool
		wantSend   bool
		wantEvents int
	}{
		{name: "notification", wantEvents: 1},
		{name: "explicit null ID", hasID: true, wantSend: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &httpEventRecorder{}
			ctx := loomtransport.WithObserver(context.Background(), recorder)
			called := false

			err := CompleteStream(ctx, test.hasID, nil, "result", func(response *Response) error {
				called = true
				require.Nil(t, response.ID)
				return nil
			})

			require.NoError(t, err)
			require.Equal(t, test.wantSend, called)
			require.Len(t, recorder.events, test.wantEvents)
			if test.wantEvents > 0 {
				require.Equal(t, loomtransport.ReasonStreamFinalResponseSuppressed, recorder.events[0].Reason)
			}
		})
	}
}

func TestCompleteStreamErrorSuppressesNotificationResponse(t *testing.T) {
	tests := []struct {
		name       string
		hasID      bool
		wantSend   bool
		wantEvents int
	}{
		{name: "notification", wantEvents: 1},
		{name: "explicit null ID", hasID: true, wantSend: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &httpEventRecorder{}
			ctx := loomtransport.WithObserver(context.Background(), recorder)
			called := false

			err := CompleteStreamError(ctx, test.hasID, func() error {
				called = true
				return nil
			})

			require.NoError(t, err)
			require.Equal(t, test.wantSend, called)
			require.Len(t, recorder.events, test.wantEvents)
			if test.wantEvents > 0 {
				require.Equal(t, loomtransport.ReasonStreamFinalResponseSuppressed, recorder.events[0].Reason)
			}
		})
	}
}
