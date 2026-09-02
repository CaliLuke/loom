package http

import (
	"io"
	"net/http"
	"strconv"
)

type derivedHeadResponseWriter struct {
	header http.Header
	status int
	length int64
}

// DerivedHeadHandler adapts an ordinary unary GET handler to HEAD. It executes
// the complete handler so headers, cookies, authentication, and application
// effects stay aligned with GET, while counting and suppressing the body.
func DerivedHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := &derivedHeadResponseWriter{header: w.Header().Clone()}
		next.ServeHTTP(writer, r)
		status := writer.status
		if status == 0 {
			status = http.StatusOK
		}
		if writer.header.Get("Content-Length") == "" && writer.length > 0 && headStatusAllowsBody(status) {
			writer.header.Set("Content-Length", strconv.FormatInt(writer.length, 10))
		}
		replaceHeaders(w.Header(), writer.header)
		w.WriteHeader(status)
	})
}

// MountDerivedHead mounts a HEAD companion for an ordinary unary GET handler.
// Use an explicitly designed HEAD route for files and streaming responses.
func MountDerivedHead(mux Muxer, pattern string, getHandler http.Handler) {
	MountHandler(mux, http.MethodHead, pattern, DerivedHeadHandler(getHandler))
}

func (w *derivedHeadResponseWriter) Header() http.Header {
	return w.header
}

func (w *derivedHeadResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
}

func (w *derivedHeadResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.length += int64(len(body))
	return len(body), nil
}

func (w *derivedHeadResponseWriter) ReadFrom(reader io.Reader) (int64, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	written, err := io.Copy(io.Discard, reader)
	w.length += written
	return written, err
}

func replaceHeaders(destination, source http.Header) {
	for key := range destination {
		destination.Del(key)
	}
	for key, values := range source {
		destination[key] = append([]string(nil), values...)
	}
}

func headStatusAllowsBody(status int) bool {
	return status >= http.StatusOK &&
		status != http.StatusNoContent &&
		status != http.StatusResetContent &&
		status != http.StatusNotModified
}
