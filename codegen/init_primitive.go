package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

// initPrimitiveField retains the selected physical target and reports loss when
// an authored integer representation cannot carry the supplied source value.
func initPrimitiveField(arg *InitArgData, args []*InitArgData, targetVar string, source, target *expr.AttributeExpr, context *TransformAttrs) (TransformPlan, error) {
	targetRef := context.TargetCtx.Scope.Ref(target, context.TargetCtx.Pkg(target))
	conversion, _, err := representation.ClassifyIntegerConversion(source, target)
	if err != nil {
		return TransformPlan{}, err
	}
	checked := conversion.Checked
	var code string
	if checked {
		for _, operand := range []struct {
			attribute *expr.AttributeExpr
			context   *AttributeContext
			opaque    bool
		}{{source, context.SourceCtx, conversion.SourceOpaque}, {target, context.TargetCtx, conversion.TargetOpaque}} {
			if operand.opaque {
				// The generator cannot inspect an external declaration. Bitwise
				// complement is an explicit compile-time integer capability check.
				code += representation.IntegerCapabilityAssertion(operand.context.Scope.Ref(operand.attribute, operand.context.Pkg(operand.attribute)))
			}
		}
	}
	if checked && arg.FieldName == "" {
		code += "var " + targetVar + " " + targetRef + "\n"
	}
	value := arg.Name
	if arg.Pointer {
		code += "if " + value + " != nil {\n"
		value = "*" + value
	}
	cast := fmt.Sprintf("%s(%s)", targetRef, value)
	if checked {
		locals := representationLocals(targetRef + " " + targetVar)
		for _, input := range args {
			locals.Unique(input.Name)
		}
		converted := locals.Unique("convertedValue")
		sourceRef := context.SourceCtx.Scope.Ref(source, context.SourceCtx.Pkg(source))
		code += fmt.Sprintf("if *transformErr == nil {\n%s := %s\nif %s {\n*transformErr = %s\n} else {\n", converted, cast, representation.IntegerLossCondition(sourceRef, value, converted), representation.IntegerLossError(targetRef, value))
		cast = converted
	}
	switch {
	case arg.FieldPointer:
		code += fmt.Sprintf("tmp%s := %s\n%s.%s = &tmp%s\n", arg.Name, cast, targetVar, arg.FieldName, arg.Name)
	case arg.FieldName != "":
		code += fmt.Sprintf("%s.%s = %s\n", targetVar, arg.FieldName, cast) // nolint: namescope -- target.field assignment, not a type ref
	default:
		assign := ":="
		if checked {
			assign = "="
		}
		code += fmt.Sprintf("%s %s %s\n", targetVar, assign, cast)
	}
	if checked {
		code += "}\n}\n"
	}
	if arg.Pointer {
		code += "}\n"
	}
	return TransformPlan{Code: code, ErrorAware: checked}, nil
}
