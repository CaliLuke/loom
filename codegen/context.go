package codegen

import (
	"io"
	"log/slog"
	"os"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// Context carries request-scoped data through the codegen pipeline: a
// structured logger for decision-point tracing and slots tracking the
// service and method currently being analyzed.
//
// Context is the shared plumbing that lets every stage of codegen emit
// traceable output and, in the future, attribute errors to the DSL element
// being processed. A zero-value Context is safe to use: logging becomes a
// no-op and the current-expression slots stay empty.
//
// Contexts are cheap to derive. [Context.WithService] and [Context.WithMethod]
// return a new Context that narrows the current-expression slot without
// mutating the receiver.
type Context struct {
	// Logger is the structured logger used for decision-point tracing. A nil
	// Logger disables logging.
	Logger *slog.Logger

	// currentService is the service whose code is being generated, if any.
	currentService *expr.ServiceExpr
	// currentMethod is the method whose code is being generated, if any.
	currentMethod *expr.MethodExpr
	// currentAttribute is the finest-grained DSL expression in scope (e.g.,
	// a payload or response attribute), if any.
	currentAttribute *expr.AttributeExpr
}

// NewContext returns a Context whose logger respects the DEBUG_LOOM
// environment variable: when DEBUG_LOOM is set to a non-empty, non-"0" value
// the logger emits debug-level records to stderr, otherwise it discards all
// records.
func NewContext() *Context {
	return &Context{Logger: defaultLogger()}
}

// NewSilentContext returns a Context whose logger discards every record.
// Useful for tests and callers that never want codegen log output regardless
// of environment.
func NewSilentContext() *Context {
	return &Context{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

// WithService returns a copy of c with its current-service slot set to s.
// The receiver is unchanged.
func (c *Context) WithService(s *expr.ServiceExpr) *Context {
	if c == nil {
		return &Context{currentService: s}
	}
	out := *c
	out.currentService = s
	out.currentMethod = nil
	return &out
}

// WithMethod returns a copy of c with its current-method slot set to m.
// The receiver is unchanged. The current-attribute slot is cleared so stale
// finer-grained attribution does not leak across method boundaries.
func (c *Context) WithMethod(m *expr.MethodExpr) *Context {
	if c == nil {
		return &Context{currentMethod: m}
	}
	out := *c
	out.currentMethod = m
	out.currentAttribute = nil
	return &out
}

// WithAttribute returns a copy of c with its current-attribute slot set to a.
// Use this when entering a deeper scope (e.g., analyzing a specific payload
// field) to improve error attribution. The receiver is unchanged.
func (c *Context) WithAttribute(a *expr.AttributeExpr) *Context {
	if c == nil {
		return &Context{currentAttribute: a}
	}
	out := *c
	out.currentAttribute = a
	return &out
}

// CurrentAttribute returns the finest-grained attribute being processed, or
// nil if none is set.
func (c *Context) CurrentAttribute() *expr.AttributeExpr {
	if c == nil {
		return nil
	}
	return c.currentAttribute
}

// CurrentExpression returns the most specific DSL expression recorded by the
// context, falling back from attribute -> method -> service. Returns nil if
// none are set.
func (c *Context) CurrentExpression() eval.Expression {
	if c == nil {
		return nil
	}
	if c.currentAttribute != nil {
		return c.currentAttribute
	}
	if c.currentMethod != nil {
		return c.currentMethod
	}
	if c.currentService != nil {
		return c.currentService
	}
	return nil
}

// CurrentService returns the service being analyzed, or nil if none is set.
func (c *Context) CurrentService() *expr.ServiceExpr {
	if c == nil {
		return nil
	}
	return c.currentService
}

// CurrentMethod returns the method being analyzed, or nil if none is set.
func (c *Context) CurrentMethod() *expr.MethodExpr {
	if c == nil {
		return nil
	}
	return c.currentMethod
}

// Debug emits a debug-level log record with the given message and attributes.
// Service and method names from the context are attached automatically when
// available. Calling Debug on a nil Context or with a nil Logger is a no-op.
func (c *Context) Debug(msg string, args ...any) {
	if c == nil || c.Logger == nil {
		return
	}
	c.Logger.Debug(msg, c.appendScope(args)...)
}

// Info emits an info-level log record. See [Context.Debug].
func (c *Context) Info(msg string, args ...any) {
	if c == nil || c.Logger == nil {
		return
	}
	c.Logger.Info(msg, c.appendScope(args)...)
}

// Warn emits a warning-level log record. See [Context.Debug].
func (c *Context) Warn(msg string, args ...any) {
	if c == nil || c.Logger == nil {
		return
	}
	c.Logger.Warn(msg, c.appendScope(args)...)
}

func (c *Context) appendScope(args []any) []any {
	if c.currentService != nil {
		args = append(args, "service", c.currentService.Name)
	}
	if c.currentMethod != nil {
		args = append(args, "method", c.currentMethod.Name)
	}
	return args
}

func defaultLogger() *slog.Logger {
	if !debugEnabled() {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func debugEnabled() bool {
	v := os.Getenv("DEBUG_LOOM")
	return v != "" && v != "0"
}
