package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSSEEventResponse(t *testing.T) {
	ctx := context.WithValue(context.Background(), mediaContractKey{}, true)
	ctx = context.WithValue(ctx, nonNullableBodyKey{}, true)
	stream := &http.Response{StatusCode: 201, Header: http.Header{"Content-Type": {"text/event-stream"}, "X-Handshake": {"original"}}, Request: httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)}
	for _, data := range []string{`{"message":"hello"}`, `null`} {
		t.Run(data, func(t *testing.T) {
			event := SSEEventResponse(stream, data)
			require.NotSame(t, stream, event)
			require.Equal(t, 201, event.StatusCode)
			require.Nil(t, event.Request)
			require.Equal(t, http.Header{"Content-Type": {"application/json"}}, event.Header)
			require.NotNil(t, event.Body)
			body, err := io.ReadAll(event.Body)
			require.NoError(t, err)
			require.Equal(t, data, string(body))
			require.NoError(t, event.Body.Close())
			event = SSEEventResponse(stream, data)
			var decoded map[string]string
			require.NoError(t, ResponseDecoder(event).Decode(&decoded))
			require.NoError(t, event.Body.Close())
			if data == "null" {
				require.Nil(t, decoded)
			} else {
				require.Equal(t, "hello", decoded["message"])
			}
			event.Header.Set("X-Handshake", "mutated")
			require.Equal(t, "original", stream.Header.Get("X-Handshake"))
		})
	}
}
