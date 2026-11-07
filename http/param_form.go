package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type paramFormBody struct {
	io.Reader
	io.Closer
}

// ParseRawFormWithLimit reads the form body once under maxBytes and initializes
// PostForm from the same bytes, preserving raw delimiters for styled fields.
// The returned carrier also retains whether any body bytes were read, including
// nonempty input that decodes to no fields.
func ParseRawFormWithLimit(req *http.Request, maxBytes int64) (RawPairs, error) {
	if req == nil || maxBytes <= 0 {
		return RawPairs{}, fmt.Errorf("raw form requires a request and a positive body limit")
	}
	if req.PostForm != nil {
		return RawPairs{}, fmt.Errorf("raw form requires an unread form body")
	}
	var captured bytes.Buffer
	if req.Body != nil {
		req.Body = &paramFormBody{Reader: io.TeeReader(req.Body, &captured), Closer: req.Body}
	}
	if err := ParseFormWithLimit(req, maxBytes); err != nil {
		return RawPairs{}, err
	}
	return ParseRawPairs(captured.String())
}

// SetRawFormRequest publishes a pre-encoded form with exact length and replay.
func SetRawFormRequest(req *http.Request, encoded string) error {
	if req == nil {
		return fmt.Errorf("form request cannot be nil")
	}
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = io.NopCloser(strings.NewReader(encoded))
	req.ContentLength = int64(len(encoded))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(encoded)), nil
	}
	return nil
}
