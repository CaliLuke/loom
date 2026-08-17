package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
	loom "github.com/CaliLuke/loom/pkg"
)

type mountedHandler struct {
	t *testing.T
}

type unaryEventRecorder struct {
	events []loomtransport.Event
}

func (r *unaryEventRecorder) ObserveEvent(_ context.Context, event loomtransport.Event) {
	r.events = append(r.events, event)
}

func (h mountedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	require.Equal(h.t, "GET /items", r.Pattern)
	w.WriteHeader(http.StatusNoContent)
}

func TestMountHandlerDispatchesHTTPHandler(t *testing.T) {
	mux := NewMuxer()
	MountHandler(mux, http.MethodGet, "/items", mountedHandler{t: t})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/items", nil)
	mux.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestUnaryHandlerLifecycle(t *testing.T) {
	errDecode := errors.New("decode")
	errInvoke := errors.New("invoke")
	errEncode := errors.New("encode")

	tests := []struct {
		name        string
		decodeErr   error
		invokeErr   error
		encodeErr   error
		wantError   error
		wantInvoked bool
		wantEncoded bool
		wantFailure bool
		wantStatus  int
		wantReason  loomtransport.Reason
	}{
		{name: "success", wantInvoked: true, wantEncoded: true, wantStatus: http.StatusCreated, wantReason: loomtransport.ReasonOK},
		{name: "decode failure", decodeErr: errDecode, wantError: errDecode, wantStatus: http.StatusBadRequest, wantReason: loomtransport.ReasonRequestDecodeFailed},
		{name: "invoke failure", invokeErr: errInvoke, wantError: errInvoke, wantInvoked: true, wantStatus: http.StatusBadRequest, wantReason: loomtransport.ReasonHandlerError},
		{name: "encode failure", encodeErr: errEncode, wantError: errEncode, wantInvoked: true, wantEncoded: true, wantFailure: true, wantStatus: http.StatusOK, wantReason: loomtransport.ReasonResponseWriteFailed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var invoked, encoded, failed bool
			recorder := &unaryEventRecorder{}
			handler := NewUnaryHandler(UnaryHandlerSpec[string, int]{
				Service: "catalog",
				Method:  "create",
				Decode: func(*http.Request) (string, error) {
					return "payload", test.decodeErr
				},
				Invoke: func(ctx context.Context, payload string) (int, error) {
					invoked = true
					require.Equal(t, "payload", payload)
					require.Equal(t, "catalog", ctx.Value(loom.ServiceKey))
					require.Equal(t, "create", ctx.Value(loom.MethodKey))
					return 42, test.invokeErr
				},
				EncodeResponse: func(_ context.Context, w http.ResponseWriter, result int) error {
					encoded = true
					require.Equal(t, 42, result)
					if test.encodeErr == nil {
						w.WriteHeader(http.StatusCreated)
					}
					return test.encodeErr
				},
				EncodeError: func(_ context.Context, w http.ResponseWriter, err error) error {
					require.ErrorIs(t, err, test.wantError)
					w.WriteHeader(http.StatusBadRequest)
					return nil
				},
				HandleFailure: func(_ context.Context, _ http.ResponseWriter, err error) {
					failed = true
					require.ErrorIs(t, err, test.wantError)
				},
			})

			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/items", nil)
			request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))
			handler.ServeHTTP(response, request)

			require.Equal(t, test.wantInvoked, invoked)
			require.Equal(t, test.wantEncoded, encoded)
			require.Equal(t, test.wantFailure, failed)
			require.Equal(t, test.wantStatus, response.Code)
			require.Len(t, recorder.events, 2)
			require.Equal(t, test.wantReason, recorder.events[1].Reason)
		})
	}
}
