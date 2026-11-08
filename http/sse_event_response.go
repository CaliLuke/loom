package http

import (
	"io"
	"net/http"
	"strings"
)

// SSEEventResponse returns an independent JSON response for one event's data.
// It retains the stream status for diagnostics and excludes the initiating
// request so event decoding does not inherit request media or nullability policy.
func SSEEventResponse(stream *http.Response, data string) *http.Response {
	return &http.Response{
		StatusCode: stream.StatusCode,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(data)),
	}
}
