package codegen

import (
	"errors"
	"fmt"

	"github.com/CaliLuke/loom/eval"
)

// Error is the codegen-layer error type. It wraps an underlying error with
// attribution pointing at the DSL expression that triggered it. The rendered
// message leads with the DSL file:line (when available) and the service /
// method names captured by [Context], making codegen failures navigable
// without grepping the design.
//
// Error satisfies [errors.Is] and [errors.As] against the wrapped error.
type Error struct {
	// Expr is the DSL expression blamed for the failure. May be nil.
	Expr eval.Expression
	// Service is the service being generated when the error occurred. May be
	// empty.
	Service string
	// Method is the method being generated when the error occurred. May be
	// empty.
	Method string
	// Err is the wrapped error.
	Err error
}

// NewError wraps err with DSL attribution derived from ctx and (when non-nil)
// expr. Returns nil if err is nil.
func NewError(ctx *Context, expr eval.Expression, err error) error {
	if err == nil {
		return nil
	}
	var existing *Error
	if errors.As(err, &existing) {
		return err
	}
	e := &Error{Expr: expr, Err: err}
	if svc := ctx.CurrentService(); svc != nil {
		e.Service = svc.Name
	}
	if m := ctx.CurrentMethod(); m != nil {
		e.Method = m.Name
	}
	if expr == nil {
		// Prefer ctx's current method as the attributed expression.
		if m := ctx.CurrentMethod(); m != nil {
			e.Expr = m
		} else if svc := ctx.CurrentService(); svc != nil {
			e.Expr = svc
		}
	}
	return e
}

// Error renders the codegen error with DSL attribution when available.
// Example output:
//
//	[design.go:42] service Foo, method Bar: invalid field type for JSON encoding: ...
func (e *Error) Error() string {
	prefix := ""
	if e.Expr != nil {
		if file, line, ok := eval.ExpressionLocation(e.Expr); ok {
			prefix = fmt.Sprintf("[%s:%d] ", file, line)
		}
	}
	scope := ""
	switch {
	case e.Service != "" && e.Method != "":
		scope = fmt.Sprintf("service %s, method %s: ", e.Service, e.Method)
	case e.Service != "":
		scope = fmt.Sprintf("service %s: ", e.Service)
	case e.Method != "":
		scope = fmt.Sprintf("method %s: ", e.Method)
	}
	return prefix + scope + e.Err.Error()
}

// Unwrap returns the wrapped error to support [errors.Is] and [errors.As].
func (e *Error) Unwrap() error {
	return e.Err
}

// Is reports whether target matches the wrapped error chain.
func (e *Error) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// RecoverPanic turns a recovered panic value into an error, preserving an
// inner [*Error] when present. Nil input returns nil so callers can write
//
//	defer func() { if err := codegen.RecoverPanic(recover()); err != nil { ... } }()
func RecoverPanic(r any) error {
	if r == nil {
		return nil
	}
	if e, ok := r.(*Error); ok {
		return e
	}
	if err, ok := r.(error); ok {
		var ce *Error
		if errors.As(err, &ce) {
			return ce
		}
		return err
	}
	return fmt.Errorf("%v", r)
}
