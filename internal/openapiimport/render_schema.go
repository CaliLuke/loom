package openapiimport

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/internal/exactnum"
	"github.com/CaliLuke/loom/internal/openapiref"
)

func (r *renderer) attribute(name string, schema *Schema, description *string, path string, metadata ...renderedMetadata) error {
	expression, object, err := r.schemaExpression(schema, path)
	if err != nil {
		return err
	}
	if object {
		r.open("Attribute(%q, func()", name)
		for _, meta := range metadata {
			r.line("Meta(%q, %q)", meta.name, meta.value)
		}
		if err := r.emitAttributeGoType(schema, path); err != nil {
			return err
		}
		if err := r.schemaBlock(schema, path, false, description); err != nil {
			return err
		}
		r.close()
		return nil
	}
	if description != nil || len(metadata) > 0 || r.hasSchemaBlock(schema) ||
		r.effectiveNullable(schema) || r.effectiveUnconstrained(schema) {
		r.open("Attribute(%q, %s, func()", name, expression)
		for _, meta := range metadata {
			r.line("Meta(%q, %q)", meta.name, meta.value)
		}
		if err := r.emitAttributeGoType(schema, path); err != nil {
			return err
		}
		if err := r.schemaAttributeBlock(schema, path, description); err != nil {
			return err
		}
		r.close()
		return nil
	}
	r.line("Attribute(%q, %s)", name, expression)
	return nil
}

func (r *renderer) responseSchemaBody(call string, schema *Schema, path string) error {
	expression, object, err := r.schemaExpression(schema, path)
	if err != nil {
		return err
	}
	if object {
		r.open("%s(func()", call)
		if err := r.schemaBlock(schema, path, false, nil); err != nil {
			return err
		}
		r.close()
		return nil
	}
	if r.hasSchemaBlock(schema) {
		r.open("%s(%s, func()", call, expression)
		if err := r.schemaAttributeBlock(schema, path, nil); err != nil {
			return err
		}
		r.close()
		return nil
	}
	r.line("%s(%s)", call, expression)
	return nil
}

func (r *renderer) resolveSchemaReference(schema *Schema, path string) (*Schema, error) {
	if schema == nil || schema.Ref == "" {
		return schema, nil
	}
	name, err := openapiref.Name(schema.Ref, "#/components/schemas/")
	if err != nil {
		return nil, fmt.Errorf("render OpenAPI design: %s schema reference %q: %w", path, schema.Ref, err)
	}
	named, ok := r.schemas[name]
	if !ok {
		return nil, fmt.Errorf("render OpenAPI design: %s schema reference %q does not resolve", path, schema.Ref)
	}
	return named.Schema, nil
}

func (r *renderer) schemaExpression(schema *Schema, path string) (string, bool, error) {
	if schema == nil {
		return "", false, fmt.Errorf("render OpenAPI design: %s schema is nil", path)
	}
	if schema.Ref != "" {
		return r.schemaReferenceExpression(schema.Ref, path)
	}
	if len(schema.OneOf) > 0 {
		branches := make([]string, 0, len(schema.OneOf))
		for index, branch := range schema.OneOf {
			expression, object, err := r.schemaExpression(branch, fmt.Sprintf("%s/oneOf/%d", path, index))
			if err != nil {
				return "", false, err
			}
			if object {
				return "", false, fmt.Errorf("render OpenAPI design: %s/oneOf/%d inline object union branch is not renderable", path, index)
			}
			branches = append(branches, expression)
		}
		return "OneOf(" + strings.Join(branches, ", ") + ")", false, nil
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
		return stringRepresentation(schema.Format).typeName, false, nil
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
	name, err := openapiref.Name(ref, "#/components/schemas/")
	if err != nil {
		return "", false, fmt.Errorf("render OpenAPI design: %s schema reference %q: %w", path, ref, err)
	}
	named, ok := r.schemas[name]
	if !ok {
		return "", false, fmt.Errorf("render OpenAPI design: %s schema reference %q does not resolve", path, ref)
	}
	if r.references != nil {
		if lookup, selected := r.references.lookups[referenceOccurrenceKey{owner: r.declaration, path: path}]; selected {
			return strconv.Quote(lookup), false, nil
		}
	}
	return "Imported" + named.GoName, false, nil
}

func (r *renderer) objectSchemaExpression(schema *Schema, path string) (string, bool, error) {
	representation, err := classifyObject(schema)
	if err != nil {
		return "", false, fmt.Errorf("render OpenAPI design: %s %w", path, err)
	}
	switch representation {
	case objectFreeMap:
		return "MapOf(String, Any)", false, nil
	case objectStructured:
		return "", true, nil
	}
	value, object, err := r.schemaExpression(schema.AdditionalProperties.Schema, path+"/additionalProperties")
	if err != nil {
		return "", false, err
	}
	if object {
		return "", false, fmt.Errorf("render OpenAPI design: %s/additionalProperties inline object requires a named type", path)
	}
	valueSchema := schema.AdditionalProperties.Schema
	if !r.hasSchemaBlock(valueSchema) && !r.effectiveNullable(valueSchema) {
		return "MapOf(String, " + value + ")", false, nil
	}
	child := r.childRenderer()
	child.open("MapOf(String, %s, func()", value)
	child.open("Elem(func()")
	if err := child.emitNullableGoType(valueSchema, path+"/additionalProperties"); err != nil {
		return "", false, err
	}
	if err := child.schemaAttributeBlock(valueSchema, path+"/additionalProperties", nil); err != nil {
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
	child := r.childRenderer()
	child.open("ArrayOf(%s, func()", item)
	if err := child.emitNullableGoType(schema.Items, path+"/items"); err != nil {
		return "", false, err
	}
	if err := child.schemaAttributeBlock(schema.Items, path+"/items", nil); err != nil {
		return "", false, err
	}
	child.close()
	return strings.TrimSpace(child.builder.String()), false, nil
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

func (r *renderer) schemaBlock(schema *Schema, path string, errorType bool, description *string) error {
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
		name, err := openapiref.Name(base.Ref, "#/components/schemas/")
		if err != nil {
			return fmt.Errorf("render OpenAPI design: %s: %w", basePath, err)
		}
		named := r.schemas[name]
		if named.Schema == nil || named.Schema.Type != "object" {
			return fmt.Errorf("render OpenAPI design: %s object base %q is not an object schema", basePath, base.Ref)
		}
		r.line("Extend(%s)", expression)
	}
	if err := r.objectFieldAttributes(schema, path, errorType); err != nil {
		return err
	}
	return r.schemaAttributeBlock(schema, path, description)
}

func (r *renderer) schemaAttributeBlock(schema *Schema, path string, description *string) error {
	if err := r.schemaRequired(schema, path); err != nil {
		return err
	}
	r.untaggedValidation(schema)
	r.referenceOverlayMetadata(schema)
	if err := r.schemaAnnotations(schema, description); err != nil {
		return err
	}
	if !schema.inheritedFormat && schema.Format == "" && (schema.Type == "integer" || schema.Type == "number") {
		r.line("Meta(%q, %q)", "openapi:format", "")
	}
	if schema.Type == "string" && !schema.inheritedFormat {
		representation := stringRepresentation(schema.Format)
		if representation.raw {
			r.line("Meta(%q, %q)", "openapi:format", schema.Format)
		}
		if representation.formatName != "" {
			r.line("Format(%s)", representation.formatName)
		}
	}
	if err := r.defaultAndExamples(schema, path); err != nil {
		return err
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

// schemaAnnotations emits schema documentation independently of constructor shape.
// A nonnil description identifies a parameter/header occurrence, whose outer
// documentation must remain separate from its schema description.
func (r *renderer) schemaAnnotations(schema *Schema, description *string) error {
	if description != nil {
		r.line("Description(%q)", *description)
		r.line("Meta(%q, %q)", "openapi:description:schema", schema.Description)
	} else if schema.Description != "" {
		r.line("Description(%q)", schema.Description)
	}
	if schema.Title != "" {
		r.line("Title(%q)", schema.Title)
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
	return r.emitExtensions("schema", schema.Extensions)
}

func (r *renderer) untaggedValidation(schema *Schema) {
	if len(schema.OneOf) > 0 {
		r.line("Untagged()")
	}
}

func (r *renderer) referenceOverlayMetadata(schema *Schema) {
	if schema.referenceSiblings && len(schema.Bases) > 0 ||
		schema.Ref != "" && !schema.referenceSiblings && !schema.Nullable && r.hasSchemaBlock(schema) {
		r.line("Meta(%q, %q)", "openapi:allOf:reference", "true")
	}
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
	if schema.Title != "" || schema.Description != "" || len(schema.OneOf) > 0 || len(schema.Bases) > 0 || len(schema.Properties) > 0 || len(schema.Required) > 0 || len(schema.Enum) > 0 ||
		schema.Pattern != "" || schema.Minimum != nil || schema.Maximum != nil || schema.ExclusiveMinimum != nil ||
		schema.ExclusiveMaximum != nil || schema.MinLength != nil || schema.MaxLength != nil ||
		schema.MinItems != nil || schema.MaxItems != nil ||
		schema.AdditionalProperties != nil && schema.AdditionalProperties.Allowed != nil && !*schema.AdditionalProperties.Allowed {
		return true
	}
	if schema.Type == "string" && !schema.inheritedFormat {
		representation := stringRepresentation(schema.Format)
		if representation.raw || representation.formatName != "" {
			return true
		}
	}
	if !schema.inheritedFormat && schema.Format == "" && (schema.Type == "integer" || schema.Type == "number") {
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
	if _, err := openapiref.Name(header.Ref, "#/components/headers/"); err != nil {
		return Header{}, fmt.Errorf("render OpenAPI design: %s header reference %q %w", path, header.Ref, err)
	}
	return Header{}, fmt.Errorf("render OpenAPI design: %s header reference %q does not resolve", path, header.Ref)
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

func (r *renderer) numericValidation(name string, value *exactnum.Number) {
	if value != nil {
		r.line("%s(%q)", name, value.String())
	}
}

func (r *renderer) quotedCall(name string, values []string) {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	r.line("%s(%s)", name, strings.Join(quoted, ", "))
}
