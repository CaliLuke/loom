package jsonrpc

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

// TestDecodeNotificationResponse asserts that the HTTP response to a
// notification succeeds when it carries no JSON-RPC response, that an error
// response or an unexpected status is reported, and that the body stays
// readable afterwards.
func TestDecodeNotificationResponse(t *testing.T) {
	cases := []struct {
		Name   string
		Status int
		Body   string
		// Error is the name of the expected loomhttp.ClientError, "rpc" for
		// a JSON-RPC error response, or empty for success.
		Error string
	}{
		{Name: "empty", Status: http.StatusOK},
		{Name: "whitespace", Status: http.StatusOK, Body: " \n\t"},
		{Name: "no content", Status: http.StatusNoContent},
		{Name: "accepted", Status: http.StatusAccepted},
		{Name: "result", Status: http.StatusOK, Body: `{"jsonrpc":"2.0","result":null,"id":null}`},
		{Name: "error response", Status: http.StatusOK, Body: `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":null}`, Error: "rpc"},
		{Name: "malformed", Status: http.StatusOK, Body: "{x", Error: "decoding_error"},
		{Name: "server error", Status: http.StatusInternalServerError, Body: "boom", Error: "invalid_response"},
		{Name: "empty server error", Status: http.StatusBadGateway, Error: "invalid_response"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			resp := &http.Response{StatusCode: c.Status, Body: io.NopCloser(strings.NewReader(c.Body))}
			err := DecodeNotificationResponse("svc", "method", resp)
			rest, rerr := io.ReadAll(resp.Body)
			require.NoError(t, rerr)
			assert.Equal(t, c.Body, string(rest))
			switch c.Error {
			case "":
				assert.NoError(t, err)
			case "rpc":
				var rpcErr *RawErrorResponse
				require.ErrorAs(t, err, &rpcErr)
				assert.Equal(t, int(InvalidRequest), rpcErr.Code)
			default:
				var clientErr *loomhttp.ClientError
				require.ErrorAs(t, err, &clientErr)
				assert.Equal(t, c.Error, clientErr.Name)
				assert.Equal(t, "svc", clientErr.Service)
				assert.Equal(t, "method", clientErr.Method)
			}
		})
	}
}

// TestDecodeNotificationResponseReadError asserts that a failure to read the
// response body is reported.
func TestDecodeNotificationResponseReadError(t *testing.T) {
	boom := errors.New("boom")
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(io.MultiReader(strings.NewReader("{"), errReader{boom}))}
	err := DecodeNotificationResponse("svc", "method", resp)
	require.ErrorIs(t, err, boom)
}

// errReader is a reader that fails with err.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }
