package jsonrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
	loom "github.com/CaliLuke/loom/pkg"
)

type httpEventRecorder struct {
	events []loomtransport.Event
}

func (r *httpEventRecorder) ObserveEvent(_ context.Context, event loomtransport.Event) {
	r.events = append(r.events, event)
}

func TestHTTPHandlerOwnsEnvelopeDispatch(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantDispatched int
		wantResponses  int
		wantCode       Code
		wantReason     loomtransport.Reason
		wantNilID      bool
	}{
		{
			name:           "single request",
			body:           `{"jsonrpc":"2.0","method":"echo","params":{"value":"a"},"id":"one"}`,
			wantDispatched: 1,
			wantResponses:  1,
			wantReason:     loomtransport.ReasonOK,
		},
		{
			name:           "notification suppresses response",
			body:           `{"jsonrpc":"2.0","method":"echo","params":{"value":"a"}}`,
			wantDispatched: 1,
			wantReason:     loomtransport.ReasonOK,
		},
		{
			name:          "unsupported request",
			body:          `{"jsonrpc":"2.0","method":"missing","id":"one"}`,
			wantResponses: 1,
			wantCode:      MethodNotFound,
			wantReason:    loomtransport.ReasonUnsupportedMethod,
		},
		{
			name:          "invalid version",
			body:          `{"jsonrpc":"1.0","method":"echo","id":"one"}`,
			wantResponses: 1,
			wantCode:      InvalidRequest,
			wantReason:    loomtransport.ReasonInvalidJSONRPCEnvelope,
		},
		{
			name:          "invalid notification is an invalid request",
			body:          `{"jsonrpc":"1.0","method":"echo"}`,
			wantResponses: 1,
			wantCode:      InvalidRequest,
			wantReason:    loomtransport.ReasonInvalidJSONRPCEnvelope,
			wantNilID:     true,
		},
		{
			name: "batch omits notification response",
			body: `[
				{"jsonrpc":"2.0","method":"echo","id":"one"},
				{"jsonrpc":"2.0","method":"echo"},
				{"jsonrpc":"2.0","method":"echo","id":"two"}
			]`,
			wantDispatched: 3,
			wantResponses:  2,
			wantReason:     loomtransport.ReasonOK,
		},
		{
			name: "batch permits leading whitespace",
			body: `  
				[{"jsonrpc":"2.0","method":"echo","id":"one"}]`,
			wantDispatched: 1,
			wantResponses:  1,
			wantReason:     loomtransport.ReasonOK,
		},
		{
			name:          "empty batch",
			body:          `[]`,
			wantResponses: 1,
			wantCode:      InvalidRequest,
			wantReason:    loomtransport.ReasonInvalidJSONRPCBatch,
		},
		{
			name:          "invalid JSON",
			body:          `{`,
			wantResponses: 1,
			wantCode:      ParseError,
			wantReason:    loomtransport.ReasonInvalidJSONRPCEnvelope,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var dispatched int
			recorder := &httpEventRecorder{}
			handler := NewHTTPHandler(HTTPHandlerSpec{
				Service: "echo",
				Decoder: loomhttp.RequestDecoder,
				Encoder: loomhttp.ResponseEncoder,
				Dispatch: func(ctx context.Context, _ *http.Request, request *RawRequest, w http.ResponseWriter) (bool, error) {
					if request.Method != "echo" {
						return false, nil
					}
					dispatched++
					response := MakeSuccessResponse(request.ID, map[string]any{"ok": true})
					return true, loomhttp.ResponseEncoder(ctx, w).Encode(response)
				},
				HandleFailure: func(_ context.Context, _ http.ResponseWriter, err error) {
					require.NoError(t, err)
				},
			})
			request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			require.Equal(t, test.wantDispatched, dispatched)
			require.Len(t, recorder.events, 2)
			require.Equal(t, test.wantReason, recorder.events[1].Reason)
			if test.wantResponses == 0 {
				require.Empty(t, response.Body.String())
				return
			}
			if strings.HasPrefix(strings.TrimSpace(test.body), "[") && test.body != "[]" {
				var responses []Response
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &responses))
				require.Len(t, responses, test.wantResponses)
				return
			}
			var result Response
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
			if test.wantCode != 0 {
				require.NotNil(t, result.Error)
				require.Equal(t, test.wantCode, result.Error.Code)
			}
			if test.wantNilID {
				require.Nil(t, result.ID)
			}
		})
	}
}

func TestFirstJSONTokenBoundsAndPreservesWhitespaceLookahead(t *testing.T) {
	body := strings.Repeat(" ", maxJSONPrefixWhitespace+1) + "["
	reader := bufio.NewReader(strings.NewReader(body))

	first, err := firstJSONToken(reader)

	require.NoError(t, err)
	require.Zero(t, first)
	require.Equal(t, maxJSONPrefixWhitespace, reader.Buffered())
	remaining, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, body, string(remaining))
}

func TestHTTPHandlerRecordsBatchCount(t *testing.T) {
	recorder := &httpEventRecorder{}
	handler := NewHTTPHandler(HTTPHandlerSpec{
		Service: "echo",
		Decoder: loomhttp.RequestDecoder,
		Encoder: loomhttp.ResponseEncoder,
		Dispatch: func(context.Context, *http.Request, *RawRequest, http.ResponseWriter) (bool, error) {
			return true, nil
		},
	})
	request := httptest.NewRequest(
		http.MethodPost,
		"/rpc",
		strings.NewReader(`[{"jsonrpc":"2.0","method":"echo"},{"jsonrpc":"2.0","method":"echo"}]`),
	)
	request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))

	handler.ServeHTTP(httptest.NewRecorder(), request)

	require.Len(t, recorder.events, 2)
	require.Equal(t, 2, recorder.events[1].BatchCount)
}

func TestHTTPHandlerSuppressesNotificationFailureResponse(t *testing.T) {
	handler := NewHTTPHandler(HTTPHandlerSpec{
		Service: "echo",
		Decoder: loomhttp.RequestDecoder,
		Encoder: loomhttp.ResponseEncoder,
		Dispatch: func(context.Context, *http.Request, *RawRequest, http.ResponseWriter) (bool, error) {
			return true, errors.New("endpoint failed")
		},
		HandleFailure: func(ctx context.Context, w http.ResponseWriter, err error) {
			require.Error(t, err)
			require.NoError(t, loomhttp.ResponseEncoder(ctx, w).Encode(MakeErrorResponse(nil, InternalError, "Internal error", nil)))
		},
	})
	request := httptest.NewRequest(
		http.MethodPost,
		"/rpc",
		strings.NewReader(`{"jsonrpc":"2.0","method":"echo"}`),
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Empty(t, response.Body.String())
}

func TestCodeForServiceError(t *testing.T) {
	tests := []struct {
		name string
		err  *loom.ServiceError
		want Code
	}{
		{name: "nil", want: InternalError},
		{name: "request too large", err: &loom.ServiceError{Name: loom.RequestBodyTooLarge}, want: InvalidRequest},
		{name: "invalid field", err: &loom.ServiceError{Name: loom.InvalidFieldType}, want: InvalidParams},
		{name: "internal", err: &loom.ServiceError{Name: "unexpected"}, want: InternalError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, CodeForServiceError(test.err))
		})
	}
}
