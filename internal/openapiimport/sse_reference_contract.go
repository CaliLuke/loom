package openapiimport

import (
	"reflect"

	"github.com/CaliLuke/loom/internal/openapiref"
)

// sseReferenceContract resolves declaration edges for comparison only. The
// normalized source and rendering occurrence remain untouched. Annotation-only
// siblings can be compared against an expansion; assertion intersections retain
// their explicit reference form rather than claiming general schema equivalence.
func (p *documentPlanner) sseReferenceContract(source *Schema, seen map[*Schema]bool) (*Schema, bool) {
	// Failed normalization is not a bare reference: projection must not erase
	// the unsupported authored facts retained by its diagnostic.
	if source != nil && source.referenceError != nil {
		return nil, false
	}
	if source == nil || source.Ref == "" {
		return source, true
	}
	if seen[source] {
		return nil, false
	}
	seen[source] = true
	name, err := openapiref.Name(source.Ref, "#/components/schemas/")
	if err != nil {
		return nil, false
	}
	declaration, present := p.schemas[name]
	if !present {
		return nil, false
	}
	base, ok := p.sseReferenceContract(declaration.Schema, seen)
	if !ok || base == nil {
		return nil, false
	}
	if !source.referenceSiblings {
		return base, true
	}
	if !sseAnnotationOnly(source) {
		return source, true
	}
	result := *base
	if source.annotationPresence&schemaAnnotationTitle != 0 {
		result.Title = source.Title
	}
	if source.annotationPresence&schemaAnnotationDescription != 0 {
		result.Description = source.Description
	}
	if source.annotationPresence&schemaAnnotationDeprecated != 0 {
		result.Deprecated = source.Deprecated
	}
	if source.annotationPresence&schemaAnnotationReadOnly != 0 {
		result.ReadOnly = source.ReadOnly
	}
	if source.annotationPresence&schemaAnnotationWriteOnly != 0 {
		result.WriteOnly = source.WriteOnly
	}
	if source.annotationPresence&(schemaAnnotationExample|schemaAnnotationExamples) != 0 {
		result.Examples = source.Examples
		result.examplesPresent = source.examplesPresent
		result.emptyExamples = source.emptyExamples
		result.annotationPresence &^= (schemaAnnotationExample | schemaAnnotationExamples)
	}
	if source.annotationPresence&schemaAnnotationDefault != 0 {
		result.Default = source.Default
	}
	result.annotationPresence |= source.annotationPresence
	return &result, true
}

func sseAnnotationOnly(source *Schema) bool {
	if source.Format != "" && !source.inheritedFormat {
		return false
	}
	remaining := *source
	remaining.Ref, remaining.Type, remaining.Format, remaining.sourceType, remaining.sourcePath = "", "", "", "", ""
	remaining.Nullable, remaining.declaredNull, remaining.referenceSiblings, remaining.inheritedFormat = false, false, false, false
	remaining.Title, remaining.Description = "", ""
	remaining.Deprecated, remaining.ReadOnly, remaining.WriteOnly = false, false, false
	remaining.Examples, remaining.Default = nil, nil
	remaining.examplesPresent, remaining.emptyExamples, remaining.annotationPresence = false, false, 0
	return reflect.DeepEqual(remaining, Schema{})
}
