package service

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type viewValidationCarrier struct {
	function  codegen.ValidationFunction
	nullable  bool
	intrinsic bool
	pointer   bool
	nilable   bool
}

func newViewValidationCarrier(att *expr.AttributeExpr, ctx *codegen.AttributeContext, view string) viewValidationCarrier {
	declaration := &expr.AttributeExpr{Type: att.Type, Meta: att.Meta}
	function := codegen.NamedValidationFunction(declaration, ctx, "result")
	if view != expr.DefaultView {
		function.Name += codegen.Goify(view, true)
	}
	return viewValidationCarrier{
		function:  function,
		nullable:  expr.IsNullable(att),
		intrinsic: expr.IsNullable(declaration),
		pointer:   expr.IsObject(att.Type),
		nilable:   expr.IsObject(att.Type) || expr.IsArray(att.Type) || expr.IsMap(att.Type),
	}
}

func (c viewValidationCarrier) missing(source string) string {
	if c.nullable {
		return "!" + source + ".Present()"
	}
	return source + " == nil"
}

func (c viewValidationCarrier) validation(source, path string) string {
	argument := source
	unwrap := c.nullable && !c.intrinsic
	if unwrap {
		argument = "actual"
		if c.pointer {
			argument = "&actual"
		}
	}
	call := "if err2 := " + c.function.Call(argument, path) + "; err2 != nil {\n\terr = loom.MergeErrors(err, err2)\n}"
	if unwrap {
		return "if actual, ok := " + source + ".Value(); ok {\n" + indentViewValidation(call) + "\n}"
	}
	return call
}

func indentViewValidation(code string) string {
	return "\t" + strings.ReplaceAll(code, "\n", "\n\t")
}
