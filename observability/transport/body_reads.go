package transport

import (
	"errors"
	"io"
	"net/http"
	"slices"
)

type (
	// BodyReadCapture observes bytes consumed downstream of its wrapped body.
	// Body snapshots are safe during Read and Close. Underlying body Read/Close
	// concurrency requirements remain unchanged.
	BodyReadCapture struct {
		body     bodyCapture
		complete bool
	}

	bodyReadObserver struct {
		io.ReadCloser
		capture *BodyReadCapture
	}
)

// CaptureBodyReads returns a passive body wrapper and its separate capture
// state. Each Read and Close reaches the original body with its returned count
// and error unchanged. No reads, drains or closes occur on construction or
// observation. Nil and http.NoBody are returned unchanged as complete bodies.
// At most limit bytes are retained; exceeding it discards the entire preview
// without limiting downstream bytes. limit must be positive or this panics.
func CaptureBodyReads(body io.ReadCloser, limit int) (io.ReadCloser, *BodyReadCapture) {
	if limit <= 0 {
		panic("body read capture limit must be positive")
	}
	capture := &BodyReadCapture{body: bodyCapture{limit: limit}}
	if body == nil || body == http.NoBody {
		capture.complete = true
		return body, capture
	}
	return &bodyReadObserver{ReadCloser: body, capture: capture}, capture
}

// Body returns detached consumed bytes, whether the preview exceeded its limit,
// and whether EOF was observed (or the original body was nil/http.NoBody).
// Complete describes the read observation, not a Content-Length comparison;
// an error or Close does not imply completion. Snapshots never read the body.
func (c *BodyReadCapture) Body() ([]byte, bool, bool) {
	c.body.mu.Lock()
	defer c.body.mu.Unlock()
	return slices.Clone(c.body.data), c.body.omitted, c.complete
}

func (r *bodyReadObserver) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	r.capture.body.mu.Lock()
	defer r.capture.body.mu.Unlock()
	r.capture.body.recordLocked(p, "", int64(n))
	r.capture.complete = r.capture.complete || errors.Is(err, io.EOF)
	return n, err
}
