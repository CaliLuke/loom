package transport

import (
	"bufio"
	"net"
	"net/http"
)

// CaptureResponseWriter wraps a [http.ResponseWriter] and records the final
// HTTP status code and number of bytes written by the handler. Generated
// HTTP handlers use it to populate the StatusCode and BytesWritten fields of
// terminal transport events without retaining the raw response body.
//
// CaptureResponseWriter forwards [http.Hijacker] directly. Its Unwrap method
// lets [http.ResponseController] reach other optional writer interfaces.
type CaptureResponseWriter struct {
	// ResponseWriter is the wrapped writer. It is intentionally exported so
	// generated code can pass the wrapper directly to helpers that perform
	// type assertions (for example, hijack and flush probes).
	ResponseWriter http.ResponseWriter

	status int
	bytes  int64
}

// NewCaptureResponseWriter returns a wrapper that records status and byte
// counts for w.
func NewCaptureResponseWriter(w http.ResponseWriter) *CaptureResponseWriter {
	return &CaptureResponseWriter{ResponseWriter: w}
}

// Header returns the wrapped writer's header map.
func (c *CaptureResponseWriter) Header() http.Header {
	return c.ResponseWriter.Header()
}

// WriteHeader records the first status code observed and forwards the call
// to the wrapped writer. Subsequent calls are forwarded but only the first
// status is retained, matching net/http behavior.
func (c *CaptureResponseWriter) WriteHeader(status int) {
	if c.status == 0 {
		c.status = status
	}
	c.ResponseWriter.WriteHeader(status)
}

// Write writes b to the wrapped writer and accumulates the byte count. If
// no status has been written yet, an implicit 200 is recorded to match
// net/http semantics.
func (c *CaptureResponseWriter) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	n, err := c.ResponseWriter.Write(b)
	c.bytes += int64(n)
	return n, err
}

// Hijack forwards ownership of the HTTP connection when the wrapped writer
// supports connection hijacking.
func (c *CaptureResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(c.ResponseWriter).Hijack()
}

// StatusCode returns the HTTP status code recorded by WriteHeader or Write,
// or 0 if the handler wrote no response.
func (c *CaptureResponseWriter) StatusCode() int {
	return c.status
}

// BytesWritten returns the number of bytes the handler has written to the
// underlying response writer through Write.
func (c *CaptureResponseWriter) BytesWritten() int64 {
	return c.bytes
}

// Unwrap returns the wrapped [http.ResponseWriter] so that
// [http.ResponseController] can locate underlying interfaces such as
// Flusher and Hijacker.
func (c *CaptureResponseWriter) Unwrap() http.ResponseWriter {
	return c.ResponseWriter
}
