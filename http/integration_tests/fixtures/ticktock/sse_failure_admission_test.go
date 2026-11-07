package ticktock_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/http-ticktock/gen/clock"
	clockserver "example.com/http-ticktock/gen/http/clock/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type failureAdmissionEncoder struct {
	encode func(any) error
}

func TestGeneratedSSEHandlerFailureClaimsAdmission(t *testing.T) {
	for _, operation := range []string{"comment", "typed send"} {
		t.Run(operation, func(t *testing.T) {
			var retained clock.GuardedServerStream
			var retainedErr error
			var prefix string
			called, encoded, handled := 0, 0, 0
			rec := httptest.NewRecorder()
			mux := loomhttp.NewMuxer()
			handler := clockserver.NewGuardedHandler(func(_ context.Context, value any) (any, error) {
				called++
				retained = value.(*clock.GuardedEndpointInput).Stream
				return nil, loom.PermanentError("unauthorized", "denied")
			}, mux, loomhttp.RequestDecoder, func(ctx context.Context, w http.ResponseWriter) loomhttp.Encoder {
				delegate := loomhttp.ResponseEncoder(ctx, w)
				return failureAdmissionEncoder{encode: func(value any) error {
					encoded++
					if err := delegate.Encode(value); err != nil {
						return err
					}
					prefix = rec.Body.String()
					// The endpoint returned, but the generated HTTP handler still owns the
					// response: its error encoder has not returned and End has not run.
					if operation == "comment" {
						retainedErr = retained.(loomhttp.SSEControl).SendComment(ctx, "late")
					} else {
						retainedErr = retained.Send(&clock.TickTockEvent{Event: "late", Data: "late"})
					}
					return nil
				}}
			}, func(context.Context, http.ResponseWriter, error) {
				handled++
			}, nil)
			clockserver.MountGuardedHandler(mux, handler)
			req := httptest.NewRequest(http.MethodGet, "/guarded", nil)
			req.Header.Set("Accept", "text/event-stream")
			mux.ServeHTTP(rec, req)
			require.Equal(t, 1, called)
			require.Equal(t, 1, encoded)
			require.Zero(t, handled)
			require.ErrorIs(t, retainedErr, loomhttp.ErrSSEStreamClosed)
			require.Equal(t, http.StatusUnauthorized, rec.Code)
			require.Contains(t, rec.Header().Get("Content-Type"), "application/problem+json")
			require.Contains(t, prefix, `"detail":"denied"`)
			require.Contains(t, prefix, `"code":"unauthorized"`)
			require.Equal(t, prefix, rec.Body.String())
			require.False(t, rec.Flushed)
			require.Empty(t, rec.Header().Get("X-Accel-Buffering"))
		})
	}
}

func (e failureAdmissionEncoder) Encode(value any) error {
	return e.encode(value)
}
