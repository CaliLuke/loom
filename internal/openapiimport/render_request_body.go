package openapiimport

import (
	"fmt"
	"strconv"
	"strings"
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
