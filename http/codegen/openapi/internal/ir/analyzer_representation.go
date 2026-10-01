package ir

import (
	"encoding/json/v2"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/byteschema"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

type representationComponentName struct {
	desired  string
	logical  string
	baseline string
	explicit bool
	json     bool
}

// analyzeSchemaPositions carries structural and prepared-example identities
// independently. Recursive calls select both actual child handles before
// entering here.
func (a *Analyzer) analyzeSchemaPositions(
	attr *expr.AttributeExpr,
	context string,
	plan expr.ValuePlanNode,
	examplePlan expr.ValuePlanNode,
	noRef ...bool,
) (result *Schema) {
	restore := a.schemaPlanScope(plan, examplePlan)
	defer restore()
	if acquisition := a.asyncAcquisition; acquisition != nil {
		key := asyncBaselineKey{
			plan: plan, examplePlan: examplePlan, position: acquisition.shape.attribute,
			context: context, reference: acquisition.shape.reference, noRef: len(noRef) > 0,
		}
		if cached := acquisition.schemas[key]; cached != nil {
			return cached
		}
		reserved := &Schema{}
		acquisition.schemas[key] = reserved
		defer func() {
			if result == nil {
				return
			}
			*reserved = *result
			if construction, found := a.constructions[result]; found {
				a.constructions[reserved] = construction
			}
			result = reserved
		}()
	}
	constructing := a.constructingBaseline
	a.constructingBaseline = true
	baseline := func() *Schema {
		defer func() {
			a.constructingBaseline = constructing
		}()
		return a.analyzeSchema(attr, context, noRef...)
	}()
	if constructing || !plan.Valid() {
		return baseline
	}
	return a.projectBaseline(baseline, attr, plan)
}

func (a *Analyzer) analyzeUnderlyingSchema(attr *expr.AttributeExpr, context string, noRef ...bool) *Schema {
	restore := a.schemaPlanScope(a.plan.Underlying(), a.examplePlan.Underlying())
	defer restore()
	return a.analyzeSchema(attr, context, noRef...)
}

func (a *Analyzer) schemaPlanScope(plan, examplePlan expr.ValuePlanNode) func() {
	previous, previousExample := a.plan, a.examplePlan
	restoreShape := a.asyncShapeScope(plan)
	a.plan, a.examplePlan = plan, examplePlan
	return func() {
		a.plan, a.examplePlan = previous, previousExample
		restoreShape()
	}
}

func (a *Analyzer) byteProjection(attr *expr.AttributeExpr, plan expr.ValuePlanNode) string {
	if a.constructingBaseline {
		return ""
	}
	return byteProjectionIdentity(attr, plan)
}

func componentPublicName(attr *expr.AttributeExpr, typ expr.UserType) string {
	meta, canonical := schemaTypeNaming(attr, typ)
	if canonical && meta != "" {
		return meta
	}
	return codegen.Goify(schemaTypeName(typ, meta), true)
}

func representationRoot(attribute *expr.AttributeExpr, target *transportir.ValueTarget) expr.ValuePlanNode {
	if target == nil {
		return expr.ValuePlanNode{}
	}
	if target.Error != nil {
		panic(fmt.Errorf("OpenAPI representation: %w", target.Error))
	}
	node := target.Plan.Root()
	// HTTP's inline body wrapper is intentionally absent from the schema graph.
	// Its immediate child is paired here rather than guessed by structural hashes.
	if attribute != nil && node.Valid() {
		captured := node.Attribute()
		if expr.UnwrapInlineHTTPBody(captured) != captured && expr.UnwrapInlineHTTPBody(attribute) == attribute {
			node = node.Underlying()
		}
	}
	return node
}

func byteConstraintSchema(constraint byteschema.Constraint) *Schema {
	if constraint.Unsatisfiable {
		return &Schema{Not: &Schema{}}
	}
	gate := &Schema{Not: &Schema{Pattern: constraint.ForbiddenPattern}}
	for _, branch := range constraint.Branches {
		minimum := branch.MinLength
		gate.AnyOf = append(gate.AnyOf, &Schema{Pattern: branch.Pattern, MinLength: &minimum, MaxLength: branch.MaxLength})
	}
	return gate
}

func applyByteGate(schema, gate *Schema) {
	if len(schema.AnyOf) > 0 {
		nullable := false
		for _, child := range schema.AnyOf {
			if child.Type == "null" {
				nullable = true
			}
		}
		if nullable {
			for _, child := range schema.AnyOf {
				if child.Type != "null" {
					applyByteGate(child, gate)
				}
			}
			return
		}
	}
	schema.Not = gate.Not
	if len(schema.AnyOf) == 0 {
		schema.AnyOf = gate.AnyOf
	} else {
		schema.AllOf = append(schema.AllOf, &Schema{AnyOf: gate.AnyOf})
	}
}

func onlySchemaReference(schema *Schema) bool {
	copy := *schema
	copy.Ref = ""
	return reflect.DeepEqual(copy, Schema{})
}

func byteOccurrenceLayers(attribute *expr.AttributeExpr, node expr.ValuePlanNode) ([]*expr.AttributeExpr, bool) {
	var layers []*expr.AttributeExpr
	seen := make(map[expr.UserType]bool)
	for attribute != nil {
		if !node.Valid() || node.Codec() != expr.ValueCodecJSON || encodingmeta.SchemaOverride(attribute.Meta) {
			return nil, false
		}
		layers = append(layers, attribute)
		typ, named := attribute.Type.(expr.UserType)
		if !named {
			return layers, attribute.Type == expr.Bytes
		}
		if seen[typ] {
			return nil, false
		}
		seen[typ] = true
		attribute, node = typ.Attribute(), node.Underlying()
	}
	return nil, false
}

func (a *Analyzer) logicalDeclaration(typ expr.UserType) string {
	if identity := a.plan.TargetDeclarationID(); identity != "" {
		return identity
	}
	return typ.ID()
}

// byteProjectionIdentity walks the complete paired occurrence graph before
// component reuse. Only effective schema changes enter this identity; raw and
// custom descendants keep their existing schema and naming identity.
func byteProjectionIdentity(attribute *expr.AttributeExpr, node expr.ValuePlanNode) string {
	active := make(map[expr.ValuePlanNode]bool)
	var visit func(*expr.AttributeExpr, expr.ValuePlanNode, string) string
	visit = func(attribute *expr.AttributeExpr, node expr.ValuePlanNode, path string) string {
		if attribute == nil || !node.Valid() || active[node] || byteSchemaExcluded(attribute) {
			return ""
		}
		active[node] = true
		defer delete(active, node)
		if layers, ok := byteOccurrenceLayers(attribute, node); ok {
			constraint := effectiveByteConstraint(attribute, layers, path)
			encoded, err := json.Marshal(constraint, json.Deterministic(true))
			if err != nil {
				panic(err)
			}
			return string(encoded)
		}
		var parts []string
		add := func(key string, child *expr.AttributeExpr, target expr.ValuePlanNode) {
			if value := visit(child, target, path+"/"+key); value != "" {
				parts = append(parts, key+":"+value)
			}
		}
		switch typ := attribute.Type.(type) {
		case expr.UserType:
			add("alias", typ.Attribute(), node.Underlying())
		case *expr.Array:
			add("items", typ.ElemType, node.Element())
		case *expr.Map:
			add("values", typ.ElemType, node.Element())
		case *expr.Object:
			for _, field := range *typ {
				for _, member := range node.Members() {
					if field.Name == member.Name && member.Visible {
						add(field.Name, field.Attribute, member.Node)
					}
				}
			}
		case *expr.Union:
			for _, branch := range typ.Values {
				for _, target := range node.Branches() {
					if expr.UnionVariantTag(branch) == target.Tag {
						add(target.Tag, branch.Attribute, target.Node)
					}
				}
			}
		}
		sort.Strings(parts)
		return strings.Join(parts, ";")
	}
	return visit(attribute, node, "$")
}

func effectiveByteConstraint(attribute *expr.AttributeExpr, layers []*expr.AttributeExpr, path string) byteschema.Constraint {
	bounds := make([]byteschema.Bounds, 0, len(layers))
	for _, layer := range layers {
		if layer.Validation != nil {
			bounds = append(bounds, byteschema.Bounds{Minimum: layer.Validation.MinLength, Maximum: layer.Validation.MaxLength})
		}
	}
	effective := byteschema.Intersect(bounds)
	constraint, err := byteschema.Project(effective.Minimum, effective.Maximum)
	if err != nil {
		panic(fmt.Errorf("OpenAPI byte schema %s at %s: %w", attribute.Type.Name(), path, err))
	}
	return constraint
}

// A schema override owns the entire scalar alias occurrence, not only the
// declaration layer carrying its metadata. Sibling object fields stay separate.
func byteSchemaExcluded(attribute *expr.AttributeExpr) bool {
	seen := make(map[expr.UserType]bool)
	excluded := false
	for attribute != nil {
		excluded = excluded || encodingmeta.SchemaOverride(attribute.Meta)
		typ, named := attribute.Type.(expr.UserType)
		if !named {
			return attribute.Type == expr.Bytes && excluded
		}
		if seen[typ] {
			return false
		}
		seen[typ] = true
		attribute = typ.Attribute()
	}
	return false
}
