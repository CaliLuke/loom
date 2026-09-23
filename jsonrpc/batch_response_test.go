package jsonrpc

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

func TestHTTPHandlerBatchFramesOneEntryPerResponse(t *testing.T) {
	const body = `[{"jsonrpc":"2.0","method":"echo","id":1},{"jsonrpc":"2.0","method":"echo"},{"jsonrpc":"2.0","method":"missing","id":"m"}]`
	success := `{"jsonrpc":"2.0","result":{"ok":true},"id":1}`
	internal := `{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":1}`
	missing := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":"m"}`
	tests := []struct {
		name          string
		dispatch      func(context.Context, http.ResponseWriter) error
		handleFailure func(context.Context, http.ResponseWriter, error)
		want          string
		wantReason    loomtransport.Reason
	}{
		{
			name: "single encoder write",
			dispatch: func(ctx context.Context, w http.ResponseWriter) error {
				return loomhttp.ResponseEncoder(ctx, w).Encode(MakeSuccessResponse(1.0, map[string]bool{"ok": true}))
			},
			want:       "[" + success + "," + missing + "]",
			wantReason: loomtransport.ReasonUnsupportedMethod,
		},
		{
			name: "response split across writes",
			dispatch: func(_ context.Context, w http.ResponseWriter) error {
				for _, chunk := range []string{`{"jsonrpc":"2.0",`, `"result":{"ok":true},`, `"id":1}`} {
					if _, err := w.Write([]byte(chunk)); err != nil {
						return err
					}
				}
				return nil
			},
			want:       "[" + success + "," + missing + "]",
			wantReason: loomtransport.ReasonUnsupportedMethod,
		},
		{
			name: "failure handler writes JSON-RPC error",
			dispatch: func(context.Context, http.ResponseWriter) error {
				return errors.New("endpoint failed")
			},
			handleFailure: func(ctx context.Context, w http.ResponseWriter, _ error) {
				if err := loomhttp.ResponseEncoder(ctx, w).Encode(MakeErrorResponse(1.0, InternalError, "", nil)); err != nil {
					panic(err)
				}
			},
			want:       "[" + internal + "," + missing + "]",
			wantReason: loomtransport.ReasonHandlerError,
		},
		{
			name: "failure handler writes plain text",
			dispatch: func(context.Context, http.ResponseWriter) error {
				return errors.New("endpoint failed")
			},
			handleFailure: func(_ context.Context, w http.ResponseWriter, err error) {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			},
			want:       "[" + internal + "," + missing + "]",
			wantReason: loomtransport.ReasonHandlerError,
		},
		{
			name: "adapter writes nothing",
			dispatch: func(context.Context, http.ResponseWriter) error {
				return nil
			},
			want:       "[" + internal + "," + missing + "]",
			wantReason: loomtransport.ReasonResponseWriteFailed,
		},
		{
			name: "adapter writes two responses",
			dispatch: func(ctx context.Context, w http.ResponseWriter) error {
				encoder := loomhttp.ResponseEncoder(ctx, w)
				response := MakeSuccessResponse(1.0, map[string]bool{"ok": true})
				if err := encoder.Encode(response); err != nil {
					return err
				}
				return encoder.Encode(response)
			},
			want:       "[" + internal + "," + missing + "]",
			wantReason: loomtransport.ReasonResponseWriteFailed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &httpEventRecorder{}
			handler := NewHTTPHandler(HTTPHandlerSpec{
				Service: "echo",
				Decoder: loomhttp.RequestDecoder,
				Encoder: loomhttp.ResponseEncoder,
				Dispatch: func(ctx context.Context, _ *http.Request, request *RawRequest, w http.ResponseWriter) (bool, error) {
					if request.Method != "echo" {
						return false, nil
					}
					if !request.HasID {
						return true, nil
					}
					return true, test.dispatch(ctx, w)
				},
				HandleFailure: test.handleFailure,
			})
			request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
			request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			require.Equal(t, test.want, response.Body.String())
			require.True(t, jsontext.Value(response.Body.Bytes()).IsValid())
			require.Equal(t, "application/json", response.Header().Get("Content-Type"))
			require.Len(t, recorder.events, 2)
			require.Equal(t, test.wantReason, recorder.events[1].Reason)
		})
	}
}
