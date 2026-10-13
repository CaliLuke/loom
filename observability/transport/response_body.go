package transport

import (
	"io"
	"slices"
	"sync"
)

type (
	// ResponseCaptureOption configures optional response observation before any
	// writes occur. CaptureResponse retains no body unless an option enables it.
	ResponseCaptureOption func(*ResponseCapture)

	responseBodyCapture struct {
		mu      sync.Mutex
		limit   int
		data    []byte
		omitted bool
	}
	responseCaptureReader struct {
		io.Reader
		limit int
		data  []byte
	}
)

// WithResponseBodyCapture retains at most limit accepted body bytes. If the
// response exceeds limit, the entire preview is discarded and Body reports
// omission. The complete response still reaches the underlying writer. limit
// must be positive; otherwise this function panics. Hijacked I/O is not captured.
func WithResponseBodyCapture(limit int) ResponseCaptureOption {
	if limit <= 0 {
		panic("response body capture limit must be positive")
	}
	return func(c *ResponseCapture) {
		c.body = &responseBodyCapture{limit: limit}
	}
}

// Body returns a detached copy of accepted response bytes and whether capture
// was omitted after exceeding its configured limit. Without body capture it
// returns nil, false. It is safe to call concurrently with response writes.
func (c *ResponseCapture) Body() ([]byte, bool) {
	if c.body == nil {
		return nil, false
	}
	c.body.mu.Lock()
	defer c.body.mu.Unlock()
	return slices.Clone(c.body.data), c.body.omitted
}

func (b *responseBodyCapture) record(data []byte, text string, n int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.omitted {
		return
	}
	if n > int64(b.limit-len(b.data)) {
		b.data = nil
		b.omitted = true
		return
	}
	if data != nil {
		b.data = append(b.data, data[:int(n)]...)
	} else {
		b.data = append(b.data, text[:int(n)]...)
	}
}

func (b *responseBodyCapture) reader(r io.Reader) *responseCaptureReader {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.omitted {
		return nil
	}
	return &responseCaptureReader{Reader: r, limit: b.limit - len(b.data)}
}

func (r *responseCaptureReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	retained := min(n, r.limit-len(r.data))
	r.data = append(r.data, p[:retained]...)
	return n, err
}
