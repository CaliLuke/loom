package log

import (
	"context"
	"slices"
)

const (
	ctxLogger ctxKey = iota + 1
)

// Context initializes a context for logging or updates its existing logger.
// Updates retain buffered entries and fields. Contexts derived with With keep
// their own configuration snapshot; updates do not alter that configuration.
func Context(ctx context.Context, opts ...LogOption) context.Context {
	l, ok := ctx.Value(ctxLogger).(*logger)
	if !ok {
		l = &logger{options: defaultOptions(), buffer: &logBuffer{}}
	}
	l.lock.Lock()
	defer l.lock.Unlock()
	configured := *l.options
	configured.outputs = slices.Clone(l.options.outputs)
	configured.keyvals = slices.Clone(l.options.keyvals)
	configured.kvfuncs = slices.Clone(l.options.kvfuncs)
	for _, opt := range opts {
		opt(&configured)
	}
	l.options = &configured
	if l.options.disableBuffering != nil && l.options.disableBuffering(ctx) {
		l.buffer.flush()
	}
	return context.WithValue(ctx, ctxLogger, l)
}

// WithContext will inject the second context in the given one.
//
// It is useful when building middleware handlers such as log.HTTP
func WithContext(parentCtx, logCtx context.Context) context.Context {
	logger, ok := logCtx.Value(ctxLogger).(*logger)
	if !ok {
		return parentCtx
	}
	return context.WithValue(parentCtx, ctxLogger, logger)
}

// MustContainLogger will panic if the given context is missing the logger.
//
// It can be used during server initialisation when you have a function or
// middleware that you want to ensure receives a context with a logger.
func MustContainLogger(logCtx context.Context) {
	_, ok := logCtx.Value(ctxLogger).(*logger)
	if !ok {
		panic("provided a context without a logger. Use log.Context")
	}
}

// DebugEnabled returns true if the given context has debug logging enabled.
func DebugEnabled(ctx context.Context) bool {
	v := ctx.Value(ctxLogger)
	if v == nil {
		return false
	}
	l := v.(*logger)
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.options.debug
}

// requestContext starts a fresh logging scope while preserving the incoming
// request's context and the root logger's configuration and inherited fields.
func requestContext(ctx, logCtx context.Context, fields ...Fielder) context.Context {
	parent := logCtx.Value(ctxLogger).(*logger)
	parent.lock.Lock()
	request := &logger{
		options: parent.options,
		buffer:  &logBuffer{},
		keyvals: parent.keyvals.merge(fields),
	}
	parent.lock.Unlock()
	ctx = context.WithValue(ctx, ctxLogger, request)
	if request.options.disableBuffering != nil && request.options.disableBuffering(ctx) {
		FlushAndDisableBuffering(ctx)
	}
	return ctx
}
