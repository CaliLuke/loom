package openapiimport

import (
	"fmt"
	"sort"
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
		if err := r.emitAttributeGoType(schema, path); err != nil {
			return err
		}
		if err := r.schemaBlock(schema, path, false); err != nil {
			return err
		}
		r.close()
		return nil
	}
	if description != "" || len(metadata) > 0 || r.hasSchemaBlock(schema) ||
		r.effectiveNullable(schema) || r.effectiveUnconstrained(schema) {
		r.open("Attribute(%q, %s, func()", name, expression)
		if description != "" {
			r.line("Description(%q)", description)
		}
		for _, meta := range metadata {
			r.line("Meta(%q, %q)", meta.name, meta.value)
		}
		if err := r.emitAttributeGoType(schema, path); err != nil {
			return err
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

func (r *renderer) validateRequestTransportBodySchema(schema *Schema, path string) error {
	resolved := schema
	if schema != nil && schema.Ref != "" {
		name := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
		named, ok := r.schemas[name]
		if name == schema.Ref || name == "" || !ok {
			return fmt.Errorf("render OpenAPI design: %s schema reference %q does not resolve", path, schema.Ref)
		}
		resolved = named.Schema
	}
	if resolved == nil || resolved.Type != "object" {
		return fmt.Errorf("render OpenAPI design: %s form and multipart bodies require an object schema", path)
	}
	_, object, err := r.objectSchemaExpression(resolved, path)
	if err != nil {
		return err
	}
	if !object {
		return fmt.Errorf("render OpenAPI design: %s form and multipart bodies require object properties", path)
	}
	return nil
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

func (r *renderer) openAPIBody(schema *Schema, path string) error {
	expression, object, err := r.schemaExpression(schema, path)
	if err != nil {
		return err
	}
	if object {
		r.open("OpenAPIBody(func()")
		if err := r.schemaBlock(schema, path, false); err != nil {
			return err
		}
		r.close()
		return nil
	}
	if r.hasSchemaBlock(schema) {
		r.open("OpenAPIBody(%s, func()", expression)
		if err := r.validationBlock(schema, path); err != nil {
			return err
		}
		r.close()
		return nil
	}
	r.line("OpenAPIBody(%s)", expression)
	return nil
}

func (r *renderer) schemaExpression(schema *Schema, path string) (string, bool, error) {
	if schema == nil {
		return "", false, fmt.Errorf("render OpenAPI design: %s schema is nil", path)
	}
	if schema.Ref != "" {
		return r.schemaReferenceExpression(schema.Ref, path)
	}
	if schema.Unconstrained {
		return "Any", false, nil
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
			if len(schema.Properties) > 0 || len(schema.Required) > 0 || len(schema.Bases) > 0 {
				return "", false, fmt.Errorf("render OpenAPI design: %s object members with additionalProperties true are not renderable", path)
			}
			return "MapOf(String, Any)", false, nil
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
	valueSchema := schema.AdditionalProperties.Schema
	if !r.hasSchemaBlock(valueSchema) && !r.effectiveNullable(valueSchema) {
		return "MapOf(String, " + value + ")", false, nil
	}
	child := renderer{document: r.document, schemas: r.schemas}
	child.open("MapOf(String, %s, func()", value)
	child.open("Elem(func()")
	if err := child.emitNullableGoType(valueSchema, path+"/additionalProperties"); err != nil {
		return "", false, err
	}
	if err := child.validationBlock(valueSchema, path+"/additionalProperties"); err != nil {
		return "", false, err
	}
	child.close()
	child.close()
	return strings.TrimSpace(child.builder.String()), false, nil
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
	if !r.hasSchemaBlock(schema.Items) && !r.effectiveNullable(schema.Items) {
		return "ArrayOf(" + item + ")", false, nil
	}
	child := renderer{document: r.document, schemas: r.schemas}
	child.open("ArrayOf(%s, func()", item)
	if err := child.emitNullableGoType(schema.Items, path+"/items"); err != nil {
		return "", false, err
	}
	if err := child.validationBlock(schema.Items, path+"/items"); err != nil {
		return "", false, err
	}
	child.close()
	return strings.TrimSpace(child.builder.String()), false, nil
}

// stringSchemaExpression selects the Loom base type for a string schema.
// "byte" and "binary" map to Bytes; every other format, including one Loom does not
// recognize, maps to String. A recognized format additionally gets a
// Format(...) validation call (see validationBlock); an unrecognized one
// renders as a plain String, matching OpenAPI 3.1's rule that unknown format
// values must not fail validation or processing. Analyze reports unrecognized
// non-empty formats as the lossy-allowed "schema-format" diagnostic.
func stringSchemaExpression(format string) string {
	if format == "byte" || format == "binary" {
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
	for index, base := range schema.Bases {
		if base == nil || base.Ref == "" {
			return fmt.Errorf("render OpenAPI design: %s/bases/%d object base must be a schema reference", path, index)
		}
		basePath := fmt.Sprintf("%s/bases/%d", path, index)
		expression, _, err := r.schemaExpression(base, basePath)
		if err != nil {
			return err
		}
		name := strings.TrimPrefix(base.Ref, "#/components/schemas/")
		named := r.schemas[name]
		if named.Schema == nil || named.Schema.Type != "object" {
			return fmt.Errorf("render OpenAPI design: %s object base %q is not an object schema", basePath, base.Ref)
		}
		r.line("Extend(%s)", expression)
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
	if schema.Title != "" {
		r.line("Title(%q)", schema.Title)
	}
	if err := r.emitExtensions("schema", schema.Extensions); err != nil {
		return err
	}
	if schema.Format == "" && (schema.Type == "integer" || schema.Type == "number") {
		r.line("Meta(%q, %q)", "openapi:format", "")
	}
	if schema.Type == "string" && schema.Format == "byte" {
		r.line("Meta(%q, %q)", "openapi:format", schema.Format)
	}
	if schema.Format != "" && schema.Type == "string" && schema.Format != "byte" && schema.Format != "binary" {
		if formatName, ok := stringFormatDSL(schema.Format); ok {
			r.line("Format(%s)", formatName)
		}
		// An unrecognized format renders the attribute without a Format(...)
		// validation; Analyze already reported the omission.
	}
	if err := r.defaultAndExamples(schema, path); err != nil {
		return err
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
	if err := r.enumValidation(schema, path); err != nil {
		return err
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

func (r *renderer) defaultAndExamples(schema *Schema, path string) error {
	if schema.Default != nil {
		if schema.Default.Value == nil {
			return fmt.Errorf("render OpenAPI design: %s default: null defaults are not supported", path)
		}
		literal, err := scalarLiteral(schema.Default.Value)
		if err != nil {
			return fmt.Errorf("render OpenAPI design: %s default: %w", path, err)
		}
		r.line("Default(%s)", literal)
	}
	for _, example := range schema.Examples {
		if err := r.example(example, path); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) enumValidation(schema *Schema, path string) error {
	if len(schema.Enum) == 0 {
		return nil
	}
	values := make([]string, 0, len(schema.Enum))
	for _, value := range schema.Enum {
		literal, err := scalarLiteral(value)
		if err != nil {
			return fmt.Errorf("render OpenAPI design: %s enum: %w", path, err)
		}
		values = append(values, literal)
	}
	r.line("Enum(%s)", strings.Join(values, ", "))
	return nil
}

func (r *renderer) hasSchemaBlock(schema *Schema) bool {
	if schema == nil {
		return false
	}
	if schema.Title != "" || schema.Description != "" || len(schema.Bases) > 0 || len(schema.Properties) > 0 || len(schema.Required) > 0 || len(schema.Enum) > 0 ||
		schema.Pattern != "" || schema.Minimum != nil || schema.Maximum != nil || schema.ExclusiveMinimum != nil ||
		schema.ExclusiveMaximum != nil || schema.MinLength != nil || schema.MaxLength != nil ||
		schema.MinItems != nil || schema.MaxItems != nil ||
		schema.AdditionalProperties != nil && schema.AdditionalProperties.Allowed != nil && !*schema.AdditionalProperties.Allowed {
		return true
	}
	if schema.Type == "string" && schema.Format != "" && schema.Format != "byte" && schema.Format != "binary" {
		if _, ok := stringFormatDSL(schema.Format); ok {
			return true
		}
	}
	if schema.Type == "string" && schema.Format == "byte" {
		return true
	}
	if schema.Format == "" && (schema.Type == "integer" || schema.Type == "number") {
		return true
	}
	return schema.Nullable || schema.Deprecated || schema.ReadOnly || schema.WriteOnly || schema.Default != nil || len(schema.Examples) > 0 ||
		len(schema.Extensions) > 0
}

func (r *renderer) emitNullableGoType(schema *Schema, path string) error {
	_ = path
	if !r.effectiveNullable(schema) {
		return nil
	}
	r.line("Nullable()")
	return nil
}

func (r *renderer) example(example Example, path string) error {
	literal, err := exampleLiteral(example.Value)
	if err != nil {
		return fmt.Errorf("render OpenAPI design: %s example: %w", path, err)
	}
	structured := example.ComponentName != "" || example.DataValue || example.SerializedValue != ""
	if example.Name == "" && example.Summary == "" && example.Description == "" && !structured {
		r.line("Example(%s)", literal)
		return nil
	}
	name := example.Name
	if name == "" {
		name = "default"
	}
	if example.Summary == "" && example.Description == "" && !structured {
		r.line("Example(%q, %s)", name, literal)
		return nil
	}
	r.open("Example(%q, func()", name)
	if example.ComponentName != "" {
		r.line("Meta(%q, %q)", "openapi:component:example", example.ComponentName)
	}
	if example.DataValue {
		r.line("Meta(%q)", "openapi:example:dataValue")
	}
	if example.SerializedValue != "" {
		r.line("Meta(%q, %q)", "openapi:example:serializedValue", example.SerializedValue)
	}
	if example.Summary != "" && example.Summary != name {
		r.line("Meta(%q, %q)", "openapi:example:summary", example.Summary)
	}
	if example.Description != "" {
		r.line("Description(%q)", example.Description)
	}
	r.line("Value(%s)", literal)
	r.close()
	return nil
}

func exampleLiteral(value any) (string, error) {
	return exampleLiteralValue(value, false)
}

func exampleLiteralValue(value any, nested bool) (string, error) {
	if value == nil {
		if nested {
			return "nil", nil
		}
		return "Null()", nil
	}
	if literal, ok := exampleScalarLiteral(value); ok {
		return literal, nil
	}
	switch actual := value.(type) {
	case []any:
		return exampleArrayLiteral(actual)
	case map[string]any:
		return exampleMapLiteral(actual)
	default:
		return "", fmt.Errorf("value of type %T is not a supported Loom example", value)
	}
}

func exampleScalarLiteral(value any) (string, bool) {
	switch actual := value.(type) {
	case string:
		return strconv.Quote(actual), true
	case bool:
		return strconv.FormatBool(actual), true
	case int:
		return strconv.Itoa(actual), true
	case int8:
		return strconv.FormatInt(int64(actual), 10), true
	case int16:
		return strconv.FormatInt(int64(actual), 10), true
	case int32:
		return strconv.FormatInt(int64(actual), 10), true
	case int64:
		return strconv.FormatInt(actual, 10), true
	case uint:
		return strconv.FormatUint(uint64(actual), 10), true
	case uint8:
		return strconv.FormatUint(uint64(actual), 10), true
	case uint16:
		return strconv.FormatUint(uint64(actual), 10), true
	case uint32:
		return strconv.FormatUint(uint64(actual), 10), true
	case uint64:
		return strconv.FormatUint(actual, 10), true
	case float32:
		return strconv.FormatFloat(float64(actual), 'g', -1, 32), true
	case float64:
		return strconv.FormatFloat(actual, 'g', -1, 64), true
	default:
		return "", false
	}
}

func exampleArrayLiteral(items []any) (string, error) {
	values := make([]string, len(items))
	for index, item := range items {
		literal, err := exampleLiteralValue(item, true)
		if err != nil {
			return "", err
		}
		values[index] = literal
	}
	return "[]any{" + strings.Join(values, ", ") + "}", nil
}

func exampleMapLiteral(mapping map[string]any) (string, error) {
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		literal, err := exampleLiteralValue(mapping[key], true)
		if err != nil {
			return "", err
		}
		values = append(values, strconv.Quote(key)+": "+literal)
	}
	return "Val{" + strings.Join(values, ", ") + "}", nil
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
