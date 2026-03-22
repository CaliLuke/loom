package transformassign

import "fmt"

type PrimitiveObjectPlan struct {
	SourceField    string
	TargetVar      string
	TargetField    string
	Expression     string
	TempVar        string
	SourcePointer  bool
	TargetPointer  bool
	SourceRequired bool
}

func BuildPrimitiveObjectAssignment(plan PrimitiveObjectPlan) (string, string, bool) {
	switch {
	case plan.SourcePointer && !plan.SourceRequired:
		return "", conditionalPrimitiveAssignment(plan), true
	case plan.TargetPointer:
		return "", pointerPrimitiveAssignment(plan), true
	default:
		return plan.Expression, "", false
	}
}

func conditionalPrimitiveAssignment(plan PrimitiveObjectPlan) string {
	code := fmt.Sprintf("if %s != nil {\n", plan.SourceField)
	if plan.TargetPointer {
		code += pointerPrimitiveAssignment(plan)
	} else {
		code += fmt.Sprintf("%s.%s = %s\n", plan.TargetVar, plan.TargetField, plan.Expression)
	}
	code += "}\n"
	return code
}

func pointerPrimitiveAssignment(plan PrimitiveObjectPlan) string {
	return fmt.Sprintf("%s := %s\n%s.%s = &%s\n", plan.TempVar, plan.Expression, plan.TargetVar, plan.TargetField, plan.TempVar)
}
