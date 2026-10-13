package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"google.golang.org/protobuf/proto"
)

type (
	streamOperationKind uint8
	streamContextKey    struct{}
	streamContext       struct {
		mu       sync.Mutex
		ctx      context.Context
		cancel   context.CancelCauseFunc
		stops    []func() bool
		client   bool
		terminal error
		done     chan struct{}
		active   [2]*streamOperation
	}
	streamOperation struct {
		done      chan struct{}
		completed bool
		canceled  bool
		value     any
		err       error
		panic     any
		panicked  bool
	}
)

const (
	streamSend streamOperationKind = iota
	streamReceive
	streamFinalReceive
	streamHalfClose
)

// SendStreamMessage sends a detached protobuf message using the stream lifetime
// installed by ServeStream or NewStreamInvoker. An already-done operation
// context performs no I/O. Cancellation during native I/O terminates the RPC;
// server I/O is reclaimed when the synchronous service handler returns.
func SendStreamMessage[T proto.Message](ctx, streamCtx context.Context, message T, send func(T) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	owner, err := streamContextFor(streamCtx)
	if err != nil {
		return err
	}
	detached := proto.Clone(message).(T)
	_, _, err = runStreamOperation(ctx, owner, streamSend, func() (struct{}, error) {
		return struct{}{}, send(detached)
	})
	return err
}

// ReceiveStreamMessage receives into native-worker-owned storage. A canceled
// operation never exposes a result still being filled by native I/O. final marks
// CloseAndRecv, which reserves both native directions. A successful final client
// receive releases the opening context.
// Admitted client receive errors are terminal; server receive EOF is only a half-close.
func ReceiveStreamMessage[T proto.Message](ctx, streamCtx context.Context, final bool, receive func() (T, error)) (T, error) {
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	owner, err := streamContextFor(streamCtx)
	if err != nil {
		var zero T
		return zero, err
	}
	kind := streamReceive
	if final {
		kind = streamFinalReceive
	}
	value, admitted, err := runStreamOperation(ctx, owner, kind, receive)
	if admitted && owner.client && (err != nil || final) {
		cause := err
		if cause == nil {
			cause = io.EOF
		}
		owner.abort(cause)
	}
	return value, err
}

// CloseStreamSend closes only the client send direction. It preserves the
// opening context so subsequent receives can obtain the server's final status.
// After terminal release, half-close succeeds without starting native I/O.
func CloseStreamSend(ctx, streamCtx context.Context, closeSend func() error) error {
	owner, err := streamContextFor(streamCtx)
	if err != nil {
		return err
	}
	_, _, err = runStreamOperation(ctx, owner, streamHalfClose, func() (struct{}, error) {
		return struct{}{}, closeSend()
	})
	return err
}

func newStreamContext(parent context.Context, client bool) (context.Context, *streamContext) {
	ctx, cancel := context.WithCancelCause(parent)
	owner := &streamContext{cancel: cancel, client: client, done: make(chan struct{})}
	owner.ctx = context.WithValue(ctx, streamContextKey{}, owner)
	owner.watch(parent)
	return owner.ctx, owner
}

func streamContextFor(streamCtx context.Context) (*streamContext, error) {
	if streamCtx == nil {
		return nil, errors.New("gRPC stream has no invocation context")
	}
	owner, ok := streamCtx.Value(streamContextKey{}).(*streamContext)
	if !ok {
		return nil, errors.New("gRPC stream requires ServeStream or NewStreamInvoker")
	}
	return owner, nil
}

func runStreamOperation[T any](ctx context.Context, owner *streamContext, kind streamOperationKind, fn func() (T, error)) (T, bool, error) {
	var zero T
	op, err := owner.begin(ctx, kind)
	if err != nil {
		return zero, false, err
	}
	if op == nil {
		return zero, false, nil
	}
	callbackDone := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		defer close(callbackDone)
		owner.cancelOperation(op, ctx.Err())
	})
	defer func() {
		if !stop() {
			<-callbackDone
		}
	}()
	go func() {
		var value T
		var err error
		panicked := true
		defer func() {
			owner.complete(kind.directions(), op, value, err, recover(), panicked)
		}()
		owner.mu.Lock()
		canceled := op.canceled
		owner.mu.Unlock()
		if !canceled {
			value, err = fn()
		}
		panicked = false
	}()
	select {
	case <-op.done:
	case <-owner.done:
	}
	owner.mu.Lock()
	if !op.completed {
		err = owner.terminal
		if kind == streamHalfClose {
			err = nil
		}
		owner.mu.Unlock()
		return zero, true, err
	}
	owner.mu.Unlock()
	if op.panicked {
		panic(op.panic)
	}
	return op.value.(T), true, op.err
}

func (s *streamContext) begin(ctx context.Context, kind streamOperationKind) (*streamOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	contextErr := ctx.Err()
	terminal := s.terminal
	if terminal == nil {
		terminal = s.ctx.Err()
	}
	if kind == streamHalfClose && terminal != nil {
		return nil, nil
	}
	if contextErr != nil {
		return nil, contextErr
	}
	if terminal != nil {
		return nil, terminal
	}
	direction := kind.directions()
	for slot, active := range s.active {
		if direction&(1<<slot) != 0 && active != nil {
			return nil, errors.New("concurrent gRPC stream operations in the same direction")
		}
	}
	op := &streamOperation{done: make(chan struct{})}
	for slot := range s.active {
		if direction&(1<<slot) != 0 {
			s.active[slot] = op
		}
	}
	return op, nil
}

func (k streamOperationKind) directions() int {
	switch k {
	case streamReceive:
		return 2
	case streamFinalReceive:
		return 3
	default:
		return 1
	}
}

func (s *streamContext) cancelOperation(op *streamOperation, cause error) {
	s.mu.Lock()
	if !op.completed {
		s.abortLocked(cause)
	}
	s.mu.Unlock()
}

func (s *streamContext) abort(cause error) {
	s.mu.Lock()
	s.abortLocked(cause)
	s.mu.Unlock()
}

func (s *streamContext) abortLocked(cause error) {
	if s.terminal != nil {
		return
	}
	for _, op := range s.active {
		if op != nil && !op.completed {
			op.canceled = true
		}
	}
	s.terminal = cause
	s.cancel(cause)
	for _, stop := range s.stops {
		stop()
	}
	s.stops = nil
	close(s.done)
}

func (s *streamContext) watch(parent context.Context) {
	stop := context.AfterFunc(parent, func() {
		s.abort(context.Cause(parent))
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.terminal != nil {
		stop()
		return
	}
	s.stops = append(s.stops, stop)
}

func (s *streamContext) cause() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.terminal
}

func (s *streamContext) complete(direction int, op *streamOperation, value any, err error, panicValue any, panicked bool) {
	s.mu.Lock()
	for slot := range s.active {
		if direction&(1<<slot) != 0 {
			s.active[slot] = nil
		}
	}
	late := op.canceled
	if !late {
		op.value, op.err, op.panic, op.panicked = value, err, panicValue, panicked
		op.completed = true
		close(op.done)
	}
	s.mu.Unlock()
	if late && panicked {
		// Panic contents may contain payload credentials. Report the event and
		// type without exporting the value or changing completed observation.
		slog.ErrorContext(context.WithoutCancel(s.ctx), "gRPC native stream operation panicked after cancellation", "panic_type", fmt.Sprintf("%T", panicValue))
	}
}
