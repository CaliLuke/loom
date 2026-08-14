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
		return stringSchemaExpression(schema.Format), false, nil
	case "integer":
		return integerSchemaExpression(schema.Format), false, nil
	case "number":
		return numberSchemaExpression(schema.Format), false, nil
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

// stringSchemaExpression selects the Loom base type for a string schema.
// "byte" maps to Bytes; every other format, including one Loom does not
// recognize, maps to String. A recognized format additionally gets a
// Format(...) validation call (see validationBlock); an unrecognized one
// renders as a plain String, matching OpenAPI 3.1's rule that unknown format
// values must not fail validation or processing. Analyze reports unrecognized
// non-empty formats as the lossy-allowed "schema-format" diagnostic.
func stringSchemaExpression(format string) string {
	if format == "byte" {
		return "Bytes"
	}
	return "String"
}

// integerSchemaExpression selects the Loom base type for an integer schema.
// "int32" and "int64" map to their fixed-width types; an absent, empty, or
// unrecognized format maps to Int, Loom's unformatted integer, which is the
// widest representation and therefore never narrows the source contract.
// Analyze reports unrecognized non-empty formats as the lossy-allowed
// "schema-format" diagnostic; an absent or empty format is fully supported
// and reported nowhere.
func integerSchemaExpression(format string) string {
	switch format {
	case "int32":
		return "Int32"
	case "int64":
		return "Int64"
	default:
		return "Int"
	}
}

// numberSchemaExpression selects the Loom base type for a number schema.
// "float" maps to Float32; "double", an absent/empty format, or an
// unrecognized format maps to Float64, the widest representation. Analyze
// reports unrecognized non-empty formats as the lossy-allowed "schema-format"
// diagnostic; an absent or empty format is fully supported and reported
// nowhere.
func numberSchemaExpression(format string) string {
	if format == "float" {
		return "Float32"
	}
	return "Float64"
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
		if formatName, ok := stringFormatDSL(schema.Format); ok {
			r.line("Format(%s)", formatName)
		}
		// An unrecognized format renders the attribute without a Format(...)
		// validation; Analyze already reported the omission.
	}
	if schema.Default != nil {
		literal, err := scalarLiteral(schema.Default.Value)
		if err != nil {
			return fmt.Errorf("render OpenAPI design: %s default: %w", path, err)
		}
		r.line("Default(%s)", literal)
	}
	if schema.Deprecated {
		r.line("Meta(%q, %q)", "openapi:deprecated", "true")
	}
	if schema.ReadOnly {
		r.line("Meta(%q, %q)", "openapi:readOnly", "true")
	}
	if schema.WriteOnly {
		r.line("Meta(%q, %q)", "openapi:writeOnly", "true")
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
	if schema == nil {
		return false
	}
	if schema.Description != "" || len(schema.Properties) > 0 || len(schema.Required) > 0 || len(schema.Enum) > 0 ||
		schema.Pattern != "" || schema.Minimum != nil || schema.Maximum != nil || schema.ExclusiveMinimum != nil ||
		schema.ExclusiveMaximum != nil || schema.MinLength != nil || schema.MaxLength != nil ||
		schema.MinItems != nil || schema.MaxItems != nil ||
		schema.AdditionalProperties != nil && schema.AdditionalProperties.Allowed != nil && !*schema.AdditionalProperties.Allowed {
		return true
	}
	if schema.Type == "string" && schema.Format != "" && schema.Format != "byte" {
		if _, ok := stringFormatDSL(schema.Format); ok {
			return true
		}
	}
	return schema.Deprecated || schema.ReadOnly || schema.WriteOnly || schema.Default != nil
}

func (r *renderer) resolveParameter(parameter Parameter, path string) (Parameter, string, error) {
	resolved, componentName, err := resolveParameterReference(parameter, r.document.Components)
	if err != nil {
		return Parameter{}, "", fmt.Errorf("render OpenAPI design: %s %w", path, err)
	}
	return resolved, componentName, nil
}

// resolveHeader normalizes an inline or referenced response Header. Unlike
// resolveParameter, it never returns a resolved component: planDocument
// raises the unconditional "component-header" diagnostic for every
// #/components/headers entry a document declares (see plan.go), so any
// document that reaches this renderer through the package's Analyze/Render
// entry points is already guaranteed to have an empty
// document.Components.Headers set. A header.Ref observed here can therefore
// only be a dangling reference (or one produced by a manually-constructed
// Document that bypassed planDocument), which is reported as unresolved
// rather than silently inlined.
func (r *renderer) resolveHeader(header Header, path string) (Header, error) {
	if header.Ref == "" {
		return header, nil
	}
	name := strings.TrimPrefix(header.Ref, "#/components/headers/")
	if name == header.Ref || name == "" {
		return Header{}, fmt.Errorf("render OpenAPI design: %s header reference %q has the wrong kind", path, header.Ref)
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
