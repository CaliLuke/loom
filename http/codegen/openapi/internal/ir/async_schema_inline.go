package ir

import "github.com/CaliLuke/loom/expr"

type (
	asyncInlineShape struct {
		attribute *expr.AttributeExpr
		reference bool
	}
	asyncMaterialization struct {
		schema *Schema
		// structures pairs generated nullable and assertion wrappers with their
		// owned structural child. Sampling follows this correspondence rather
		// than treating an arbitrary allOf branch as the source fields.
		structures map[*Schema]*Schema
	}
	asyncMaterializer struct {
		constructions map[*Schema]schemaConstruction
		components    map[string]*Schema
		active        map[string]bool
		structures    map[*Schema]*Schema
	}
)

// materializeAsyncSchema combines the fully analyzed representation with the
// retained legacy inline/reference shape. The sampler controls only shape and
// example ownership; schema constraints always come from complete analysis.
func materializeAsyncSchema(prepared *asyncSchema, components map[string]*Schema) *asyncMaterialization {
	worker := &asyncMaterializer{constructions: prepared.constructions, components: components, active: make(map[string]bool), structures: make(map[*Schema]*Schema)}
	return &asyncMaterialization{schema: worker.prepared(prepared), structures: worker.structures}
}

func (m *asyncMaterializer) prepared(prepared *asyncSchema) *Schema {
	if len(prepared.projections) == 0 {
		return m.inline(prepared.schema, asyncInlineShape{attribute: prepared.sampler})
	}
	result := *prepared.schema
	result.OneOf = make([]*Schema, len(prepared.projections))
	for index, child := range prepared.projections {
		result.OneOf[index] = m.prepared(child)
	}
	return &result
}

// inline expands only nodes that the retained source shape inlines.
// Declared cuts and synthetic tagged-union envelope references keep their final
// component identity. Assertion siblings stay conjoined; annotations overlay
// the owned expansion without introducing a new assertion wrapper.
func (m *asyncMaterializer) inline(schema *Schema, shape asyncInlineShape) *Schema {
	if schema == nil {
		return nil
	}
	result := *schema
	var referenced *Schema
	_, named := asyncSamplerType(shape.attribute).(expr.UserType)
	cut := shape.reference || named
	if name, ok := schemaComponentName(schema.Ref); ok && !cut && !m.active[name] && m.components[name] != nil {
		m.active[name] = true
		base := m.inline(m.components[name], shape)
		delete(m.active, name)
		if asyncAnnotationOnly(schema) {
			return overlayAsyncAnnotations(base, schema)
		}
		result.Ref = ""
		referenced = base
	}
	child := func(value *Schema) *Schema {
		return m.inline(value, asyncInlineShape{})
	}
	result.Items = m.inline(result.Items, asyncSamplerChild(shape.attribute, "items", 0))
	result.ContentSchema, result.Not = child(result.ContentSchema), child(result.Not)
	m.properties(&result, shape)
	m.alternatives(&result, shape)
	for _, target := range []**BoolOrSchema{&result.AdditionalProperties, &result.UnevaluatedProperties} {
		if *target != nil {
			copied := **target
			copied.Schema = m.inline(copied.Schema, asyncSamplerChild(shape.attribute, "additional-properties", 0))
			*target = &copied
		}
	}
	if referenced != nil {
		result.AllOf = append([]*Schema{referenced}, result.AllOf...)
		m.structures[&result] = referenced
	} else if asyncNullableWrapper(&result) {
		m.structures[&result] = result.AnyOf[0]
	} else if construction, known := m.constructions[schema]; known && construction.kind == 0 && len(construction.slots) == 1 {
		slot := construction.slots[0]
		if slot.location == schemaAllOf {
			m.structures[&result] = slot.get(&result)
		}
	}
	return &result
}

// applyNullableSchema constructs exactly a value arm followed by a plain null
// arm. This shape belongs to complete constraint analysis, independently of
// flags on the legacy sampler, whose alias flattening can erase Nullable.
func asyncNullableWrapper(schema *Schema) bool {
	if len(schema.AnyOf) != 2 || schema.AnyOf[1] == nil || schema.AnyOf[1].Type != "null" {
		return false
	}
	null := *schema.AnyOf[1]
	null.Type = ""
	return null.Ref == "" && onlySchemaReference(&null)
}

// Properties correspond to authored fields; definitions are independent schema
// resources and carry no source materialization shape.
func (m *asyncMaterializer) properties(result *Schema, shape asyncInlineShape) {
	for _, target := range []*map[string]*Schema{&result.Properties, &result.Defs} {
		if *target == nil {
			continue
		}
		copied := make(map[string]*Schema, len(*target))
		for name, value := range *target {
			childShape := asyncInlineShape{}
			if target == &result.Properties {
				childShape = asyncSamplerChild(shape.attribute, name, 0)
			}
			copied[name] = m.inline(value, childShape)
		}
		*target = copied
	}
}

// Conjunctions and nullable alternatives preserve the occurrence shape. Union
// alternatives instead carry either their source branch or an envelope cut.
func (m *asyncMaterializer) alternatives(result *Schema, shape asyncInlineShape) {
	for _, target := range []*[]*Schema{&result.AllOf, &result.AnyOf, &result.OneOf} {
		if *target == nil {
			continue
		}
		copied := make([]*Schema, len(*target))
		for index, value := range *target {
			childShape := shape
			if target == &result.OneOf {
				childShape = asyncSamplerChild(shape.attribute, "branch", index)
			}
			copied[index] = m.inline(value, childShape)
		}
		*target = copied
	}
}

// Child shape follows the source occurrence. Tagged-union envelopes have no
// source attribute: their references are introduced by analysis and retained
// explicitly, independently of declaration names or component naming policy.
func asyncSamplerChild(sampler *expr.AttributeExpr, name string, index int) asyncInlineShape {
	switch actual := asyncSamplerType(sampler).(type) {
	case *expr.Object:
		for _, field := range *actual {
			if expr.JSONFieldName(expr.ElementName(field.Name), field.Attribute) == name {
				return asyncInlineShape{attribute: field.Attribute}
			}
		}
	case *expr.Array:
		if name == "items" {
			return asyncInlineShape{attribute: actual.ElemType}
		}
	case *expr.Map:
		if name == "additional-properties" {
			return asyncInlineShape{attribute: actual.ElemType}
		}
	case *expr.Union:
		if name == "branch" {
			if !actual.Untagged {
				return asyncInlineShape{reference: true}
			}
			branches := sortedUnionValues(actual)
			if index < len(branches) {
				return asyncInlineShape{attribute: branches[index].Attribute}
			}
		}
	}
	return asyncInlineShape{}
}

func asyncSamplerType(sampler *expr.AttributeExpr) expr.DataType {
	if sampler == nil {
		return nil
	}
	return sampler.Type
}

func asyncAnnotationOnly(schema *Schema) bool {
	copy := *schema
	copy.Title, copy.Description = "", ""
	copy.DefaultValue, copy.Example = nil, nil
	copy.ReadOnly, copy.WriteOnly, copy.Deprecated = false, false, false
	return onlySchemaReference(&copy)
}

func overlayAsyncAnnotations(base, local *Schema) *Schema {
	if local.Title != "" {
		base.Title = local.Title
	}
	if local.Description != "" {
		base.Description = local.Description
	}
	if local.DefaultValue != nil {
		base.DefaultValue = local.DefaultValue
	}
	if local.Example != nil {
		base.Example = local.Example
	}
	base.ReadOnly = base.ReadOnly || local.ReadOnly
	base.WriteOnly = base.WriteOnly || local.WriteOnly
	base.Deprecated = base.Deprecated || local.Deprecated
	return base
}
