package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// attachHTTPValueCarriers binds finalized transport occurrences to existing
// service authority. It does not change the legacy raw example fields.
func attachHTTPValueCarriers(endpoint *transportir.Endpoint, method *service.MethodData) {
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
	request.BodyValue = httpValueTarget(method.PayloadValue, request.Body, request.BodyOriginKey, codec)
	request.StreamingValue = httpValueTarget(method.StreamingPayloadValue, request.StreamingBody, "", expr.ValueCodecJSON)
	for _, group := range [][]*transportir.Parameter{request.PathParams, request.QueryParams, request.Headers, request.Cookies} {
		for _, parameter := range group {
			parameter.Value = httpValueTarget(method.PayloadValue, parameter.Attribute, parameter.Name, expr.ValueCodecText)
		}
	}
	if request.DocumentBody != nil {
		request.DocumentValue = httpDocumentValue(method.PayloadValue, request.DocumentBody, "", true)
	}
	for _, status := range append(append([]*transportir.ResponseStatus(nil), endpoint.Response.Responses...), endpoint.Response.ErrorResponses...) {
		source := method.ResultValue
		if status.Error != nil {
			source = method.ErrorValues[status.Error.Name]
		}
		responseCodec := expr.ValueCodecJSON
		if status.BinaryBody || endpoint.Response.SkipBodyEncode || endpoint.Response.FileResponse {
			responseCodec = expr.ValueCodecRaw
		}
		status.BodyValue = httpValueTarget(source, status.Body, status.BodyOrigin, responseCodec)
		status.DocumentValue = httpDocumentValue(source, status.DocumentBody, status.BodyOrigin, status.IndependentDocumentBody)
		for _, header := range status.Headers {
			header.Value = httpValueTarget(source, header.Attribute, header.Name, expr.ValueCodecText)
		}
		for _, cookie := range status.Cookies {
			cookie.Value = httpValueTarget(source, cookie.Attribute, cookie.Name, expr.ValueCodecText)
		}
	}
	if endpoint.Stream != nil {
		endpoint.Stream.RequestValue = request.StreamingValue
		endpoint.Stream.ResponseValue = httpValueTarget(method.StreamingResultValue, endpoint.Stream.ResponseMessage, "", expr.ValueCodecJSON)
	}
}

func httpValueTarget(source *service.ValueData, target *expr.AttributeExpr, selection string, codec expr.ValueCodec) *transportir.ValueTarget {
	if target == nil || target.Type == expr.Empty {
		return nil
	}
	value := &transportir.ValueTarget{Source: source, Codec: codec}
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
	if len(target.ExtractUserExamples()) == 0 {
		return value
	}
	// Only a distinct authored source changes source authority. A derived copy
	// keeps original authored provenance even when its legacy example was filtered.
	occurrence, err := source.Context.NewOccurrence(target)
	if err != nil {
		value.Error = err
		return value
	}
	chosen := source.Context.SelectExample(occurrence, expr.ExamplePolicy{Reachable: true})
	if supplied, found := chosen.Source(); found && !source.Context.ContainsExampleSource(source.Occurrence, supplied) {
		value.Source = &service.ValueData{Context: source.Context, Occurrence: occurrence,
			Example: source.Context.Resolve(occurrence, supplied, expr.ValueRoleExample)}
		value.Selection = nil
	}
	return value
}

func httpDocumentValue(source *service.ValueData, target *expr.AttributeExpr, selected string, independent bool) *transportir.ValueTarget {
	if target == nil || target.Type == expr.Empty {
		return nil
	}
	value := httpValueTarget(source, target, selected, expr.ValueCodecJSON)
	if independent {
		context := expr.NewValueContext()
		if source != nil {
			context = source.Context
		}
		occurrence, err := context.NewOccurrence(target)
		if err != nil {
			value.Error = err
			return value
		}
		selection := context.SelectExample(occurrence, expr.ExamplePolicy{Reachable: true})
		var result expr.ValueResult
		if supplied, found := selection.Source(); found {
			result = context.Resolve(occurrence, supplied, expr.ValueRoleExample)
		} else {
			// No new sampling is performed during carrier attachment. A later
			// documentation consumer may synthesize through this same occurrence.
			result = context.Synthesize(selection, nil)
		}
		value.Source = &service.ValueData{Context: context, Occurrence: occurrence, Example: result}
		value.Error = nil
		value.Selection = nil
	}
	value.Documentary = true
	if value.Error == nil && value.Source != nil {
		value.Plan, value.Error = value.Source.Context.NewValuePlan(value.Source.Occurrence, expr.ValuePlanRequest{
			Target: target, Selection: value.Selection, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanDocumentation,
		})
	}
	return value
}
