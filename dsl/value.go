package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// Val is an alias for expr.Val.
type Val expr.Val

// Value sets the example value.
//
// Value must appear in Example.
//
// Value takes one argument: the example value.
//
// Example:
//
//	Example("A simple bottle", func() {
//	    Description("This bottle has an ID set to 1")
//	    Value(Val{"ID": 1})
//	})
func Value(val any) {
	switch e := eval.Current().(type) {
	case *expr.ExampleExpr:
		if v, ok := val.(expr.Val); ok {
			val = map[string]any(v)
		}
		setExampleValue(e, val)
	default:
		eval.IncompatibleDSL()
	}
}
