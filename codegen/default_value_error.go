package codegen

import "github.com/CaliLuke/loom/expr"

// DefaultValueError identifies an expected failure to materialize an authored
// default at an invoked operation. Generation boundaries may return this error
// from an analysis panic without converting unrelated programmer panics into
// design diagnostics. It does not cause unused defaults to be evaluated.
type DefaultValueError struct {
	// Attribute identifies the selected physical default occurrence.
	Attribute *expr.AttributeExpr
	// Err is the underlying resolution or materialization failure.
	Err error
}

// Error returns the materialization diagnostic without changing its member path.
func (e *DefaultValueError) Error() string {
	return e.Err.Error()
}

// Unwrap preserves the underlying failure for errors.Is and errors.As.
func (e *DefaultValueError) Unwrap() error {
	return e.Err
}
