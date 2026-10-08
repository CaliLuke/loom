package transport

import (
	"io"
	"net/http"
	"sync/atomic"

	"github.com/felixge/httpsnoop"
)

// ResponseCapture records response status and byte counts without retaining
// payloads. Its accessors are safe to call concurrently with response writes.
// The wrapped ResponseWriter retains its own concurrency requirements.
type ResponseCapture struct {
	status atomic.Int32
	bytes  atomic.Int64
}

// CaptureResponse returns a writer and its separate capture state. The writer
// preserves the optional interfaces supported by httpsnoop and exposes Unwrap
// for http.ResponseController. Informational responses do not commit the final
// status, except 101 (Switching Protocols). Writes and flushes commit an implicit
// 200; a ReadFrom that transfers no bytes does not commit a response.
func CaptureResponse(w http.ResponseWriter) (http.ResponseWriter, *ResponseCapture) {
	c := &ResponseCapture{}
	return httpsnoop.Wrap(w, httpsnoop.Hooks{
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(code int) {
				next(code)
				if code == http.StatusSwitchingProtocols || code >= 200 {
					c.commit(code)
				}
			}
		},
		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(b []byte) (int, error) {
				n, err := next(b)
				c.commit(http.StatusOK)
				c.bytes.Add(int64(n))
				return n, err
			}
		},
		WriteString: func(next httpsnoop.WriteStringFunc) httpsnoop.WriteStringFunc {
			return func(s string) (int, error) {
				n, err := next(s)
				c.commit(http.StatusOK)
				c.bytes.Add(int64(n))
				return n, err
			}
		},
		ReadFrom: func(next httpsnoop.ReadFromFunc) httpsnoop.ReadFromFunc {
			return func(r io.Reader) (int64, error) {
				n, err := next(r)
				if n > 0 {
					c.commit(http.StatusOK)
				}
				c.bytes.Add(n)
				return n, err
			}
		},
		Flush: func(next httpsnoop.FlushFunc) httpsnoop.FlushFunc {
			return func() {
				next()
				c.commit(http.StatusOK)
			}
		},
		FlushError: func(next httpsnoop.FlushErrorFunc) httpsnoop.FlushErrorFunc {
			return func() error {
				err := next()
				c.commit(http.StatusOK)
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

func (c *ResponseCapture) commit(code int) {
	c.status.CompareAndSwap(0, int32(code))
}
