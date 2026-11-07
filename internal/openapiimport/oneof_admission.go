package openapiimport

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/openapiref"
)

type oneOfGrammarProjection struct {
	schemas map[string]NamedSchema
	named   map[string]*expr.UserTypeExpr
	seen    map[*Schema]*expr.AttributeExpr
}

func (p *documentPlanner) admitUntaggedOneOf(schema *Schema, path string) {
	projection := oneOfGrammarProjection{schemas: p.schemas, named: make(map[string]*expr.UserTypeExpr), seen: make(map[*Schema]*expr.AttributeExpr)}
	attribute := projection.attribute(schema)
	union := expr.AsUnion(attribute.Type)
	if errors := expr.ValidateUntaggedUnion(union); errors != nil && len(errors.Errors) > 0 {
		p.unsupported("schema-oneof-branch", path, errors.Error())
	}
}

func (p *oneOfGrammarProjection) attribute(schema *Schema) *expr.AttributeExpr {
	if schema == nil {
		return &expr.AttributeExpr{Type: expr.Any}
	}
	if attribute, exists := p.seen[schema]; exists {
		return attribute
	}
	attribute := &expr.AttributeExpr{Nullable: schema.Nullable}
	p.seen[schema] = attribute
	if schema.Ref != "" {
		attribute.Type = p.referenceType(schema.Ref)
		return attribute
	}
	if len(schema.OneOf) > 0 {
		union := &expr.Union{TypeName: "ImportedUnion", Untagged: len(schema.MediaTypes) == 0}
		attribute.Type = union
		for index, branch := range schema.OneOf {
			union.Values = append(union.Values, &expr.NamedAttributeExpr{Name: fmt.Sprintf("Branch%d", index+1), Attribute: p.attribute(branch)})
		}
		return attribute
	}
	if schema.Unconstrained {
		attribute.Type = expr.Any
		return attribute
	}
	switch schema.Type {
	case "boolean":
		attribute.Type = expr.Boolean
	case "integer":
		attribute.Type = expr.Int
	case "number":
		attribute.Type = expr.Float64
	case "string":
		attribute.Type = expr.String
	case "array":
		attribute.Type = &expr.Array{ElemType: p.attribute(schema.Items)}
	case "object":
		p.objectAttribute(schema, attribute)
	default:
		attribute.Type = expr.Any
	}
	return attribute
}

func (p *oneOfGrammarProjection) referenceType(ref string) expr.DataType {
	name, err := openapiref.Name(ref, "#/components/schemas/")
	if err != nil {
		return expr.Any
	}
	named, exists := p.named[name]
	if !exists {
		named = &expr.UserTypeExpr{TypeName: name}
		p.named[name] = named
		named.AttributeExpr = p.attribute(p.schemas[name].Schema)
	}
	return named
}

func (p *oneOfGrammarProjection) objectAttribute(schema *Schema, attribute *expr.AttributeExpr) {
	representation := classifyObject(schema)
	if representation == objectFreeMap || representation == objectTypedMap {
		value := &expr.AttributeExpr{Type: expr.Any}
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
			value = p.attribute(schema.AdditionalProperties.Schema)
		}
		attribute.Type = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: value}
		return
	}
	object := expr.Object{}
	attribute.Type = &object
	for _, property := range schema.Properties {
		object = append(object, &expr.NamedAttributeExpr{Name: property.Name, Attribute: p.attribute(property.Schema)})
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		attribute.AdditionalAttributes = p.attribute(schema.AdditionalProperties.Schema)
	}
}
