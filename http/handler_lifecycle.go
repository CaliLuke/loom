package http

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// HandlerLifecycle owns the shared context, observation, and failure routing
	// for one generated HTTP handler invocation.
	HandlerLifecycle struct {
		ctx      context.Context
		writer   http.ResponseWriter
		observer *loomtransport.RequestObserver
		service  string
		method   string
	}

	stagedResponseWriter struct {
		header http.Header
		body   bytes.Buffer
		status int
	}

	deferredStatusWriter struct {
		writer    http.ResponseWriter
		status    int
		committed bool
	}
)

// NewHandlerLifecycle starts the shared runtime lifecycle for a generated HTTP
// handler invocation.
func NewHandlerLifecycle(
	w http.ResponseWriter,
	r *http.Request,
	service string,
	method string,
) *HandlerLifecycle {
	ctx := context.WithValue(r.Context(), AcceptTypeKey, requestAcceptHeader(r))
	ctx = context.WithValue(ctx, loom.MethodKey, method)
	ctx = context.WithValue(ctx, loom.ServiceKey, service)
	observer, writer := loomtransport.BeginHTTPRequest(ctx, w, service, method, r)
	return &HandlerLifecycle{
		ctx:      ctx,
		writer:   writer,
		observer: observer,
		service:  service,
		method:   method,
	}
}

// Context returns the request context enriched with Loom transport metadata.
func (l *HandlerLifecycle) Context() context.Context {
	return l.ctx
}

// Writer returns the response writer wrapped for transport observation.
func (l *HandlerLifecycle) Writer() http.ResponseWriter {
	return l.writer
}

// End completes request observation and preserves panic propagation.
func (l *HandlerLifecycle) End() {
	defer l.observer.End()
	if recovered := recover(); recovered != nil {
		panic(recovered)
	}
}

// DecodeFailed records and writes a request decoding failure.
func (l *HandlerLifecycle) DecodeFailed(
	err error,
	encodeError func(context.Context, http.ResponseWriter, error) error,
	handleFailure func(context.Context, http.ResponseWriter, error),
) {
	l.observer.Fail(loomtransport.ReasonRequestDecodeFailed)
	l.encodeError(err, encodeError, handleFailure)
}

// HandlerFailed records an endpoint failure. When committed is true, the
// transport can no longer encode an HTTP error response and the failure is
// routed directly to handleFailure.
func (l *HandlerLifecycle) HandlerFailed(
	err error,
	committed bool,
	encodeError func(context.Context, http.ResponseWriter, error) error,
	handleFailure func(context.Context, http.ResponseWriter, error),
) {
	l.observer.Fail(loomtransport.ReasonHandlerError)
	if committed || l.responseCommitted() {
		l.handleFailure(err, handleFailure)
		return
	}
	l.encodeError(err, encodeError, handleFailure)
}

// ResponseFailed records a response write or close failure and routes it to
// handleFailure.
func (l *HandlerLifecycle) ResponseFailed(
	err error,
	handleFailure func(context.Context, http.ResponseWriter, error),
) {
	l.observer.Fail(loomtransport.ReasonResponseWriteFailed)
	l.handleFailure(err, handleFailure)
}

// EncodeResponse writes response metadata or a buffered response body. When
// encoding fails before commit, it restores the original headers and encodes
// the failure through the generated error formatter. It reports whether the
// response encoding succeeded.
func (l *HandlerLifecycle) EncodeResponse(
	encodeResponse func(context.Context, http.ResponseWriter) error,
	encodeError func(context.Context, http.ResponseWriter, error) error,
	handleFailure func(context.Context, http.ResponseWriter, error),
) bool {
	initialHeaders := l.writer.Header().Clone()
	if err := encodeResponse(l.ctx, l.writer); err != nil {
		l.observer.Fail(loomtransport.ReasonResponseWriteFailed)
		if l.responseCommitted() {
			l.handleFailure(err, handleFailure)
			return false
		}
		replaceHeaders(l.writer.Header(), initialHeaders)
		l.encodeError(err, encodeError, handleFailure)
		return false
	}
	return true
}

// WriteRawBody encodes response metadata, streams body, closes it, and applies
// the generated HTTP late-error policy.
func (l *HandlerLifecycle) WriteRawBody(
	body io.ReadCloser,
	encodeResponse func(context.Context, http.ResponseWriter) error,
	encodeError func(context.Context, http.ResponseWriter, error) error,
	handleFailure func(context.Context, http.ResponseWriter, error),
) {
	defer func() {
		if err := body.Close(); err != nil {
			l.ResponseFailed(err, handleFailure)
		}
	}()

	var writeBody func(io.Writer) (int64, error)
	if writerTo, ok := body.(io.WriterTo); ok {
		writeBody = writerTo.WriteTo
	} else {
		buffered := bufio.NewReader(body)
		if _, err := buffered.Peek(1); err != nil && err != io.EOF {
			l.HandlerFailed(err, false, encodeError, handleFailure)
			return
		}
		writeBody = func(writer io.Writer) (int64, error) {
			return io.Copy(writer, buffered)
		}
	}

	initialHeaders := l.writer.Header().Clone()
	staged := &stagedResponseWriter{header: initialHeaders.Clone()}
	if err := encodeResponse(l.ctx, staged); err != nil {
		l.observer.Fail(loomtransport.ReasonResponseWriteFailed)
		l.encodeError(err, encodeError, handleFailure)
		return
	}
	replaceHeaders(l.writer.Header(), staged.header)
	writer := &deferredStatusWriter{writer: l.writer, status: staged.status}
	if staged.body.Len() > 0 {
		if _, err := writer.Write(staged.body.Bytes()); err != nil {
			l.observer.Fail(loomtransport.ReasonResponseWriteFailed)
			l.abortLateWrite(handleFailure)
			return
		}
	}
	if _, err := writeBody(writer); err != nil {
		l.observer.Fail(loomtransport.ReasonResponseWriteFailed)
		if !l.responseCommitted() {
			replaceHeaders(l.writer.Header(), initialHeaders)
			l.encodeError(err, encodeError, handleFailure)
			return
		}
		l.abortLateWrite(handleFailure)
		return
	}
	writer.commit()
}

// ServeFile encodes response metadata, applies the designed content type,
// serves file with standard HTTP semantics, and closes seekable content when
// it also implements io.Closer.
func (l *HandlerLifecycle) ServeFile(
	r *http.Request,
	file *FileResponse,
	contentType string,
	encodeResponse func(context.Context, http.ResponseWriter) error,
	encodeError func(context.Context, http.ResponseWriter, error) error,
	handleFailure func(context.Context, http.ResponseWriter, error),
) {
	if file == nil || file.Content == nil {
		err := fmt.Errorf("%s.%s returned nil file response content", l.service, l.method)
		l.HandlerFailed(err, false, encodeError, handleFailure)
		return
	}
	if closer, ok := file.Content.(io.Closer); ok {
		defer func() {
			if err := closer.Close(); err != nil {
				l.ResponseFailed(err, handleFailure)
			}
		}()
	}
	if !l.EncodeResponse(encodeResponse, encodeError, handleFailure) {
		return
	}
	if contentType != "" {
		l.writer.Header().Set("Content-Type", contentType)
	}
	file.ServeHTTP(l.writer, r)
}

func (l *HandlerLifecycle) encodeError(
	err error,
	encode func(context.Context, http.ResponseWriter, error) error,
	handleFailure func(context.Context, http.ResponseWriter, error),
) {
	encodeErrorWithFallback(l.ctx, l.writer, err, encode, handleFailure)
}

func (l *HandlerLifecycle) handleFailure(
	err error,
	handleFailure func(context.Context, http.ResponseWriter, error),
) {
	if handleFailure != nil {
		handleFailure(l.ctx, l.writer, err)
	}
}

func (l *HandlerLifecycle) abortLateWrite(
	handleFailure func(context.Context, http.ResponseWriter, error),
) {
	if err := http.NewResponseController(l.writer).Flush(); err != nil {
		l.handleFailure(err, handleFailure)
	}
	panic(http.ErrAbortHandler)
}

func (l *HandlerLifecycle) responseCommitted() bool {
	capture, ok := l.writer.(interface{ StatusCode() int })
	return ok && capture.StatusCode() != 0
}

func (w *stagedResponseWriter) Header() http.Header {
	return w.header
}

func (w *stagedResponseWriter) WriteHeader(status int) {
	if status < http.StatusOK || w.status != 0 {
		return
	}
	w.status = status
}

func (w *stagedResponseWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *deferredStatusWriter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if w.status != 0 {
		w.commit()
	}
	written, err := w.writer.Write(data)
	w.committed = true
	return written, err
}

func (w *deferredStatusWriter) commit() {
	if w.committed {
		return
	}
	w.committed = true
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	w.writer.WriteHeader(status)
}
