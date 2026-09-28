package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBooleanMapJSONCodecs(t *testing.T) {
	const body = `{"false":true,"true":false}`
	want := map[bool]bool{false: true, true: false}
	t.Run("request", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var decoded map[bool]bool
		require.NoError(t, RequestDecoder(request).Decode(&decoded))
		require.Equal(t, want, decoded)
		require.NoError(t, RequestEncoder(request).Encode(want))
		encoded, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		require.JSONEq(t, body, string(encoded))
	})
	t.Run("response", func(t *testing.T) {
		response := httptest.NewRecorder()
		require.NoError(t, ResponseEncoder(context.Background(), response).Encode(want))
		require.JSONEq(t, body, response.Body.String())
		result := response.Result()
		t.Cleanup(func() {
			require.NoError(t, result.Body.Close())
		})
		var decoded map[bool]bool
		require.NoError(t, ResponseDecoder(result).Decode(&decoded))
		require.Equal(t, want, decoded)
	})
}
