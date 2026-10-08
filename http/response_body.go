package http

import (
	"bytes"
	"errors"
	"io"
	"net/http"
)

// DecodeResponse owns a response body while decode interprets its status,
// headers and body. It closes the original body exactly once and joins cleanup
// failures with decoding failures. Any failure returns a nil result.
//
// restoreBody buffers a bounded body before decoding and restores the captured
// bytes afterwards, including partial bytes on read failure. streamBody marks
// a successful raw-body response: decode must inspect only its metadata. If
// decode succeeds the original body stays open and caller-owned, independently
// of restoreBody; if metadata validation fails it is closed.
func DecodeResponse(resp *http.Response, restoreBody, streamBody bool, decode func(*http.Response) (any, error)) (result any, err error) {
	original := resp.Body
	transferred := false
	defer func() {
		if !transferred && original != nil {
			if closeErr := original.Close(); closeErr != nil {
				err = errors.Join(err, closeErr)
			}
		}
		if err != nil {
			result = nil
		}
	}()
	if restoreBody && !streamBody {
		body, readErr := ReadResponseBody(resp)
		resp.Body = io.NopCloser(bytes.NewReader(body))
		defer func() {
			resp.Body = io.NopCloser(bytes.NewReader(body))
		}()
		if readErr != nil {
			return nil, readErr
		}
	}
	result, err = decode(resp)
	transferred = streamBody && err == nil
	return result, err
}
