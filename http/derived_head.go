package http

import (
	"bufio"
	"io"
	"net/http"
	"strconv"
)

type derivedHeadResponseWriter struct {
	header          http.Header
	committedHeader http.Header
	status          int
	length          int64
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
			writer.commit(http.StatusOK)
			status = writer.status
		}
		if writer.committedHeader.Get("Content-Length") == "" && writer.length > 0 && headStatusAllowsBody(status) {
			writer.committedHeader.Set("Content-Length", strconv.FormatInt(writer.length, 10))
		}
		replaceHeaders(w.Header(), writer.committedHeader)
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
	if status < http.StatusOK || w.status != 0 {
		return
	}
	w.commit(status)
}

func (w *derivedHeadResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.sniffContentType(body)
		w.commit(http.StatusOK)
	}
	w.length += int64(len(body))
	return len(body), nil
}

func (w *derivedHeadResponseWriter) ReadFrom(reader io.Reader) (int64, error) {
	buffered := bufio.NewReaderSize(reader, 512)
	preview, _ := buffered.Peek(512)
	if w.status == 0 && len(preview) > 0 {
		w.sniffContentType(preview)
		w.commit(http.StatusOK)
	}
	written, err := io.Copy(io.Discard, buffered)
	w.length += written
	return written, err
}

func (w *derivedHeadResponseWriter) commit(status int) {
	w.status = status
	w.committedHeader = w.header.Clone()
}

func (w *derivedHeadResponseWriter) sniffContentType(body []byte) {
	if _, set := w.header["Content-Type"]; set {
		return
	}
	if _, set := w.header["Content-Encoding"]; set {
		return
	}
	w.header.Set("Content-Type", http.DetectContentType(body))
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
