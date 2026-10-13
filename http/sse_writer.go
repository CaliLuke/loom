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
		w           http.ResponseWriter
		requestCtx  context.Context
		observation *loomtransport.StreamObserver
		policy      StreamWritePolicy

		lock    sync.Mutex
		started bool
		closed  bool
		openErr error
		failErr error
	}

	// sseFrameWriter tracks whether an event callback touched the response
	// body, so a callback error that emitted nothing leaves the stream usable.
	sseFrameWriter struct {
		w       io.Writer
		touched bool
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
		w:           w,
		requestCtx:  requestCtx,
		observation: loomtransport.NewStreamObserver(requestCtx, transport),
		policy:      policy,
	}
}

// Context returns the inbound request context associated with the stream.
func (s *SSEStreamWriter) Context() context.Context {
	return s.requestCtx
}

// Open commits and flushes the successful SSE response once.
func (s *SSEStreamWriter) Open(ctx context.Context) error {
	var reason loomtransport.Reason
	s.lock.Lock()
	defer s.finishObservation(&reason)
	if err := s.checkContext(ctx); err != nil {
		return err
	}
	if err := s.closedErr(); err != nil {
		return err
	}
	if s.started {
		return s.openErr
	}
	s.initHeaders()
	s.openErr, reason = s.flush(ctx)
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

// WriteEvent serializes and flushes one generated SSE event. The stream becomes
// terminal when the callback fails after writing any bytes, when a write to the
// response itself fails, or when the flush fails: later calls return
// ErrSSEStreamClosed wrapping the original failure and write nothing, so a
// partially written frame is never followed by another event. A callback error
// returned before any byte was written, such as an encoding failure, is
// returned as is and leaves the stream usable. Cancellation observed before an
// operation starts does not end the stream. If ctx is canceled after the frame
// was serialized, WriteEvent returns ctx.Err() without flushing; the frame
// stays buffered and may still reach the client, so callers must not re-send
// it.
func (s *SSEStreamWriter) WriteEvent(ctx context.Context, write func(io.Writer) error) error {
	var reason loomtransport.Reason
	s.lock.Lock()
	defer s.finishObservation(&reason)
	err, failure := s.writeEventLocked(ctx, write)
	reason = failure
	return err
}

// Complete admits one terminal SSE operation. prepare executes under write
// ownership and must not reenter the writer. An encoding error leaves the
// stream usable. Once preparation succeeds, every write/flush outcome is
// terminal. A nil prepare closes without committing headers or a frame.
func (s *SSEStreamWriter) Complete(ctx context.Context, prepare func() (SSEMessage, error)) error {
	var reason loomtransport.Reason
	s.lock.Lock()
	defer s.finishObservation(&reason)
	if err := s.closedErr(); err != nil {
		return err
	}
	var err error
	if prepare != nil {
		message, prepareErr := prepare()
		if prepareErr != nil {
			return prepareErr
		}
		err, reason = s.writeEventLocked(ctx, func(w io.Writer) error {
			return WriteSSEEvent(w, message)
		})
	}
	s.closed = true
	s.observation.RecordClose()
	return err
}

// End finalizes a handler's writer without opening an unused stream. Defer End
// so the original handler panic keeps precedence if an observer also panics.
func (s *SSEStreamWriter) End() {
	if panicked := recover(); panicked != nil {
		defer func() {
			panic(panicked)
		}()
	}
	s.close()
}

// Observation returns the writer's immutable stream identity for protocol-level
// events. It does not change per-call cancellation or physical lifecycle state.
func (s *SSEStreamWriter) Observation() *loomtransport.StreamObserver {
	return s.observation
}

// Started reports whether the successful SSE response has been committed.
func (s *SSEStreamWriter) Started() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.started
}

// Close prevents later control and event writes.
func (s *SSEStreamWriter) Close() error {
	s.close()
	return nil
}

// Write forwards p to the response and records that the frame touched the
// body. A failed response write also counts, because the underlying writer may
// have buffered part of p or become unusable.
func (w *sseFrameWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	if n > 0 || err != nil {
		w.touched = true
	}
	return n, err
}

func streamFailureReason(failed, timeout loomtransport.Reason, err error) loomtransport.Reason {
	if isTimeoutError(err) {
		return timeout
	}
	return failed
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

func (s *SSEStreamWriter) close() {
	s.lock.Lock()
	s.closed = true
	s.observation.RecordClose()
	s.lock.Unlock()
	s.observation.Drain()
}

func (s *SSEStreamWriter) writeEventLocked(ctx context.Context, write func(io.Writer) error) (error, loomtransport.Reason) {
	if err := s.checkContext(ctx); err != nil {
		return err, ""
	}
	if err := s.closedErr(); err != nil {
		return err, ""
	}
	s.initHeaders()
	frame := &sseFrameWriter{w: s.w}
	if err := s.runOperation(ctx, func(http.ResponseWriter) error {
		return write(frame)
	}); err != nil {
		if frame.touched {
			s.fail(err)
		}
		reason := streamFailureReason(loomtransport.ReasonStreamWriteFailed, loomtransport.ReasonStreamWriteTimeout, err)
		return err, reason
	}
	if err := s.checkContext(ctx); err != nil {
		return err, ""
	}
	return s.flush(ctx)
}

func (s *SSEStreamWriter) closedErr() error {
	if s.failErr != nil {
		return fmt.Errorf("%w: %w", ErrSSEStreamClosed, s.failErr)
	}
	if s.closed {
		return ErrSSEStreamClosed
	}
	return nil
}

func (s *SSEStreamWriter) fail(err error) {
	s.closed = true
	s.observation.RecordClose()
	if s.failErr == nil {
		s.failErr = err
	}
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
	if s.started {
		return
	}
	header := s.w.Header()
	setHeaderDefault(header, "Content-Type", "text/event-stream")
	setHeaderDefault(header, "Cache-Control", "no-cache")
	setHeaderDefault(header, "Connection", "keep-alive")
	setHeaderDefault(header, "X-Accel-Buffering", "no")
	s.w.WriteHeader(http.StatusOK)
	s.started = true
	s.observation.RecordOpen()
}

func (s *SSEStreamWriter) flush(ctx context.Context) (error, loomtransport.Reason) {
	err := s.runOperation(ctx, func(http.ResponseWriter) error {
		return http.NewResponseController(s.w).Flush()
	})
	if err != nil {
		s.fail(err)
		return err, streamFailureReason(loomtransport.ReasonStreamFlushFailed, loomtransport.ReasonStreamFlushTimeout, err)
	}
	return nil, ""
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
	// The per-call context may cancel at any point, including after the
	// operation returned but before stop. When the cancellation hook ran and
	// expired the connection deadline, restore the policy's idle state (no
	// deadline) so later operations with a fresh context are not poisoned.
	restore := bounded
	if !stop() {
		deadlineErr := <-cancelDeadline
		if deadlineErr != nil && err == nil {
			err = deadlineErr
		}
		restore = restore || deadlineErr == nil
	}
	if restore {
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

func (s *SSEStreamWriter) finishObservation(reason *loomtransport.Reason) {
	s.lock.Unlock()
	s.observation.Drain()
	if *reason != "" {
		s.observation.Failure(*reason)
	}
}
