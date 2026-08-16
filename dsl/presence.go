package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

type nullExample struct{}

// Nullable marks the current attribute occurrence as accepting an explicit
// null value. Requiredness remains controlled independently by Required.
func Nullable() {
	attribute, ok := eval.Current().(*expr.AttributeExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	attribute.Nullable = true
}

// Null returns the explicit null sentinel accepted by Example for nullable
// attributes. Null is not a data type or a default value.
func Null() any {
	return nullExample{}
}

func setExampleValue(example *expr.ExampleExpr, value any) {
	if _, ok := value.(nullExample); ok {
		example.Value = nil
		example.ExplicitNull = true
		return
	}
	if isNilDSLValue(value) {
		eval.ReportError("null example values must use Null()")
		return
	}
	example.Value = value
	example.ExplicitNull = false
}
