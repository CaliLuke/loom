package http

import (
	"bytes"
	"context"
	"errors"
	"net/http"
)

type nonNullableBodyKey struct{}

var (
	errNullRequestBody  = errors.New("JSON null is not allowed for this request body")
	errNullResponseBody = errors.New("JSON null is not allowed for this response body")
)

// WithNonNullableBody returns a shallow request copy whose built-in JSON
// RequestDecoder rejects a top-level null. Generated decoders use it when the
// DSL body is not nullable. It preserves all other decoding behavior, including
// absent bodies, null object fields, and non-JSON content types. Custom decoder
// factories remain responsible for enforcing their body contract.
func WithNonNullableBody(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), nonNullableBodyKey{}, true))
}

// WithNonNullableResponseBody returns a shallow response copy whose request
// context makes the built-in JSON ResponseDecoder reject a top-level null.
// Generated clients use it when the DSL response body is not nullable. The
// caller's response and request are unchanged; a missing request is replaced
// by an empty one in the copy. Custom decoder factories remain responsible for
// enforcing their body contract.
func WithNonNullableResponseBody(response *http.Response) *http.Response {
	copy := *response
	request := response.Request
	if request == nil {
		request = new(http.Request)
	}
	copy.Request = WithNonNullableBody(request)
	return &copy
}

func decodeNonNullableJSON(data []byte, v any) error {
	if bytes.Equal(bytes.Trim(data, " \t\r\n"), []byte("null")) {
		return errNullRequestBody
	}
	return decodeJSON(data, v)
}
