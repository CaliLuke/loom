package openapiimport

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func (a *analyzer) promoteInlineArrayItems(document *Document) {
	used := make(map[string]struct{}, len(document.Components.Schemas))
	for _, named := range document.Components.Schemas {
		used[named.Name] = struct{}{}
	}

	initialSchemas := len(document.Components.Schemas)
	for index := 0; index < initialSchemas; index++ {
		named := &document.Components.Schemas[index]
		a.promoteSchemaArrayItems(document, named.Schema, named.Name,
			"#/components/schemas/"+escapeJSONPointer(named.Name), used)
	}
	for index := range document.Components.Parameters {
		named := &document.Components.Parameters[index]
		a.promoteSchemaArrayItems(document, named.Parameter.Schema, named.Name+"Parameter",
			"#/components/parameters/"+escapeJSONPointer(named.Name)+"/schema", used)
	}
	for index := range document.Components.RequestBodies {
		named := &document.Components.RequestBodies[index]
		a.promoteSchemaArrayItems(document, named.RequestBody.Schema, named.Name+"Request",
			requestBodySchemaPath("#/components/requestBodies/"+escapeJSONPointer(named.Name), named.RequestBody), used)
	}
	for index := range document.Components.Responses {
		named := &document.Components.Responses[index]
		a.promoteSchemaArrayItems(document, named.Response.Schema, named.Name+"Response",
			responseSchemaPath("#/components/responses/"+escapeJSONPointer(named.Name), named.Response), used)
	}
	for index := range document.Components.Headers {
		named := &document.Components.Headers[index]
		a.promoteSchemaArrayItems(document, named.Header.Schema, named.Name+"Header",
			"#/components/headers/"+escapeJSONPointer(named.Name)+"/schema", used)
	}
	for operationIndex := range document.Operations {
		operation := &document.Operations[operationIndex]
		operationPath := "#/paths/" + escapeJSONPointer(operation.Path) + "/" + strings.ToLower(operation.Method)
		for parameterIndex := range operation.Parameters {
			parameter := &operation.Parameters[parameterIndex]
			a.promoteSchemaArrayItems(document, parameter.Schema, operation.GoName+"Parameter",
				fmt.Sprintf("%s/parameters/%d/schema", operationPath, parameterIndex), used)
		}
		if operation.RequestBody != nil {
			a.promoteSchemaArrayItems(document, operation.RequestBody.Schema, operation.GoName+"Request",
				requestBodySchemaPath(operationPath+"/requestBody", *operation.RequestBody), used)
		}
		for responseIndex := range operation.Responses {
			response := &operation.Responses[responseIndex]
			a.promoteSchemaArrayItems(document, response.Response.Schema, operation.GoName+"Response",
				responseSchemaPath(operationPath+"/responses/"+escapeJSONPointer(response.Status), response.Response), used)
		}
	}
}

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

func (a *analyzer) promoteSchemaArrayItems(
	document *Document,
	schema *Schema,
	name string,
	path string,
	used map[string]struct{},
) {
	if schema == nil || schema.Ref != "" {
		return
	}
	for index := range schema.Properties {
		property := &schema.Properties[index]
		a.promoteSchemaArrayItems(document, property.Schema, name+codegen.Goify(property.Name, true),
			path+"/properties/"+escapeJSONPointer(property.Name), used)
	}
	if schema.AdditionalProperties != nil {
		a.promoteSchemaArrayItems(document, schema.AdditionalProperties.Schema, name+"AdditionalProperty",
			path+"/additionalProperties", used)
	}
	for index := range schema.OneOf {
		branch := schema.OneOf[index]
		branchName := name + "Variant"
		if len(branch.Required) > 0 {
			branchName = name + codegen.Goify(branch.Required[0], true)
		}
		a.promoteSchemaArrayItems(document, branch, branchName,
			fmt.Sprintf("%s/oneOf/%d", path, index), used)
		if branch.Ref != "" || branch.Type != "object" {
			continue
		}
		componentName := uniqueComponentName(branchName, used)
		document.Components.Schemas = append(document.Components.Schemas, NamedSchema{
			Name:   componentName,
			Schema: branch,
		})
		schema.OneOf[index] = &Schema{Ref: "#/components/schemas/" + componentName}
	}
	if schema.Items == nil {
		return
	}
	a.promoteSchemaArrayItems(document, schema.Items, name+"Item", path+"/items", used)
	if schema.Items.Ref != "" || schema.Items.Type != "object" {
		return
	}

	componentName := uniqueComponentName(name+"Item", used)
	document.Components.Schemas = append(document.Components.Schemas, NamedSchema{
		Name:   componentName,
		Schema: schema.Items,
	})
	schema.Items = &Schema{Ref: "#/components/schemas/" + componentName}
	a.unsupported(
		"schema-inline-array-item-promoted",
		path+"/items",
		fmt.Sprintf("inline object array items are rendered as synthetic component %q", componentName),
	)
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
