package ir

import "github.com/CaliLuke/loom/expr"

type asyncSchema struct {
	constructions map[*Schema]schemaConstruction
	schema        *Schema
	sampler       *expr.AttributeExpr
	context       string
	projections   []*asyncSchema
}

// Async sampling retains its pre-representation input and path seeds until the
// example-consumer migration. Constraint analysis never consumes this flattened
// input and suppresses sampling; only the owned inline result is sampled here.
func applyPreparedAsyncExamples(a *Analyzer, schema *Schema, prepared *asyncSchema, structures map[*Schema]*Schema) {
	if len(prepared.projections) > 0 {
		for index, child := range prepared.projections {
			applyPreparedAsyncExamples(a, schema.OneOf[index], child, structures)
		}
		return
	}
	applyAsyncExamples(a, schema, prepared.sampler, prepared.context, structures)
}

func applyAsyncExamples(a *Analyzer, schema *Schema, attr *expr.AttributeExpr, context string, structures map[*Schema]*Schema) {
	if schema == nil || attr == nil || schema.Ref != "" {
		return
	}
	target := schema
	for structures[target] != nil {
		target = structures[target]
	}
	switch actual := attr.Type.(type) {
	case *expr.Array:
		applyAsyncExamples(a, target.Items, actual.ElemType, childExampleContext(context, "items"), structures)
	case *expr.Map:
		if target.AdditionalProperties != nil {
			applyAsyncExamples(a, target.AdditionalProperties.Schema, actual.ElemType, childExampleContext(context, "additional-properties"), structures)
		}
	case *expr.Object:
		for _, field := range *actual {
			name := expr.JSONFieldName(expr.ElementName(field.Name), field.Attribute)
			applyAsyncExamples(a, target.Properties[name], field.Attribute, childExampleContext(context, "property", name), structures)
		}
	case *expr.Union:
		if actual.Untagged {
			for index, branch := range sortedUnionValues(actual) {
				if index < len(target.OneOf) {
					applyAsyncExamples(a, target.OneOf[index], branch.Attribute, childExampleContext(context, "branch", expr.UnionVariantTag(branch)), structures)
				}
			}
		}
	}
	schema.Example = nil
	a.applySchemaExample(schema, attr, context)
}

func asyncSamplerAttribute(attr *expr.AttributeExpr) *expr.AttributeExpr {
	if attr == nil {
		return nil
	}
	cloned := expr.DupAtt(attr)
	inlineAsyncSamplerTypes(cloned, make(map[string]struct{}))
	return cloned
}

func inlineAsyncSamplerTypes(attr *expr.AttributeExpr, seen map[string]struct{}) {
	if attr == nil || attr.Type == nil || attr.Type == expr.Empty {
		return
	}
	switch actual := attr.Type.(type) {
	case expr.UserType:
		key := actual.Hash()
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		inlined := componentAttribute(attr, actual)
		*attr = *inlined
		inlineAsyncSamplerTypes(attr, seen)
		delete(seen, key)
	case *expr.Array:
		inlineAsyncSamplerTypes(actual.ElemType, seen)
	case *expr.Map:
		inlineAsyncSamplerTypes(actual.ElemType, seen)
	case *expr.Object:
		for _, named := range *actual {
			inlineAsyncSamplerTypes(named.Attribute, seen)
		}
	case *expr.Union:
		for _, named := range actual.Values {
			inlineAsyncSamplerTypes(named.Attribute, seen)
		}
	}
}
