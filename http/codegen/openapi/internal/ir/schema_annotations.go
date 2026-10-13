package ir

import (
	"encoding/json/jsontext"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

func (a *Analyzer) schemaValueAnnotations(attr *expr.AttributeExpr) (expr.JSONSchemaAnnotations, error) {
	if a.preparedExamples != nil {
		if annotations, found := a.preparedExamples.Annotations[a.examplePlan]; found {
			return annotations, nil
		}
	}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attr)
	if err != nil {
		return expr.JSONSchemaAnnotations{}, err
	}
	plan, err := context.NewValuePlan(occurrence, representation.DocumentationPlanRequest(expr.ValuePlanRequest{Target: attr}))
	if err != nil {
		return expr.JSONSchemaAnnotations{}, err
	}
	return context.PrepareJSONSchemaAnnotations(occurrence, plan)
}

func schemaAnnotationValue(value expr.JSONSchemaValue) any {
	if wire, emitted := value.Projection().JSON(); emitted {
		return materializeOpenAPIRawJSON(jsontext.Value(wire))
	}
	return toStringMap(value.DeclaredValue())
}

func schemaAnnotationEnums(clauses [][]expr.JSONSchemaValue) [][]any {
	values := make([][]any, len(clauses))
	for index, clause := range clauses {
		values[index] = make([]any, len(clause))
		for member, value := range clause {
			values[index][member] = schemaAnnotationValue(value)
		}
	}
	return values
}
