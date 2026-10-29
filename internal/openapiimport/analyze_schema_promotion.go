package openapiimport

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func requestBodySchemaPath(base string, body RequestBody) string {
	if len(body.ContentTypes) == 0 {
		return base + "/content/schema"
	}
	return base + "/content/" + escapeJSONPointer(body.ContentTypes[0]) + "/schema"
}

func responseSchemaPath(base string, response Response) string {
	if response.ContentType == "" {
		return base + "/content/schema"
	}
	return base + "/content/" + escapeJSONPointer(response.ContentType) + "/schema"
}

func uniqueComponentName(base string, used map[string]struct{}) string {
	base = codegen.Goify(base, true)
	if base == "" {
		base = "ImportedArrayItem"
	}
	for suffix := 1; ; suffix++ {
		candidate := base
		if suffix > 1 {
			candidate = fmt.Sprintf("%s%d", base, suffix)
		}
		if _, exists := used[candidate]; exists {
			continue
		}
		used[candidate] = struct{}{}
		return candidate
	}
}

func (a *analyzer) prepareObjectConstructors(document *Document) {
	used := make(map[string]struct{}, len(document.Components.Schemas))
	for _, named := range document.Components.Schemas {
		used[named.Name] = struct{}{}
	}

	initialSchemas := len(document.Components.Schemas)
	for index := 0; index < initialSchemas; index++ {
		named := &document.Components.Schemas[index]
		a.prepareSchemaObjects(document, named.Schema, named.Name,
			"#/components/schemas/"+escapeJSONPointer(named.Name), objectDeclaration, used)
	}
	for index := range document.Components.Parameters {
		named := &document.Components.Parameters[index]
		named.Parameter.Schema = a.prepareSchemaObjects(document, named.Parameter.Schema, named.Name+"Parameter",
			"#/components/parameters/"+escapeJSONPointer(named.Name)+"/schema", objectField, used)
	}
	for index := range document.Components.RequestBodies {
		named := &document.Components.RequestBodies[index]
		named.RequestBody.Schema = a.prepareSchemaObjects(document, named.RequestBody.Schema, named.Name+"Request",
			requestBodySchemaPath("#/components/requestBodies/"+escapeJSONPointer(named.Name), named.RequestBody), requestObjectRole(named.RequestBody), used)
	}
	for index := range document.Components.Responses {
		named := &document.Components.Responses[index]
		a.prepareResponseObjects(document, &named.Response, named.Name+"Response",
			"#/components/responses/"+escapeJSONPointer(named.Name), false, used)
	}
	for index := range document.Components.Headers {
		named := &document.Components.Headers[index]
		named.Header.Schema = a.prepareSchemaObjects(document, named.Header.Schema, named.Name+"Header",
			"#/components/headers/"+escapeJSONPointer(named.Name)+"/schema", objectField, used)
	}
	for operationIndex := range document.Operations {
		operation := &document.Operations[operationIndex]
		operationPath := "#/paths/" + escapeJSONPointer(operation.Path) + "/" + strings.ToLower(operation.Method)
		for parameterIndex := range operation.Parameters {
			parameter := &operation.Parameters[parameterIndex]
			parameter.Schema = a.prepareSchemaObjects(document, parameter.Schema, operation.GoName+"Parameter",
				fmt.Sprintf("%s/parameters/%d/schema", operationPath, parameterIndex), objectField, used)
		}
		if operation.RequestBody != nil {
			operation.RequestBody.Schema = a.prepareSchemaObjects(document, operation.RequestBody.Schema, operation.GoName+"Request",
				requestBodySchemaPath(operationPath+"/requestBody", *operation.RequestBody), requestObjectRole(*operation.RequestBody), used)
		}
		for responseIndex := range operation.Responses {
			response := &operation.Responses[responseIndex]
			a.prepareResponseObjects(document, &response.Response, operation.GoName+"Response",
				operationPath+"/responses/"+escapeJSONPointer(response.Status), operationResponseBodyMode(operation) != responseBodyEncoded, used)
		}
	}
}

// prepareSchemaObjects keeps concrete constructor preparation in the existing
// schema traversal, before names and reference edges are planned.
func (a *analyzer) prepareSchemaObjects(
	document *Document,
	schema *Schema,
	name string,
	path string,
	role objectConstructorRole,
	used map[string]struct{},
) *Schema {
	if schema == nil || schema.Ref != "" {
		return schema
	}
	for index := range schema.Properties {
		property := &schema.Properties[index]
		property.Schema = a.prepareSchemaObjects(document, property.Schema, name+codegen.Goify(property.Name, true),
			path+"/properties/"+escapeJSONPointer(property.Name), objectField, used)
	}
	if schema.AdditionalProperties != nil {
		schema.AdditionalProperties.Schema = a.prepareSchemaObjects(document, schema.AdditionalProperties.Schema, name+"AdditionalProperty",
			path+"/additionalProperties", objectMapValue, used)
	}
	for index, branch := range schema.OneOf {
		if branch == nil {
			continue
		}
		if len(schema.MediaTypes) > 0 {
			componentName := uniqueComponentName(name+"Media", used)
			a.prepareSchemaObjects(document, branch, componentName,
				fmt.Sprintf("%s/content/%s/schema", path, escapeJSONPointer(schema.MediaTypes[index])), objectDeclaration, used)
			document.Components.Schemas = append(document.Components.Schemas, NamedSchema{Name: componentName, Schema: branch})
			schema.OneOf[index] = &Schema{Ref: "#/components/schemas/" + componentName}
			continue
		}
		branchName := name + "Variant"
		if len(branch.Required) > 0 {
			branchName = name + codegen.Goify(branch.Required[0], true)
		}
		schema.OneOf[index] = a.prepareSchemaObjects(document, branch, branchName,
			fmt.Sprintf("%s/oneOf/%d", path, index), objectUnionBranch, used)
	}
	schema.Items = a.prepareSchemaObjects(document, schema.Items, name+"Item", path+"/items", objectArrayItem, used)
	if !objectNeedsName(schema, role) {
		return schema
	}
	componentName := uniqueComponentName(name, used)
	document.Components.Schemas = append(document.Components.Schemas, NamedSchema{
		Name:   componentName,
		Schema: schema,
	})
	if role == objectArrayItem {
		a.unsupported(
			"schema-inline-array-item-promoted",
			path,
			fmt.Sprintf("inline object array items are rendered as synthetic component %q", componentName),
		)
	}
	return &Schema{Ref: "#/components/schemas/" + componentName}
}

// prepareResponseObjects includes header occurrences in the same preparation
// owner as the body, preserving the actual wrapper constructor role.
func (a *analyzer) prepareResponseObjects(
	document *Document,
	response *Response,
	name, path string,
	raw bool,
	used map[string]struct{},
) {
	response.Schema = a.prepareSchemaObjects(document, response.Schema, name,
		responseSchemaPath(path, *response), responseObjectRole(*response, raw), used)
	for index := range response.Headers {
		header := &response.Headers[index]
		header.Header.Schema = a.prepareSchemaObjects(document, header.Header.Schema, name+codegen.Goify(header.Name, true)+"Header",
			path+"/headers/"+escapeJSONPointer(header.Name)+"/schema", objectField, used)
	}
}
