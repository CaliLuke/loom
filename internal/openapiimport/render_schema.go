package openapiimport

import (
	"fmt"
	"strconv"
	"strings"
)

func (r *renderer) attribute(name string, schema *Schema, description, path string, metadata ...renderedMetadata) error {
	expression, object, err := r.schemaExpression(schema, path)
	if err != nil {
		return err
	}
	if object {
		r.open("Attribute(%q, func()", name)
		if description != "" {
			r.line("Description(%q)", description)
		}
		for _, meta := range metadata {
			r.line("Meta(%q, %q)", meta.name, meta.value)
		}
		if err := r.schemaBlock(schema, path, false); err != nil {
			return err
		}
		r.close()
		return nil
	}
	if description != "" || len(metadata) > 0 || r.hasSchemaBlock(schema) {
		r.open("Attribute(%q, %s, func()", name, expression)
		if description != "" {
			r.line("Description(%q)", description)
		}
		for _, meta := range metadata {
			r.line("Meta(%q, %q)", meta.name, meta.value)
		}
		if err := r.validationBlock(schema, path); err != nil {
			return err
		}
		r.close()
		return nil
	}
	r.line("Attribute(%q, %s)", name, expression)
	return nil
}

func (r *renderer) schemaExpression(schema *Schema, path string) (string, bool, error) {
	if schema == nil {
		return "", false, fmt.Errorf("render OpenAPI design: %s schema is nil", path)
	}
	if schema.Ref != "" {
		return r.schemaReferenceExpression(schema.Ref, path)
	}
	if schema.Title != "" {
		return "", false, fmt.Errorf("render OpenAPI design: %s schema title is not renderable", path)
	}
	switch schema.Type {
	case "object":
		return r.objectSchemaExpression(schema, path)
	case "array":
		return r.arraySchemaExpression(schema, path)
	case "string":
		return stringSchemaExpression(schema.Format, path)
	case "integer":
		return integerSchemaExpression(schema.Format, path)
	case "number":
		return numberSchemaExpression(schema.Format, path)
	case "boolean":
		return booleanSchemaExpression(schema.Format, path)
	default:
		return "", false, fmt.Errorf("render OpenAPI design: %s schema type %q is not renderable", path, schema.Type)
	}
}

func (r *renderer) schemaReferenceExpression(ref, path string) (string, bool, error) {
	name := strings.TrimPrefix(ref, "#/components/schemas/")
	if name == ref || name == "" {
		return "", false, fmt.Errorf("render OpenAPI design: %s schema reference %q is not a local component schema", path, ref)
	}
	named, ok := r.schemas[name]
	if !ok {
		return "", false, fmt.Errorf("render OpenAPI design: %s schema reference %q does not resolve", path, ref)
	}
	return "Imported" + named.GoName, false, nil
}

func (r *renderer) objectSchemaExpression(schema *Schema, path string) (string, bool, error) {
	if schema.AdditionalProperties == nil || schema.AdditionalProperties.Schema == nil {
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.Allowed != nil && *schema.AdditionalProperties.Allowed {
			return "", false, fmt.Errorf("render OpenAPI design: %s additionalProperties true is not renderable without Any", path)
		}
		return "", true, nil
	}
	if len(schema.Properties) > 0 {
		return "", false, fmt.Errorf("render OpenAPI design: %s object properties with schema-valued additionalProperties are not renderable", path)
	}
	value, _, err := r.schemaExpression(schema.AdditionalProperties.Schema, path+"/additionalProperties")
	if err != nil {
		return "", false, err
	}
	return "MapOf(String, " + value + ")", false, nil
}

func (r *renderer) arraySchemaExpression(schema *Schema, path string) (string, bool, error) {
	if schema.Items == nil {
		return "", false, fmt.Errorf("render OpenAPI design: %s array has no items schema", path)
	}
	item, object, err := r.schemaExpression(schema.Items, path+"/items")
	if err != nil {
		return "", false, err
	}
	if object {
		return "", false, fmt.Errorf("render OpenAPI design: %s inline object array items are not renderable", path)
	}
	if !r.hasSchemaBlock(schema.Items) {
		return "ArrayOf(" + item + ")", false, nil
	}
	child := renderer{document: r.document, schemas: r.schemas}
	child.open("ArrayOf(%s, func()", item)
	if err := child.validationBlock(schema.Items, path+"/items"); err != nil {
		return "", false, err
	}
	child.close()
	return strings.TrimSpace(child.builder.String()), false, nil
}

func stringSchemaExpression(format, path string) (string, bool, error) {
	primitives := map[string]string{"": "String", "byte": "Bytes"}
	if primitive, ok := primitives[format]; ok {
		return primitive, false, nil
	}
	if _, ok := stringFormatDSL(format); !ok {
		return "", false, fmt.Errorf("render OpenAPI design: %s string format %q is not renderable", path, format)
	}
	return "String", false, nil
}

func integerSchemaExpression(format, path string) (string, bool, error) {
	switch format {
	case "int32":
		return "Int32", false, nil
	case "int64":
		return "Int64", false, nil
	default:
		return "", false, fmt.Errorf("render OpenAPI design: %s integer format %q is not renderable", path, format)
	}
}

func numberSchemaExpression(format, path string) (string, bool, error) {
	switch format {
	case "float":
		return "Float32", false, nil
	case "double":
		return "Float64", false, nil
	default:
		return "", false, fmt.Errorf("render OpenAPI design: %s number format %q is not renderable", path, format)
	}
}

func booleanSchemaExpression(format, path string) (string, bool, error) {
	if format != "" {
		return "", false, fmt.Errorf("render OpenAPI design: %s boolean format %q is not renderable", path, format)
	}
	return "Boolean", false, nil
}

func (r *renderer) schemaBlock(schema *Schema, path string, errorType bool) error {
	if schema.Description != "" {
		r.line("Description(%q)", schema.Description)
	}
	if schema.Type == "object" && schema.AdditionalProperties != nil && schema.AdditionalProperties.Allowed != nil && !*schema.AdditionalProperties.Allowed {
		r.line("Meta(%q, %q)", "openapi:additionalProperties", "false")
	}
	fieldOverrides := map[int]string(nil)
	if errorType {
		fields := make([]string, len(schema.Properties))
		for index, property := range schema.Properties {
			fields[index] = property.Name
		}
		fieldOverrides = errorTypeFieldOverrides(fields)
	}
	for index, property := range schema.Properties {
		var metadata []renderedMetadata
		if field := fieldOverrides[index]; field != "" {
			metadata = append(metadata, renderedMetadata{name: "struct:field:name", value: field})
		}
		if err := r.attribute(property.Name, property.Schema, "", path+"/properties/"+escapeJSONPointer(property.Name), metadata...); err != nil {
			return err
		}
	}
	if len(schema.Required) > 0 {
		properties := make(map[string]struct{}, len(schema.Properties))
		for _, property := range schema.Properties {
			properties[property.Name] = struct{}{}
		}
		for _, name := range schema.Required {
			if _, ok := properties[name]; !ok {
				return fmt.Errorf("render OpenAPI design: %s required property %q is not defined", path, name)
			}
		}
		r.quotedCall("Required", schema.Required)
	}
	return r.validationBlock(schema, path)
}

func (r *renderer) validationBlock(schema *Schema, path string) error {
	if schema.Format != "" && schema.Type == "string" && schema.Format != "byte" {
		formatName, ok := stringFormatDSL(schema.Format)
		if !ok {
			return fmt.Errorf("render OpenAPI design: %s string format %q is not renderable", path, schema.Format)
		}
		r.line("Format(%s)", formatName)
	}
	if len(schema.Enum) > 0 {
		values := make([]string, 0, len(schema.Enum))
		for _, value := range schema.Enum {
			literal, err := scalarLiteral(value)
			if err != nil {
				return fmt.Errorf("render OpenAPI design: %s enum: %w", path, err)
			}
			values = append(values, literal)
		}
		r.line("Enum(%s)", strings.Join(values, ", "))
	}
	if schema.Pattern != "" {
		r.line("Pattern(%q)", schema.Pattern)
	}
	r.numericValidation("Minimum", schema.Minimum)
	r.numericValidation("Maximum", schema.Maximum)
	r.numericValidation("ExclusiveMinimum", schema.ExclusiveMinimum)
	r.numericValidation("ExclusiveMaximum", schema.ExclusiveMaximum)
	minLength, err := collectionLength(schema.Type, schema.MinLength, schema.MinItems, path, "minimum")
	if err != nil {
		return err
	}
	maxLength, err := collectionLength(schema.Type, schema.MaxLength, schema.MaxItems, path, "maximum")
	if err != nil {
		return err
	}
	if minLength != nil {
		r.line("MinLength(%d)", *minLength)
	}
	if maxLength != nil {
		r.line("MaxLength(%d)", *maxLength)
	}
	return nil
}

func (r *renderer) hasSchemaBlock(schema *Schema) bool {
	return schema != nil && (schema.Description != "" || schema.Type == "string" && schema.Format != "" && schema.Format != "byte" ||
		len(schema.Properties) > 0 || len(schema.Required) > 0 || len(schema.Enum) > 0 || schema.Pattern != "" ||
		schema.Minimum != nil || schema.Maximum != nil || schema.ExclusiveMinimum != nil || schema.ExclusiveMaximum != nil ||
		schema.MinLength != nil || schema.MaxLength != nil || schema.MinItems != nil || schema.MaxItems != nil ||
		schema.AdditionalProperties != nil && schema.AdditionalProperties.Allowed != nil && !*schema.AdditionalProperties.Allowed)
}

func (r *renderer) resolveParameter(parameter Parameter, path string) (Parameter, string, error) {
	resolved, componentName, err := resolveParameterReference(parameter, r.document.Components)
	if err != nil {
		return Parameter{}, "", fmt.Errorf("render OpenAPI design: %s %w", path, err)
	}
	return resolved, componentName, nil
}

func (r *renderer) resolveHeader(header Header, path string) (Header, error) {
	if header.Ref == "" {
		return header, nil
	}
	name := strings.TrimPrefix(header.Ref, "#/components/headers/")
	if name == header.Ref || name == "" {
		return Header{}, fmt.Errorf("render OpenAPI design: %s header reference %q has the wrong kind", path, header.Ref)
	}
	for _, named := range r.document.Components.Headers {
		if named.Name == name {
			if named.Header.Ref != "" {
				return Header{}, fmt.Errorf("render OpenAPI design: %s nested header references are not renderable", path)
			}
			return named.Header, nil
		}
	}
	return Header{}, fmt.Errorf("render OpenAPI design: %s header reference %q does not resolve", path, header.Ref)
}

func stringFormatDSL(format string) (string, bool) {
	formats := map[string]string{
		"date": "FormatDate", "date-time": "FormatDateTime", "uuid": "FormatUUID",
		"email": "FormatEmail", "hostname": "FormatHostname", "ipv4": "FormatIPv4",
		"ipv6": "FormatIPv6", "uri": "FormatURI",
	}
	name, ok := formats[format]
	return name, ok
}

func scalarLiteral(value any) (string, error) {
	switch actual := value.(type) {
	case string:
		return strconv.Quote(actual), nil
	case bool:
		return strconv.FormatBool(actual), nil
	case int:
		return strconv.Itoa(actual), nil
	case int64:
		return strconv.FormatInt(actual, 10), nil
	case float64:
		return strconv.FormatFloat(actual, 'g', -1, 64), nil
	case nil:
		return "nil", nil
	default:
		return "", fmt.Errorf("value of type %T is not a scalar Go literal", value)
	}
}

func collectionLength(schemaType string, stringValue, itemValue *int64, path, kind string) (*int64, error) {
	if stringValue != nil && itemValue != nil && *stringValue != *itemValue {
		return nil, fmt.Errorf("render OpenAPI design: %s has conflicting %s length and item constraints", path, kind)
	}
	if schemaType == "array" {
		if itemValue != nil {
			return itemValue, nil
		}
		return stringValue, nil
	}
	if itemValue != nil {
		return nil, fmt.Errorf("render OpenAPI design: %s item count constraint applies to non-array schema", path)
	}
	return stringValue, nil
}

func (r *renderer) numericValidation(name string, value *float64) {
	if value != nil {
		r.line("%s(%s)", name, strconv.FormatFloat(*value, 'g', -1, 64))
	}
}

func (r *renderer) quotedCall(name string, values []string) {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	r.line("%s(%s)", name, strings.Join(quoted, ", "))
}
