package http

import (
	"bytes"
	"context"
	"errors"
	"net/http"
)

type nonNullableBodyKey struct{}

var errNullRequestBody = errors.New("JSON null is not allowed for this request body")

// WithNonNullableBody returns a shallow request copy whose built-in JSON
// RequestDecoder rejects a top-level null. Generated decoders use it when the
// DSL body is not nullable. It preserves all other decoding behavior, including
// absent bodies, null object fields, and non-JSON content types. Custom decoder
// factories remain responsible for enforcing their body contract.
func WithNonNullableBody(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), nonNullableBodyKey{}, true))
}

func decodeNonNullableJSON(data []byte, v any) error {
	if bytes.Equal(bytes.Trim(data, " \t\r\n"), []byte("null")) {
		return errNullRequestBody
	}
	return decodeJSON(data, v)
}
