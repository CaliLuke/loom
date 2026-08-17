package http

import (
	"bufio"
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
)

// NewHandlerLifecycle starts the shared runtime lifecycle for a generated HTTP
// handler invocation.
func NewHandlerLifecycle(
	w http.ResponseWriter,
	r *http.Request,
	service string,
	method string,
) *HandlerLifecycle {
	ctx := context.WithValue(r.Context(), AcceptTypeKey, r.Header.Get("Accept"))
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

	if writerTo, ok := body.(io.WriterTo); ok {
		if err := encodeResponse(l.ctx, l.writer); err != nil {
			l.ResponseFailed(err, handleFailure)
			return
		}
		_, err := writerTo.WriteTo(l.writer)
		if err == nil {
			return
		}
		l.observer.Fail(loomtransport.ReasonResponseWriteFailed)
		if !l.responseCommitted() {
			l.encodeError(err, encodeError, handleFailure)
			return
		}
		l.abortLateWrite(handleFailure)
		return
	}

	buffered := bufio.NewReader(body)
	if _, err := buffered.Peek(1); err != nil && err != io.EOF {
		l.HandlerFailed(err, false, encodeError, handleFailure)
		return
	}
	if err := encodeResponse(l.ctx, l.writer); err != nil {
		l.ResponseFailed(err, handleFailure)
		return
	}
	if _, err := io.Copy(l.writer, buffered); err != nil {
		l.observer.Fail(loomtransport.ReasonResponseWriteFailed)
		l.abortLateWrite(handleFailure)
	}
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
	if err := encodeResponse(l.ctx, l.writer); err != nil {
		l.ResponseFailed(err, handleFailure)
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
	if encodeErr := encode(l.ctx, l.writer, err); encodeErr != nil {
		l.handleFailure(encodeErr, handleFailure)
	}
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
