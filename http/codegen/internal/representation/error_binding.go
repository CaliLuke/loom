package representation

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// bindErrorSources preserves the declaration selected when the HTTP mapping was
// finalized. Its declaration scope may differ from the method's same-name error.
// Existing semantic values are reused by source attribute identity. Referenced
// errors absent from service analysis receive only a structural occurrence: this
// preparation must not select or synthesize another example. Their legacy raw
// example owner remains authoritative until that consumer migrates.
func bindErrorSources(endpoint *transportir.Endpoint, method *expr.MethodExpr, data *service.MethodData, fallback *expr.ValueContext) (map[*expr.AttributeExpr]*service.ValueData, error) {
	bindings := make(map[*expr.AttributeExpr]*service.ValueData)
	context := methodValueContext(method, data, fallback)
	for _, failure := range method.Errors {
		if value := data.ErrorValues[failure.Name]; value != nil {
			bindings[failure.AttributeExpr] = value
		}
	}
	for _, response := range serviceResponses(endpoint) {
		if response.Error == nil {
			continue
		}
		attribute := response.Error.Attribute
		if _, found := bindings[attribute]; found {
			continue
		}
		value, err := captureValue(context, attribute)
		if err != nil {
			return nil, fmt.Errorf("capture HTTP error %q: %w", response.Error.Name, err)
		}
		bindings[attribute] = value
	}
	return bindings, nil
}

func methodValueContext(source *expr.MethodExpr, method *service.MethodData, fallback *expr.ValueContext) *expr.ValueContext {
	for _, value := range []*service.ValueData{method.PayloadValue, method.ResultValue, method.StreamingPayloadValue, method.StreamingResultValue} {
		if value != nil {
			return value.Context
		}
	}
	for _, failure := range source.Errors {
		if value := method.ErrorValues[failure.Name]; value != nil {
			return value.Context
		}
	}
	return fallback
}
