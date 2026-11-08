package openapiimport

import "github.com/pb33f/libopenapi/datamodel/high/base"

const (
	schemaAnnotationTitle uint16 = 1 << iota
	schemaAnnotationDescription
	schemaAnnotationDeprecated
	schemaAnnotationReadOnly
	schemaAnnotationWriteOnly
	schemaAnnotationExample
	schemaAnnotationExamples
	schemaAnnotationDefault
)

// retainSchemaPresence records presence before normalization. Ordinary schema
// admission never gains permission from an SSE occurrence sharing a declaration.
func retainSchemaPresence(schema *Schema, source *base.Schema) {
	if source.GoLow() == nil {
		return
	}
	low := source.GoLow()
	schema.examplesPresent = !low.Examples.IsEmpty()
	schema.emptyExamples = schema.examplesPresent && len(low.Examples.Value) == 0
	for _, annotation := range []struct {
		bit     uint16
		present bool
	}{
		{schemaAnnotationTitle, !low.Title.IsEmpty()},
		{schemaAnnotationDescription, !low.Description.IsEmpty()},
		{schemaAnnotationDeprecated, !low.Deprecated.IsEmpty()},
		{schemaAnnotationReadOnly, !low.ReadOnly.IsEmpty()},
		{schemaAnnotationWriteOnly, !low.WriteOnly.IsEmpty()},
		{schemaAnnotationExample, !low.Example.IsEmpty()},
		{schemaAnnotationExamples, !low.Examples.IsEmpty()},
		{schemaAnnotationDefault, !low.Default.IsEmpty()},
	} {
		if annotation.present {
			schema.annotationPresence |= annotation.bit
		}
	}
	for _, item := range []struct {
		name    string
		present bool
	}{
		{"contentEncoding", !low.ContentEncoding.IsEmpty()},
		{"contentMediaType", !low.ContentMediaType.IsEmpty()},
		{"contentSchema", !low.ContentSchema.IsEmpty()},
	} {
		if item.present {
			schema.contentKeywords = append(schema.contentKeywords, item.name)
		}
	}
}

func schemaPresenceDiagnostics(schema *Schema, path string) Diagnostics {
	var diagnostics Diagnostics
	if schema.emptyExamples {
		diagnostics.add("schema-examples-presence", path+"/examples", "authored empty examples cannot be preserved by the current DSL; absence and inherited examples are not an empty authored group")
	}
	if len(schema.contentKeywords) > 0 {
		diagnostics.add("schema-content", path, "content keywords are only supported inside the dedicated SSE wire envelope")
	}
	return diagnostics
}

// hasBooleanAnnotation distinguishes an authored false annotation from absence.
// A true value also carries an annotation in programmatically built schemas,
// which have no parsed source-presence bits.
func (schema *Schema) hasBooleanAnnotation(bit uint16, value bool) bool {
	return value || schema.annotationPresence&bit != 0
}
