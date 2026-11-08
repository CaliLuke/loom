package representation

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
	loom "github.com/CaliLuke/loom/pkg"
)

// FormJSONShape captures the logical JSON schema of one form body alternative.
// It consumes the resolved occurrence's field names and codecs, retains JSON
// names below JSON-valued fields, and never decodes through a typed Go carrier.
func FormJSONShape(source *expr.AttributeExpr, contract *expr.HTTPFormEncoding, context *codegen.AttributeContext) (*loom.JSONShape, error) {
	if source == nil || contract == nil || context == nil {
		return nil, fmt.Errorf("form schema requires an occurrence, captured encoding and context")
	}
	target := TargetGraph(source)
	normalizeFormTarget(target, contract, nil)
	return formJSONShape(source, target, context)
}

// FormFieldJSONShape captures a logical field alternative under its resolved
// content or style grammar. JSON item contents retain JSON names; text records
// use element names. The source and shared declarations are never mutated.
func FormFieldJSONShape(source *expr.AttributeExpr, codec httpencoding.Codec, context *codegen.AttributeContext) (*loom.JSONShape, error) {
	if source == nil || context == nil {
		return nil, fmt.Errorf("form field schema requires an occurrence and context")
	}
	target := TargetGraph(source)
	normalizeFormFieldTarget(target, codec, nil)
	return formJSONShape(source, target, context)
}

func formJSONShape(source, target *expr.AttributeExpr, context *codegen.AttributeContext) (*loom.JSONShape, error) {
	builder := httpValuePlanBuilder{
		request: expr.ValuePlanRequest{Target: target, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanDocumentation},
		seen:    make(map[*expr.AttributeExpr]bool),
	}
	if err := builder.walk(target, context, false, true); err != nil {
		return nil, err
	}
	values := expr.NewValueContext()
	occurrence, err := values.NewOccurrence(source)
	if err != nil {
		return nil, err
	}
	plan, err := values.NewValuePlan(occurrence, DocumentationPlanRequest(target, builder.request))
	if err != nil {
		return nil, err
	}
	return plan.JSONShape()
}
