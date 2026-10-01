package ir

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type asyncSchema struct {
	constructions map[*Schema]schemaConstruction
	schema        *Schema
	attribute     *expr.AttributeExpr
	sampler       *expr.AttributeExpr
	context       string
	target        *transportir.ValueTarget
	projections   []*asyncSchema
}

// Custom async projection retains its original sampler and path seeds. Builtin
// examples attach retained values through captured target-plan positions.
func applyPreparedAsyncExamples(a *Analyzer, schema *Schema, prepared *asyncSchema, structures map[*Schema]*Schema) {
	if len(prepared.projections) > 0 {
		for index, child := range prepared.projections {
			applyPreparedAsyncExamples(a, schema.OneOf[index], child, structures)
		}
		return
	}
	if !a.customExampleValue && prepared.target != nil {
		previous := a.preparedExamples
		a.preparedExamples = prepared.target
		defer func() {
			a.preparedExamples = previous
		}()
		applyPreparedTargetExamples(a, schema, prepared.attribute, prepared.context,
			prepared.target.Plan.Root(), structures)
		return
	}
	applyAsyncExamples(a, schema, prepared.sampler, prepared.context, structures)
}

func applyPreparedTargetExamples(
	a *Analyzer,
	schema *Schema,
	attr *expr.AttributeExpr,
	context string,
	plan expr.ValuePlanNode,
	structures map[*Schema]*Schema,
) {
	if schema == nil || attr == nil || !plan.Valid() {
		return
	}
	target := schema
	for structures[target] != nil {
		target = structures[target]
	}
	canDescend := schema.Ref == "" || target != schema
	if canDescend {
		switch actual := attr.Type.(type) {
		case expr.UserType:
			applyPreparedTargetExamples(a, target, componentAttribute(attr, actual), context,
				plan.Underlying(), structures)
		case *expr.Array:
			applyPreparedTargetExamples(a, target.Items, actual.ElemType,
				childExampleContext(context, "items"), plan.Element(), structures)
		case *expr.Map:
			if target.AdditionalProperties != nil {
				applyPreparedTargetExamples(a, target.AdditionalProperties.Schema, actual.ElemType,
					childExampleContext(context, "additional-properties"), plan.Element(), structures)
			}
		case *expr.Object:
			for _, field := range *actual {
				name := expr.JSONFieldName(expr.ElementName(field.Name), field.Attribute)
				applyPreparedTargetExamples(a, target.Properties[name], field.Attribute,
					childExampleContext(context, "property", name), valuePlanMemberNode(plan, field.Name), structures)
			}
		case *expr.Union:
			if actual.Untagged {
				for index, branch := range sortedUnionValues(actual) {
					if index < len(target.OneOf) {
						applyPreparedTargetExamples(a, target.OneOf[index], branch.Attribute,
							childExampleContext(context, "branch", expr.UnionVariantTag(branch)),
							valuePlanBranchNode(plan, expr.UnionVariantTag(branch)), structures)
					}
				}
			}
		}
	}
	schema.Example = nil
	restore := a.schemaPlanScope(a.plan, plan)
	a.applySchemaExample(schema, attr, context)
	restore()
}

func valuePlanMemberNode(plan expr.ValuePlanNode, name string) expr.ValuePlanNode {
	for _, member := range plan.Members() {
		if member.Name == name {
			return member.Node
		}
	}
	return expr.ValuePlanNode{}
}

func valuePlanBranchNode(plan expr.ValuePlanNode, tag string) expr.ValuePlanNode {
	for _, branch := range plan.Branches() {
		if branch.Tag == tag {
			return branch.Node
		}
	}
	return expr.ValuePlanNode{}
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
