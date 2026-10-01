package representation

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// attachTargets binds finalized transport occurrences to existing
// service authority. It does not change the legacy raw example fields.
func attachTargets(endpoint *transportir.Endpoint, method *service.MethodData, errors map[*expr.AttributeExpr]*service.ValueData, schemaOnly bool) {
	request := endpoint.Request
	codec := expr.ValueCodecJSON
	switch {
	case request.SkipBodyEncode:
		codec = expr.ValueCodecRaw
	case request.Multipart:
		codec = expr.ValueCodecMultipart
	case request.FormEncoded:
		codec = expr.ValueCodecForm
	}
	request.BodyValue = ValueTarget(method.PayloadValue, request.Body, request.BodyOriginKey, codec, schemaOnly)
	request.StreamingValue = ValueTarget(method.StreamingPayloadValue, request.StreamingBody, "", expr.ValueCodecJSON, schemaOnly)
	for _, group := range [][]*transportir.Parameter{request.PathParams, request.QueryParams, request.Headers, request.Cookies} {
		for _, parameter := range group {
			parameter.Value = ValueTarget(method.PayloadValue, parameter.Attribute, parameter.Name, expr.ValueCodecText, schemaOnly)
		}
	}
	if request.DocumentBody != nil {
		request.DocumentValue = DocumentTarget(method.PayloadValue, request.DocumentBody, "", true, schemaOnly)
		request.DocumentValues = documentationTargets(request.DocumentValue, request.DocumentContentTypes)
	}
	for _, status := range append(append([]*transportir.ResponseStatus(nil), endpoint.Response.Responses...), endpoint.Response.ErrorResponses...) {
		source := method.ResultValue
		if status.Error != nil {
			source = errors[status.Error.Attribute]
		}
		attachResponseTargets(endpoint, status, source, schemaOnly)
		for _, header := range status.Headers {
			header.Value = ValueTarget(source, header.Attribute, header.Name, expr.ValueCodecText, schemaOnly)
		}
		for _, cookie := range status.Cookies {
			cookie.Value = ValueTarget(source, cookie.Attribute, cookie.Name, expr.ValueCodecText, schemaOnly)
		}
	}
	if endpoint.Stream != nil {
		endpoint.Stream.RequestValue = request.StreamingValue
		endpoint.Stream.ResponseValue = ValueTarget(method.StreamingResultValue, endpoint.Stream.ResponseMessage, "", expr.ValueCodecJSON, schemaOnly)
	}
	attachSSETargets(endpoint)
}

func attachSSETargets(endpoint *transportir.Endpoint) {
	if endpoint.Stream == nil || endpoint.Stream.SSE == nil {
		return
	}
	field := endpoint.Stream.SSE.DataField
	if endpoint.Stream.ResponseValue != nil {
		endpoint.Stream.ResponseValue.SSEDataField = field
	}
	if endpoint.Stream.HasMixedResults {
		return
	}
	for _, status := range endpoint.Response.Responses {
		if status.BodyValue != nil {
			status.BodyValue.SSEDataField = field
			for _, value := range status.BodyValues {
				value.SSEDataField = field
			}
		}
		if status.DocumentValue != nil && !status.IndependentDocumentBody {
			status.DocumentValue.SSEDataField = field
		}
	}
}

func ValueTarget(source *service.ValueData, target *expr.AttributeExpr, selection string, codec expr.ValueCodec, schemaOnly bool) *transportir.ValueTarget {
	if target == nil || target.Type == expr.Empty {
		return nil
	}
	value := &transportir.ValueTarget{Source: source, Anchor: source, Codec: codec}
	if source == nil {
		value.Error = fmt.Errorf("HTTP target has no effective service occurrence")
		return value
	}
	if selection != "" {
		members := source.Occurrence.Members()
		for _, member := range members {
			if member.Name == selection || expr.AttributeName(member.Name) == selection {
				value.Selection = []string{member.Name}
				break
			}
		}
		if len(members) > 0 && len(value.Selection) == 0 {
			value.Error = fmt.Errorf("HTTP target selects unknown service member %q", selection)
			return value
		}
	}
	if codec != expr.ValueCodecJSON {
		value.Boundary = "target codec is outside the builtin JSON projection"
	}
	occurrence, err := source.Context.NewOccurrence(target)
	if err != nil {
		value.Error = err
		return value
	}
	value.ExampleOccurrence = occurrence
	if schemaOnly || len(target.ExtractUserExamples()) == 0 {
		return value
	}
	// Only a distinct authored source changes source authority. A derived copy
	// keeps original authored provenance even when its legacy example was filtered.
	chosen := source.Context.SelectExample(occurrence, expr.ExamplePolicy{Reachable: true})
	if supplied, found := chosen.Source(); found && !source.Context.ContainsExampleSource(source.Occurrence, supplied) {
		value.Source = &service.ValueData{Context: source.Context, Occurrence: occurrence,
			Example: source.Context.Resolve(occurrence, supplied, expr.ValueRoleExample)}
		value.Selection = nil
	}
	return value
}

func DocumentTarget(source *service.ValueData, target *expr.AttributeExpr, selected string, independent, schemaOnly bool) *transportir.ValueTarget {
	if target == nil || target.Type == expr.Empty {
		return nil
	}
	if independent {
		return independentDocumentTarget(source, target, schemaOnly)
	}
	value := ValueTarget(source, target, selected, expr.ValueCodecJSON, schemaOnly)
	if value == nil || value.Error != nil {
		return value
	}
	value.Documentary = true
	if !schemaOnly && value.Error == nil && value.Source != nil {
		value.Plan, value.Error = value.Source.Context.NewValuePlan(value.Source.Occurrence, expr.ValuePlanRequest{
			Target: target, Selection: value.Selection, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanDocumentation,
		})
	}
	return value
}

func independentDocumentTarget(source *service.ValueData, target *expr.AttributeExpr, schemaOnly bool) *transportir.ValueTarget {
	context := expr.NewValueContext()
	if source != nil {
		context = source.Context
	}
	occurrence, err := context.NewOccurrence(target)
	if err != nil {
		return &transportir.ValueTarget{Codec: expr.ValueCodecJSON, Documentary: true, Error: err}
	}
	data := &service.ValueData{Context: context, Occurrence: occurrence}
	value := &transportir.ValueTarget{
		Source: data, Anchor: data, Codec: expr.ValueCodecJSON, Documentary: true,
		ExampleOccurrence: occurrence,
	}
	if !schemaOnly {
		selection := context.SelectExample(occurrence, expr.ExamplePolicy{Reachable: true})
		if supplied, found := selection.Source(); found {
			data.Example = context.Resolve(occurrence, supplied, expr.ValueRoleExample)
		} else {
			// Carrier attachment never supplies a generator. Explicit example
			// preparation later synthesizes through this retained occurrence.
			data.Example = context.Synthesize(selection, nil)
		}
		value.Plan, value.Error = context.NewValuePlan(occurrence, expr.ValuePlanRequest{
			Target: target, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanDocumentation,
		})
	}
	return value
}
