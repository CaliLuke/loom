package ir

import (
	"fmt"
	"maps"
	"slices"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type taggedFormProjection struct {
	analyzer   *Analyzer
	key, tag   string
	components map[string]string
	inline     map[*Schema]*Schema
}

// analyzeFormOccurrence projects the captured native root after ordinary schema
// analysis. Its components are detached members of the existing representation
// registry; final naming compares their complete graph with JSON occurrences.
func (a *Analyzer) analyzeFormOccurrence(attribute *expr.AttributeExpr, context string, target *transportir.ValueTarget, form *expr.HTTPFormEncoding) *Schema {
	if form == nil || len(form.TaggedBranches) == 0 {
		return a.analyzePreparedOccurrence(attribute, context, target)
	}
	root := a.analyzePreparedSchemaPositions(attribute, context, target, representationRoot(attribute, target))
	union := expr.AsUnion(form.Body.Type)
	projection := &taggedFormProjection{analyzer: a, key: union.GetTypeKey(), components: make(map[string]string), inline: make(map[*Schema]*Schema)}
	result := projection.union(root, union.GetValueKey(), form.TaggedBranches)
	a.occurrences = append(a.occurrences, schemaOccurrenceAnalysis{result, a})
	return result
}

func (p *taggedFormProjection) union(source *Schema, valueKey string, branches []*expr.HTTPFormTaggedBranch) *Schema {
	if source == nil {
		return nil
	}
	result := *source
	if name, local := schemaComponentName(source.Ref); local {
		result.Ref = p.component(name, "root", func(schema *Schema) *Schema {
			return p.union(schema, valueKey, branches)
		})
		return &result
	}
	if source.Discriminator == nil {
		p.analyzer.err = fmt.Errorf("tagged form representation has no discriminator")
		return &result
	}
	references := make(map[string]string)
	for _, branch := range branches {
		if branch.Object == nil {
			continue // The scalar/JSON value-key envelope already matches the wire.
		}
		ref := source.Discriminator.Mapping[branch.Tag]
		name, local := schemaComponentName(ref)
		if !local {
			p.analyzer.err = fmt.Errorf("tagged form branch %q has no local envelope", branch.Tag)
			return &result
		}
		child := &taggedFormProjection{analyzer: p.analyzer, key: p.key, tag: branch.Tag, components: make(map[string]string), inline: make(map[*Schema]*Schema)}
		references[ref] = child.component(name, "envelope", func(envelope *Schema) *Schema {
			payload := envelope.Properties[valueKey]
			flat := child.object(payload)
			// Envelope annotations remain on their owning wrapper; the payload's
			// own annotations and constraints remain on the projected object.
			wrapper := *envelope
			wrapper.Properties = map[string]*Schema{p.key: envelope.Properties[p.key]}
			wrapper.Required = []string{p.key}
			wrapper.AllOf = append(slices.Clone(envelope.AllOf), flat)
			return &wrapper
		})
	}
	result.OneOf = slices.Clone(source.OneOf)
	for index, branch := range result.OneOf {
		if replacement := references[branch.Ref]; replacement != "" {
			copy := *branch
			copy.Ref = replacement
			result.OneOf[index] = &copy
		}
	}
	discriminator := *source.Discriminator
	discriminator.Mapping = maps.Clone(discriminator.Mapping)
	for key, ref := range discriminator.Mapping {
		if replacement := references[ref]; replacement != "" {
			discriminator.Mapping[key] = replacement
		}
	}
	if replacement := references[discriminator.DefaultMapping]; replacement != "" {
		discriminator.DefaultMapping = replacement
	}
	result.Discriminator = &discriminator
	projectTaggedFormValues(&result, func(value any) any {
		return taggedFormValue(value, p.key, valueKey, branches)
	})
	return &result
}

// object is the pullback of an object schema through adding the discriminator.
// Only same-value composition edges are transformed. Property values, array
// items and additional-member values keep their independent JSON contracts.
func (p *taggedFormProjection) object(source *Schema) *Schema {
	if source == nil {
		p.analyzer.err = fmt.Errorf("tagged form envelope has no payload schema")
		return nil
	}
	if prior := p.inline[source]; prior != nil {
		return prior
	}
	result := *source
	p.inline[source] = &result
	if name, local := schemaComponentName(source.Ref); local {
		result.Ref = p.component(name, "object", p.object)
	}
	result.Properties = maps.Clone(source.Properties)
	if result.Properties == nil {
		result.Properties = make(map[string]*Schema)
	}
	result.Properties[p.key] = &Schema{Type: "string", Enum: []any{p.tag}}
	result.Required = append(slices.Clone(source.Required), p.key)
	if source.MinProperties != nil {
		value := *source.MinProperties + 1
		result.MinProperties = &value
	}
	if source.MaxProperties != nil {
		value := *source.MaxProperties + 1
		result.MaxProperties = &value
	}
	result.AllOf = p.objects(source.AllOf)
	result.AnyOf = p.objects(source.AnyOf)
	result.OneOf = p.objects(source.OneOf)
	if source.Not != nil {
		result.Not = p.object(source.Not)
	}
	projectTaggedFormValues(&result, func(value any) any {
		return taggedFormObjectValue(value, p.key, p.tag)
	})
	return &result
}

func (p *taggedFormProjection) objects(source []*Schema) []*Schema {
	if source == nil {
		return nil
	}
	result := make([]*Schema, len(source))
	for index, schema := range source {
		result[index] = p.object(schema)
	}
	return result
}

func (p *taggedFormProjection) component(original, role string, project func(*Schema) *Schema) string {
	key := role + ":" + original
	if name := p.components[key]; name != "" {
		return toRef(name)
	}
	a := p.analyzer
	fingerprint := fingerprintString("tagged-form:" + role + ":" + original + ":" + p.key + ":" + p.tag)
	name := a.Uniquify(original, fingerprint)
	p.components[key] = name
	placeholder := &Schema{}
	setAcquisitionEntry(a, a.schemas, name, placeholder)
	setAcquisitionEntry(a, a.schemaFingerprints, name, fingerprint)
	naming := a.schemaNames[original]
	if naming.desired == "" {
		naming = representationComponentName{desired: original, logical: original, baseline: original}
	}
	naming.json = false
	setAcquisitionEntry(a, a.schemaNames, name, naming)
	// Prefer the existing JSON spelling when both wire representations occur.
	naming.json = true
	setAcquisitionEntry(a, a.schemaNames, original, naming)
	*placeholder = *project(a.schemas[original])
	return toRef(name)
}

func taggedFormObjectValue(value any, key, tag string) any {
	object, ok := value.(map[string]any)
	if !ok {
		return value
	}
	result := maps.Clone(object)
	result[key] = tag
	return result
}

func projectTaggedFormValues(schema *Schema, project func(any) any) {
	schema.DefaultValue = project(schema.DefaultValue)
	schema.Example = project(schema.Example)
	if schema.Enum != nil {
		schema.Enum = slices.Clone(schema.Enum)
		for index, value := range schema.Enum {
			schema.Enum[index] = project(value)
		}
	}
	if schema.Examples != nil {
		schema.Examples = slices.Clone(schema.Examples)
		for index, value := range schema.Examples {
			schema.Examples[index] = project(value)
		}
	}
}

func taggedFormValue(value any, key, valueKey string, branches []*expr.HTTPFormTaggedBranch) any {
	object, ok := value.(map[string]any)
	if !ok {
		return value
	}
	tag, _ := object[key].(string)
	for _, branch := range branches {
		if branch.Tag == tag && branch.Object != nil {
			return taggedFormObjectValue(object[valueKey], key, tag)
		}
	}
	return value
}

func projectTaggedFormMediaExamples(media *MediaType, form *expr.HTTPFormEncoding) {
	if form == nil || len(form.TaggedBranches) == 0 {
		return
	}
	union := expr.AsUnion(form.Body.Type)
	project := func(value any) any {
		return taggedFormValue(value, union.GetTypeKey(), union.GetValueKey(), form.TaggedBranches)
	}
	media.Example = project(media.Example)
	media.Examples = maps.Clone(media.Examples)
	for name, reference := range media.Examples {
		if reference == nil || reference.Value == nil {
			continue
		}
		copy := *reference
		example := *reference.Value
		example.Value = project(example.Value)
		example.DataValue = project(example.DataValue)
		copy.Value = &example
		media.Examples[name] = &copy
	}
}
