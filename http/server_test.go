package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
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
		commitFirst bool
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
		{name: "pre-commit encode failure", encodeErr: errEncode, wantError: errEncode, wantInvoked: true, wantEncoded: true, wantStatus: http.StatusBadRequest, wantReason: loomtransport.ReasonResponseWriteFailed},
		{name: "post-commit encode failure", encodeErr: errEncode, commitFirst: true, wantError: errEncode, wantInvoked: true, wantEncoded: true, wantFailure: true, wantStatus: http.StatusCreated, wantReason: loomtransport.ReasonResponseWriteFailed},
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
					if test.encodeErr == nil || test.commitFirst {
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

func TestUnaryHandlerRestoresHeadersBeforeEncodingPreCommitFailure(t *testing.T) {
	errEncode := errors.New("encode")
	handler := NewUnaryHandler(UnaryHandlerSpec[struct{}, struct{}]{
		Service: "sessions",
		Method:  "renew",
		Invoke: func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, nil
		},
		EncodeResponse: func(_ context.Context, w http.ResponseWriter, _ struct{}) error {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Add("Set-Cookie", "first=success")
			return errEncode
		},
		EncodeError: func(_ context.Context, w http.ResponseWriter, err error) error {
			require.ErrorIs(t, err, errEncode)
			w.Header().Set("Content-Type", ProblemJSONContentType)
			w.WriteHeader(http.StatusInternalServerError)
			return nil
		},
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/session", nil))

	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Equal(t, ProblemJSONContentType, response.Header().Get("Content-Type"))
	require.Empty(t, response.Header().Values("Set-Cookie"))
}

func TestUnaryHandlerFallsBackWhenErrorEncodingFailsBeforeCommit(t *testing.T) {
	errEndpoint := errors.New("denied")
	errPolicy := errors.New("invalid error response cookie policy")
	encodeCalls := 0
	handler := NewUnaryHandler(UnaryHandlerSpec[struct{}, struct{}]{
		Service: "sessions",
		Method:  "renew",
		Invoke: func(context.Context, struct{}) (struct{}, error) {
			return struct{}{}, errEndpoint
		},
		EncodeResponse: func(context.Context, http.ResponseWriter, struct{}) error {
			return nil
		},
		EncodeError: func(_ context.Context, w http.ResponseWriter, err error) error {
			encodeCalls++
			if encodeCalls == 1 {
				require.ErrorIs(t, err, errEndpoint)
				w.Header().Add("Set-Cookie", "partial=must-not-leak")
				return errPolicy
			}
			require.ErrorIs(t, err, errPolicy)
			w.Header().Set("Content-Type", ProblemJSONContentType)
			w.WriteHeader(http.StatusInternalServerError)
			return nil
		},
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/session", nil))

	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Equal(t, 2, encodeCalls)
	require.Empty(t, response.Header().Values("Set-Cookie"))
	require.Equal(t, ProblemJSONContentType, response.Header().Get("Content-Type"))
}

func TestHandlerLifecycleEncodesPreCommitResponseFailure(t *testing.T) {
	errPolicy := errors.New("invalid success response cookie policy")
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/download", nil)
	lifecycle := NewHandlerLifecycle(response, request, "files", "download")

	succeeded := lifecycle.EncodeResponse(
		func(_ context.Context, w http.ResponseWriter) error {
			w.Header().Add("Set-Cookie", "partial=must-not-leak")
			return errPolicy
		},
		func(_ context.Context, w http.ResponseWriter, err error) error {
			require.ErrorIs(t, err, errPolicy)
			w.Header().Set("Content-Type", ProblemJSONContentType)
			w.WriteHeader(http.StatusInternalServerError)
			return nil
		},
		nil,
	)
	lifecycle.End()

	require.False(t, succeeded)
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Empty(t, response.Header().Values("Set-Cookie"))
	require.Equal(t, ProblemJSONContentType, response.Header().Get("Content-Type"))
}

func TestHandlerLifecycleRoutesEndpointErrorsByCommitState(t *testing.T) {
	errEndpoint := errors.New("endpoint")
	tests := []struct {
		name         string
		committed    bool
		wantEncoded  bool
		wantHandled  bool
		wantHTTPCode int
	}{
		{name: "pre-commit", wantEncoded: true, wantHTTPCode: http.StatusBadRequest},
		{name: "post-commit", committed: true, wantHandled: true, wantHTTPCode: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var encoded, handled bool
			recorder := &unaryEventRecorder{}
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/events", nil)
			request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))
			lifecycle := NewHandlerLifecycle(response, request, "events", "watch")

			lifecycle.HandlerFailed(
				errEndpoint,
				test.committed,
				func(_ context.Context, w http.ResponseWriter, err error) error {
					encoded = true
					require.ErrorIs(t, err, errEndpoint)
					w.WriteHeader(http.StatusBadRequest)
					return nil
				},
				func(_ context.Context, _ http.ResponseWriter, err error) {
					handled = true
					require.ErrorIs(t, err, errEndpoint)
				},
			)
			lifecycle.End()

			require.Equal(t, test.wantEncoded, encoded)
			require.Equal(t, test.wantHandled, handled)
			require.Equal(t, test.wantHTTPCode, response.Code)
			require.Len(t, recorder.events, 2)
			require.Equal(t, loomtransport.ReasonHandlerError, recorder.events[1].Reason)
		})
	}
}

func TestHandlerLifecycleDoesNotEncodeFailedWebSocketUpgradeTwice(t *testing.T) {
	recorder := &unaryEventRecorder{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/events", nil)
	request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))
	lifecycle := NewHandlerLifecycle(response, request, "events", "watch")
	defer lifecycle.End()

	_, upgradeErr := (&websocket.Upgrader{}).Upgrade(lifecycle.Writer(), request, nil)
	require.Error(t, upgradeErr)
	var encoded, handled bool
	lifecycle.HandlerFailed(
		upgradeErr,
		false,
		func(context.Context, http.ResponseWriter, error) error {
			encoded = true
			return nil
		},
		func(_ context.Context, _ http.ResponseWriter, err error) {
			handled = true
			require.ErrorIs(t, err, upgradeErr)
		},
	)

	require.False(t, encoded)
	require.True(t, handled)
	require.Equal(t, http.StatusBadRequest, response.Code)
}
