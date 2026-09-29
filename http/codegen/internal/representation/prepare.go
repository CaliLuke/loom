package representation

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// PrepareService captures transport representation ownership before either Go
// emission or schema registration. An emitter supplies its existing semantic
// service data. A standalone schema caller supplies nil: only source occurrences
// are captured, without service generation, example selection or synthesis.
// Go naming remains a separate operation on the emitter's owned transport graph.
func PrepareService(source *expr.HTTPServiceExpr, semantic *service.Data) (*transportir.Service, error) {
	prepared := transportir.BuildService(source)
	if prepared == nil {
		return nil, nil
	}
	context := expr.NewValueContext()
	for _, endpoint := range prepared.Endpoints {
		sourceMethod := source.ServiceExpr.Method(endpoint.MethodName)
		var method *service.MethodData
		if semantic != nil {
			method = semantic.Method(endpoint.MethodName)
		} else {
			var err error
			method, err = captureMethod(context, sourceMethod)
			if err != nil {
				return nil, fmt.Errorf("HTTP representation %s.%s: %w", source.Name(), endpoint.Name, err)
			}
		}
		if method == nil {
			return nil, fmt.Errorf("HTTP representation has no service method %s.%s", source.Name(), endpoint.MethodName)
		}
		bindings, err := bindErrorSources(endpoint, sourceMethod, method, context)
		if err != nil {
			return nil, fmt.Errorf("HTTP representation %s.%s: %w", source.Name(), endpoint.Name, err)
		}
		attachTargets(endpoint, method, bindings, semantic == nil)
	}
	if semantic == nil {
		server, _ := ServiceLayouts(prepared.Endpoints)
		for _, endpoint := range prepared.Endpoints {
			prepareSchemaTargets(endpoint, server)
		}
	}
	return prepared, nil
}

func captureMethod(context *expr.ValueContext, method *expr.MethodExpr) (*service.MethodData, error) {
	if method == nil {
		return nil, fmt.Errorf("missing finalized method")
	}
	result := &service.MethodData{ErrorValues: make(map[string]*service.ValueData)}
	for _, pair := range []struct {
		attribute *expr.AttributeExpr
		value     **service.ValueData
	}{
		{method.Payload, &result.PayloadValue}, {method.Result, &result.ResultValue},
		{method.StreamingPayload, &result.StreamingPayloadValue}, {method.StreamingResult, &result.StreamingResultValue},
	} {
		value, err := captureValue(context, pair.attribute)
		if err != nil {
			return nil, err
		}
		*pair.value = value
	}
	for _, failure := range method.Errors {
		value, err := captureValue(context, failure.AttributeExpr)
		if err != nil {
			return nil, err
		}
		result.ErrorValues[failure.Name] = value
	}
	return result, nil
}

func captureValue(context *expr.ValueContext, attribute *expr.AttributeExpr) (*service.ValueData, error) {
	if attribute == nil || attribute.Type == expr.Empty {
		return nil, nil
	}
	occurrence, err := context.NewOccurrence(attribute)
	if err != nil {
		return nil, err
	}
	return &service.ValueData{Context: context, Occurrence: occurrence}, nil
}

func prepareSchemaTargets(endpoint *transportir.Endpoint, layout Layout) {
	requestContext := schemaContext(true)
	requestContext.JSONPresence = !endpoint.Request.FormEncoded && !endpoint.Request.Multipart
	requestContext.CollectionElementPresence = requestContext.JSONPresence
	layout.Bind(requestContext)
	request := endpoint.Request
	request.BodyValue = BuildValuePlan(request.BodyValue, request.Body, requestContext, expr.ValuePlanSchema)
	request.StreamingValue = BuildValuePlan(request.StreamingValue, request.StreamingBody, requestContext, expr.ValuePlanSchema)
	request.DocumentValue = BuildValuePlan(request.DocumentValue, request.DocumentBody, requestContext, expr.ValuePlanSchema)
	for media, value := range request.DocumentValues {
		request.DocumentValues[media] = BuildValuePlan(value, request.DocumentBody, requestContext, expr.ValuePlanSchema)
	}
	for _, group := range [][]*transportir.Parameter{request.PathParams, request.QueryParams, request.Headers, request.Cookies} {
		for _, parameter := range group {
			parameter.Value = BuildValuePlan(parameter.Value, parameter.Attribute, requestContext, expr.ValuePlanSchema)
		}
	}
	responseContext := schemaContext(false)
	layout.Bind(responseContext)
	for _, response := range serviceResponses(endpoint) {
		response.BodyValue = BuildValuePlan(response.BodyValue, response.Body, responseContext, expr.ValuePlanSchema)
		for media, value := range response.BodyValues {
			response.BodyValues[media] = BuildValuePlan(value, response.Body, responseContext, expr.ValuePlanSchema)
		}
		response.DocumentValue = BuildValuePlan(response.DocumentValue, response.DocumentBody, responseContext, expr.ValuePlanSchema)
		for media, value := range response.DocumentValues {
			response.DocumentValues[media] = BuildValuePlan(value, response.DocumentBody, responseContext, expr.ValuePlanSchema)
		}
		for _, header := range response.Headers {
			header.Value = BuildValuePlan(header.Value, header.Attribute, responseContext, expr.ValuePlanSchema)
		}
		for _, cookie := range response.Cookies {
			cookie.Value = BuildValuePlan(cookie.Value, cookie.Attribute, responseContext, expr.ValuePlanSchema)
		}
	}
	if endpoint.Stream != nil {
		endpoint.Stream.RequestValue = request.StreamingValue
		endpoint.Stream.ResponseValue = PrepareStreamSchema(endpoint, endpoint.Stream.ResponseMessage, false)
	}
}

func schemaContext(request bool) *codegen.AttributeContext {
	context := codegen.NewAttributeContext(request, false, !request, "", codegen.NewNameScope())
	context.JSONPresence = request
	context.CollectionElementPresence = request
	return context
}
