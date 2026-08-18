package openapiimport

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func (r *renderer) openAPIRequestBody(body RequestBody, path string) error {
	schema := schemaWithExamples(body.Schema, body.Examples)
	expression, object, err := r.schemaExpression(schema, path+"/content/schema")
	if err != nil {
		return err
	}
	bodyArg := expression
	if object {
		child := renderer{document: r.document, schemas: r.schemas}
		child.open("func()")
		if err := child.schemaBlock(schema, path+"/content/schema", false); err != nil {
			return err
		}
		child.indent--
		child.line("}")
		bodyArg = strings.TrimSpace(child.builder.String())
	}
	contentTypes := make([]string, len(body.ContentTypes))
	for index, contentType := range body.ContentTypes {
		contentTypes[index] = strconv.Quote(contentType)
	}
	metadata, err := renderedExtensions("requestBody", body.Extensions)
	if err != nil {
		return err
	}
	if body.Description != "" {
		metadata = append(metadata, renderedMetadata{
			name:  "openapi:description:requestBody",
			value: body.Description,
		})
	}
	needsBlock := (!object && r.hasSchemaBlock(schema)) || len(metadata) > 0
	prefix := fmt.Sprintf(
		"OpenAPIRequestBodyTypes(%s, []string{%s}, %t",
		bodyArg,
		strings.Join(contentTypes, ", "),
		body.Required,
	)
	if !needsBlock {
		r.line("%s)", prefix)
		return nil
	}
	r.open("%s, func()", prefix)
	if !object {
		if err := r.validationBlock(schema, path+"/content/schema"); err != nil {
			return err
		}
	}
	for _, item := range metadata {
		r.line("Meta(%q, %q)", item.name, item.value)
	}
	r.close()
	return nil
}

func (r *renderer) operationRequestBodyHTTP(body *renderedBody, path string) error {
	if body == nil {
		return nil
	}
	switch body.mode {
	case requestBodyMultipart:
		r.line("MultipartRequest()")
	case requestBodyForm:
		r.line("FormRequest()")
	case requestBodyRaw:
		r.line("SkipRequestBodyEncodeDecode()")
		if err := r.openAPIRequestBody(body.body, path+"/requestBody"); err != nil {
			return err
		}
	default:
		if err := r.renderJSONBodyMapping(body, path); err != nil {
			return err
		}
	}
	if body.envelope {
		r.line("Body(%q)", body.field)
	}
	if body.mode == requestBodyForm && !body.body.Required && !body.envelope {
		r.line("OptionalRequestBody()")
	}
	return nil
}

func (r *renderer) renderJSONBodyMapping(body *renderedBody, path string) error {
	if body.body.Schema == nil || body.body.Schema.Ref == "" {
		r.line("Body(%q)", body.field)
		return nil
	}
	name := strings.TrimPrefix(body.body.Schema.Ref, "#/components/schemas/")
	named, ok := r.schemas[name]
	if !ok {
		return fmt.Errorf("render OpenAPI design: %s/requestBody schema reference %q does not resolve", path, body.body.Schema.Ref)
	}
	r.open("Body(%q, func()", body.field)
	r.line("Meta(%q, %q)", "openapi:typename", named.Name)
	r.line("Meta(%q, %q)", "openapi:typename:canonical", "true")
	r.close()
	return nil
}

func (r *renderer) payloadBody(body *renderedBody, path string) (bool, error) {
	schema := schemaWithExamples(body.body.Schema, body.body.Examples)
	if body.mode != requestBodyJSON {
		if body.envelope {
			metadata, err := renderedExtensions("requestBody", body.body.Extensions)
			if err != nil {
				return false, err
			}
			if body.body.Description != "" {
				metadata = append(metadata, renderedMetadata{name: "openapi:description:requestBody", value: body.body.Description})
			}
			return body.body.Required, r.attribute(body.field, schema, "", path, metadata...)
		}
		if err := r.emitExtensions("requestBody", body.body.Extensions); err != nil {
			return false, err
		}
		if body.body.Description != "" {
			r.line("Meta(%q, %q)", "openapi:description:requestBody", body.body.Description)
		}
		return false, r.renderRequestTransportBody(schema, path)
	}
	metadata, err := renderedExtensions("requestBody", body.body.Extensions)
	if err != nil {
		return false, err
	}
	if body.body.Description != "" {
		metadata = append(metadata, renderedMetadata{name: "openapi:description:requestBody", value: body.body.Description})
	}
	return body.body.Required, r.attribute(body.field, schema, "", path, metadata...)
}

func (r *renderer) validateRequestTransportBodySchema(schema *Schema, path string) error {
	resolved, err := r.resolveRequestTransportBodySchema(schema, path)
	if err != nil {
		return err
	}
	if resolved == nil || resolved.Type != "object" {
		return fmt.Errorf("render OpenAPI design: %s form and multipart bodies require an object schema", path)
	}
	_, _, err = r.objectSchemaExpression(resolved, path)
	return err
}

func (r *renderer) resolveRequestTransportBodySchema(schema *Schema, path string) (*Schema, error) {
	if schema == nil || schema.Ref == "" {
		return schema, nil
	}
	name := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
	named, ok := r.schemas[name]
	if name == schema.Ref || name == "" || !ok {
		return nil, fmt.Errorf("render OpenAPI design: %s schema reference %q does not resolve", path, schema.Ref)
	}
	return named.Schema, nil
}

func (r *renderer) requestTransportBodyIsMap(schema *Schema, path string) (bool, error) {
	resolved, err := r.resolveRequestTransportBodySchema(schema, path)
	if err != nil {
		return false, err
	}
	if resolved == nil || resolved.Type != "object" {
		return false, fmt.Errorf("render OpenAPI design: %s form and multipart bodies require an object schema", path)
	}
	_, object, err := r.objectSchemaExpression(resolved, path)
	return !object, err
}

func (r *renderer) renderRequestTransportBody(schema *Schema, path string) error {
	if err := r.validateRequestTransportBodySchema(schema, path); err != nil {
		return err
	}
	if schema.Ref != "" {
		expression, _, err := r.schemaExpression(schema, path)
		if err != nil {
			return err
		}
		r.line("Extend(%s)", expression)
		return nil
	}
	return r.schemaBlock(schema, path, false)
}

func (r *renderer) renderRequestTransportMapPayload(body *renderedBody, path string) error {
	schema := schemaWithExamples(body.body.Schema, body.body.Examples)
	expression, object, err := r.schemaExpression(schema, path)
	if err != nil {
		return err
	}
	if object {
		return fmt.Errorf("render OpenAPI design: %s dynamic form map resolved to object properties", path)
	}
	metadata, err := renderedExtensions("requestBody", body.body.Extensions)
	if err != nil {
		return err
	}
	if body.body.Description != "" {
		metadata = append(metadata, renderedMetadata{name: "openapi:description:requestBody", value: body.body.Description})
	}
	if !r.hasSchemaBlock(schema) && len(metadata) == 0 {
		r.line("Payload(%s)", expression)
		return nil
	}
	r.open("Payload(%s, func()", expression)
	if err := r.validationBlock(schema, path); err != nil {
		return err
	}
	for _, item := range metadata {
		r.line("Meta(%q, %q)", item.name, item.value)
	}
	r.close()
	return nil
}

func validateRequestBodyGeneratedLocals(parameters []renderedParameter) error {
	for _, parameter := range parameters {
		if strings.EqualFold(codegen.Goify(parameter.field, false), "body") {
			return fmt.Errorf(
				"render OpenAPI design: request body parameter %q in %q collides with the generated request body local",
				parameter.parameter.Name,
				parameter.parameter.In,
			)
		}
	}
	return nil
}

func requestBodyFieldName(parameters []renderedParameter) string {
	used := make(map[string]struct{}, len(parameters)+1)
	used["body"] = struct{}{}
	for _, parameter := range parameters {
		field := strings.ToLower(codegen.Goify(parameter.field, false))
		used[field] = struct{}{}
	}
	for index := 1; ; index++ {
		candidate := "body"
		if index > 1 {
			candidate += strconv.Itoa(index)
		}
		if _, exists := used[strings.ToLower(codegen.Goify(candidate, false))]; !exists {
			return candidate
		}
	}
}
