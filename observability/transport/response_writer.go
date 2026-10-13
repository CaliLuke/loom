package transport

import (
	"io"
	"net/http"
	"sync/atomic"

	"github.com/felixge/httpsnoop"
)

// ResponseCapture records response status, byte counts and an optional bounded
// body preview. Its accessors are safe to call concurrently with response writes.
// The wrapped ResponseWriter retains its own concurrency requirements.
type ResponseCapture struct {
	status         atomic.Int32
	bytes          atomic.Int64
	body           *bodyCapture
	captureHeaders bool
	headers        atomic.Pointer[http.Header]
}

// CaptureResponse returns a writer and its separate capture state. The writer
// preserves the optional interfaces supported by httpsnoop and exposes Unwrap
// for http.ResponseController. Informational responses do not commit the final
// status, except 101 (Switching Protocols). Writes and flushes commit an implicit
// 200; a ReadFrom that transfers no bytes does not commit a response. By default
// no body bytes or headers are retained; options enable bounded body capture
// and first-observed-commit header snapshots. Opaque ReadFrom commits cannot
// establish header timing unless an earlier operation already committed.
func CaptureResponse(w http.ResponseWriter, options ...ResponseCaptureOption) (http.ResponseWriter, *ResponseCapture) {
	c := &ResponseCapture{}
	for _, option := range options {
		option(c)
	}
	return httpsnoop.Wrap(w, httpsnoop.Hooks{
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(code int) {
				header := c.headerCandidate(w)
				next(code)
				if code == http.StatusSwitchingProtocols || code >= 200 {
					c.commit(code, w, header)
				}
			}
		},
		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(b []byte) (int, error) {
				header := c.headerCandidate(w)
				n, err := next(b)
				c.commit(http.StatusOK, w, header)
				c.bytes.Add(int64(n))
				if c.body != nil {
					c.body.record(b, "", int64(n))
				}
				return n, err
			}
		},
		WriteString: func(next httpsnoop.WriteStringFunc) httpsnoop.WriteStringFunc {
			return func(s string) (int, error) {
				header := c.headerCandidate(w)
				n, err := next(s)
				c.commit(http.StatusOK, w, header)
				c.bytes.Add(int64(n))
				if c.body != nil {
					c.body.record(nil, s, int64(n))
				}
				return n, err
			}
		},
		ReadFrom: c.readFromHook,
		Flush: func(next httpsnoop.FlushFunc) httpsnoop.FlushFunc {
			return func() {
				header := c.headerCandidate(w)
				next()
				c.commit(http.StatusOK, w, header)
			}
		},
		FlushError: func(next httpsnoop.FlushErrorFunc) httpsnoop.FlushErrorFunc {
			return func() error {
				header := c.headerCandidate(w)
				err := next()
				c.commit(http.StatusOK, w, header)
				return err
			}
		},
	}), c
}

// StatusCode returns the first final response status or zero before any
// response is committed. Hijacked connection writes are outside the capture.
func (c *ResponseCapture) StatusCode() int {
	return int(c.status.Load())
}

// BytesWritten returns the actual bytes transferred through Write, WriteString
// and ReadFrom, including partial writes that return an error.
func (c *ResponseCapture) BytesWritten() int64 {
	return c.bytes.Load()
}

func (c *ResponseCapture) readFromHook(next httpsnoop.ReadFromFunc) httpsnoop.ReadFromFunc {
	return func(r io.Reader) (int64, error) {
		var preview *responseCaptureReader
		if c.body != nil {
			preview = c.body.reader(r)
			if preview != nil {
				r = preview
			}
		}
		n, err := next(r)
		if n > 0 {
			c.commit(http.StatusOK, nil, nil)
		}
		c.bytes.Add(n)
		if preview != nil {
			c.body.record(preview.data, "", n)
		}
		return n, err
	}
}
