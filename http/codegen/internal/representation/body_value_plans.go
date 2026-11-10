package representation

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type bodyPlanCaptureKey struct {
	context     *expr.ValueContext
	codec       expr.ValueCodec
	sseField    string
	documentary bool
}

// prepareBodyValuePlans captures one exact finalized body per codec/context,
// then builds independently owned plans and example state for its media slots.
// The cache cannot cross body occurrences, mutable layouts or preparation calls.
func prepareBodyValuePlans(primary *transportir.ValueTarget, media map[string]*transportir.ValueTarget, body *expr.AttributeExpr, context *codegen.AttributeContext) *transportir.ValueTarget {
	captures := make(map[bodyPlanCaptureKey]*httpValuePlanCapture)
	build := func(value *transportir.ValueTarget) *transportir.ValueTarget {
		if value == nil || value.Error != nil || value.Source == nil {
			return BuildValuePlan(value, body, context, expr.ValuePlanSchema)
		}
		key := bodyPlanCaptureKey{value.Source.Context, value.Codec, value.SSEDataField, value.Documentary}
		capture := captures[key]
		if capture == nil {
			var err error
			capture, err = captureValuePlan(value, body, context, expr.ValuePlanSchema, nil)
			if err != nil {
				result := *value
				clearTargetValues(&result)
				result.ProjectionContext, result.Error = capture.context, err
				return &result
			}
			captures[key] = capture
		}
		return buildCapturedValuePlan(value, capture, expr.ValuePlanSchema)
	}
	primary = build(primary)
	for name, value := range media {
		media[name] = build(value)
	}
	return primary
}
