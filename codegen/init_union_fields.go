package codegen

import "github.com/CaliLuke/loom/expr"

// initUnionField retains the constructor's actual field storage. A standalone
// union reference is a pointer, while a required union member is a value.
func initUnionField(arg *InitArgData, targetVar string, source, target *expr.AttributeExpr, attributes *TransformAttrs) (TransformPlan, error) {
	var err error
	attributes.failures, err = transformFailurePlan(source, target, attributes)
	if err != nil {
		return TransformPlan{}, err
	}
	statement, err := transformUnion(source, target, arg.Name, targetVar+"."+arg.FieldName, false, &arg.FieldPointer, attributes)
	if err != nil {
		return TransformPlan{}, err
	}
	helpers, err := transformAttributeHelpers(source, target, attributes, make(map[string]*TransformFunctionData))
	if err != nil {
		return TransformPlan{}, err
	}
	code := renderJenniferSnippet(statement)
	if arg.Pointer {
		code = "if " + arg.Name + " != nil {\n" + code + "\n}\n"
	}
	return TransformPlan{Code: code, Helpers: helpers, ErrorAware: attributes.failures.fallible(source, target, attributes)}, nil
}
