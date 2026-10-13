package transport

import (
	"maps"
	"net/http"
	"slices"
)

// WithResponseHeaderCapture snapshots headers surrounding the first final
// WriteHeader, Write, WriteString or Flush call. The snapshot is available only
// when forwarding leaves the headers unchanged. Later handler mutations do not
// alter the snapshot. An opaque ReadFrom that commits first makes headers
// unavailable: its internal read/write timing cannot establish which headers it
// sent. The underlying writer retains its normal concurrency requirements.
func WithResponseHeaderCapture() ResponseCaptureOption {
	return func(c *ResponseCapture) {
		c.captureHeaders = true
	}
}

// Headers returns a detached first-commit snapshot and whether it is available.
// It returns nil, false without the option, before an observed commit, or when
// forwarding changed headers or ReadFrom committed first. It is safe to call
// concurrently with response writes; it never reads the writer's live Header map.
func (c *ResponseCapture) Headers() (http.Header, bool) {
	header := c.headers.Load()
	if header == nil {
		return nil, false
	}
	return header.Clone(), true
}

func (c *ResponseCapture) headerCandidate(writer http.ResponseWriter) http.Header {
	if !c.captureHeaders || c.status.Load() != 0 {
		return nil
	}
	return writer.Header().Clone()
}

func (c *ResponseCapture) commit(code int, writer http.ResponseWriter, header http.Header) {
	if c.status.CompareAndSwap(0, int32(code)) && header != nil && maps.EqualFunc(header, writer.Header(), slices.Equal[[]string]) {
		c.headers.Store(&header)
	}
}
