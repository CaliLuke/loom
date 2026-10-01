package representation

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/examplegen"
)

// PrepareEndpointExamples binds every documentation surface after schema plans
// exist. Callers omit this stage when a custom projector owns raw sampling.
func PrepareEndpointExamples(endpoint *transportir.Endpoint, generator *expr.ExampleGenerator) error {
	if endpoint == nil {
		return nil
	}
	serviceName := ""
	if endpoint.Service != nil {
		serviceName = endpoint.Service.Name
	}
	prefix := []string{"service", serviceName, "method", endpoint.MethodName, "openapi"}
	prepare := func(target *transportir.ValueTarget, attribute *expr.AttributeExpr, parts ...string) error {
		scope := append(append([]string(nil), prefix...), parts...)
		if err := PrepareTargetExamples(target, attribute, examplegen.ForScope(generator, scope...)); err != nil {
			return fmt.Errorf("prepare HTTP examples %v: %w", parts, err)
		}
		return nil
	}
	if endpoint.Request == nil || endpoint.Response == nil {
		return fmt.Errorf("HTTP example preparation has incomplete endpoint %s.%s", serviceName, endpoint.MethodName)
	}
	if err := prepareEndpointRequestExamples(endpoint.Request, endpoint.Security, prepare); err != nil {
		return err
	}
	return prepareEndpointResponseExamples(endpoint.Response, prepare)
}

type endpointExamplePreparer func(*transportir.ValueTarget, *expr.AttributeExpr, ...string) error

func prepareEndpointRequestExamples(
	request *transportir.Request,
	security *transportir.Security,
	prepare endpointExamplePreparer,
) error {
	for _, input := range []struct {
		target    *transportir.ValueTarget
		attribute *expr.AttributeExpr
		parts     []string
	}{
		{request.BodyValue, request.Body, []string{"request-body"}},
		{request.StreamingValue, request.StreamingBody, []string{"streaming-request"}},
		{request.DocumentValue, request.DocumentBody, []string{"request-document"}},
	} {
		if err := prepare(input.target, input.attribute, input.parts...); err != nil {
			return err
		}
	}
	for media, target := range request.DocumentValues {
		if err := prepare(target, request.DocumentBody, "request-document-media", media); err != nil {
			return err
		}
	}
	for _, group := range []struct {
		location   string
		parameters []*transportir.Parameter
	}{
		{"path", request.PathParams},
		{"query", request.QueryParams},
		{"header", request.Headers},
		{"cookie", request.Cookies},
	} {
		for _, parameter := range group.parameters {
			if group.location != "path" && security.IsParameter(group.location, parameter.HTTPName) {
				continue
			}
			if err := prepare(parameter.Value, parameter.Attribute, "parameter", parameter.HTTPName); err != nil {
				return err
			}
		}
	}
	return nil
}

func prepareEndpointResponseExamples(response *transportir.Response, prepare endpointExamplePreparer) error {
	for _, statusResponse := range append(append([]*transportir.ResponseStatus(nil), response.Responses...), response.ErrorResponses...) {
		status := fmt.Sprint(statusResponse.StatusCode)
		if err := prepare(statusResponse.BodyValue, statusResponse.Body, "response-body", status); err != nil {
			return err
		}
		if err := prepare(statusResponse.DocumentValue, statusResponse.DocumentBody, "response-document", status); err != nil {
			return err
		}
		for media, target := range statusResponse.BodyValues {
			if err := prepare(target, statusResponse.Body, "response-media", status, media); err != nil {
				return err
			}
		}
		for media, target := range statusResponse.DocumentValues {
			if err := prepare(target, statusResponse.DocumentBody, "response-document-media", status, media); err != nil {
				return err
			}
		}
		for _, header := range statusResponse.Headers {
			if err := prepare(header.Value, header.Attribute, "response-header", status, header.HTTPName); err != nil {
				return err
			}
		}
		for _, cookie := range statusResponse.Cookies {
			if err := prepare(cookie.Value, cookie.Attribute, "response-cookie", status, cookie.HTTPName); err != nil {
				return err
			}
		}
	}
	return nil
}
