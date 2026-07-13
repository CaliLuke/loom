package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

type (
	// SSEControl exposes transport lifecycle operations implemented by generated
	// server SSE streams. Endpoint implementations may type assert their stream
	// to this interface without depending on generated concrete types.
	SSEControl interface {
		Open(context.Context) error
		SendComment(context.Context, string) error
	}

	// SSEStreamWriter owns serialized writes for a generated server SSE stream.
	SSEStreamWriter struct {
		w          http.ResponseWriter
		requestCtx context.Context
		transport  loomtransport.TransportKind
		policy     StreamWritePolicy

		lock    sync.Mutex
		started bool
		closed  bool
		openErr error
	}
)

var (
	// ErrSSEStreamClosed is returned when a control or event write is attempted
	// after the generated SSE stream has closed.
	ErrSSEStreamClosed = errors.New("loom http SSE stream closed")
	// ErrInvalidSSEComment is returned when comment text contains a line break
	// that could inject another SSE field or event.
	ErrInvalidSSEComment = errors.New("loom http SSE comment contains a line break")
)

// NewSSEStreamWriter returns a shared writer for a generated server SSE stream.
func NewSSEStreamWriter(
	w http.ResponseWriter,
	requestCtx context.Context,
	transport loomtransport.TransportKind,
	policy StreamWritePolicy,
) *SSEStreamWriter {
	return &SSEStreamWriter{
		w:          w,
		requestCtx: requestCtx,
		transport:  transport,
		policy:     policy,
	}
}

// Context returns the inbound request context associated with the stream.
func (s *SSEStreamWriter) Context() context.Context {
	return s.requestCtx
}

// Open commits and flushes the successful SSE response once.
func (s *SSEStreamWriter) Open(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	if s.closed {
		return ErrSSEStreamClosed
	}
	if s.started {
		return s.openErr
	}
	s.initHeaders()
	s.openErr = s.flush(ctx)
	return s.openErr
}

// SendComment writes and flushes one SSE comment frame.
func (s *SSEStreamWriter) SendComment(ctx context.Context, text string) error {
	if strings.ContainsAny(text, "\r\n") {
		return ErrInvalidSSEComment
	}
	return s.WriteEvent(ctx, func(w io.Writer) error {
		_, err := fmt.Fprintf(w, ": %s\n\n", text)
		return err
	})
}

// WriteEvent serializes and flushes one generated SSE event.
func (s *SSEStreamWriter) WriteEvent(ctx context.Context, write func(io.Writer) error) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	if s.closed {
		return ErrSSEStreamClosed
	}
	s.initHeaders()
	if err := s.runOperation(ctx, func(http.ResponseWriter) error {
		return write(s.w)
	}); err != nil {
		s.observeFailure(ctx, loomtransport.ReasonStreamWriteFailed, loomtransport.ReasonStreamWriteTimeout, err)
		return err
	}
	return s.flush(ctx)
}

// Started reports whether the successful SSE response has been committed.
func (s *SSEStreamWriter) Started() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.started
}

// Close prevents later control and event writes.
func (s *SSEStreamWriter) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.closed = true
	return nil
}

func (s *SSEStreamWriter) checkContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.requestCtx.Err(); err != nil {
		return err
	}
	return nil
}

func (s *SSEStreamWriter) initHeaders() {
	header := s.w.Header()
	setHeaderDefault(header, "Content-Type", "text/event-stream")
	setHeaderDefault(header, "Cache-Control", "no-cache")
	setHeaderDefault(header, "Connection", "keep-alive")
	setHeaderDefault(header, "X-Accel-Buffering", "no")
	s.w.WriteHeader(http.StatusOK)
	s.started = true
}

func (s *SSEStreamWriter) flush(ctx context.Context) error {
	err := s.runOperation(ctx, func(http.ResponseWriter) error {
		return http.NewResponseController(s.w).Flush()
	})
	if err != nil {
		s.observeFailure(ctx, loomtransport.ReasonStreamFlushFailed, loomtransport.ReasonStreamFlushTimeout, err)
	}
	return err
}

func (s *SSEStreamWriter) runOperation(ctx context.Context, operation func(http.ResponseWriter) error) error {
	controller := http.NewResponseController(s.w)
	deadline, bounded := s.operationDeadline(ctx)
	if bounded {
		if err := controller.SetWriteDeadline(deadline); err != nil {
			return fmt.Errorf("%w: %w", ErrStreamWriteDeadlineUnsupported, err)
		}
	}
	cancelDeadline := make(chan error, 1)
	stop := context.AfterFunc(ctx, func() {
		cancelDeadline <- controller.SetWriteDeadline(time.Now())
	})
	err := operation(s.w)
	if !stop() {
		if deadlineErr := <-cancelDeadline; deadlineErr != nil && err == nil {
			err = deadlineErr
		}
	}
	if bounded {
		if clearErr := controller.SetWriteDeadline(time.Time{}); clearErr != nil && err == nil {
			return clearErr
		}
	}
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return err
}

func (s *SSEStreamWriter) operationDeadline(ctx context.Context) (time.Time, bool) {
	return streamOperationDeadline(ctx, s.policy)
}

func (s *SSEStreamWriter) observeFailure(ctx context.Context, failed, timeout loomtransport.Reason, err error) {
	reason := failed
	if isTimeoutError(err) {
		reason = timeout
	}
	loomtransport.Observe(ctx, loomtransport.Event{
		Kind:      loomtransport.EventKindStreamFailure,
		Reason:    reason,
		Transport: s.transport,
	})
}

func setHeaderDefault(header http.Header, name, value string) {
	if header.Get(name) == "" {
		header.Set(name, value)
	}
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var timeout interface{ Timeout() bool }
	return errors.As(err, &timeout) && timeout.Timeout()
}
