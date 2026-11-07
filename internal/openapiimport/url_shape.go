package openapiimport

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/internal/httpencoding"
	"github.com/CaliLuke/loom/internal/openapiref"
)

func (p *documentPlanner) urlShape(schema *Schema) httpencoding.BranchShape {
	return p.urlBranchValue(schema, make(map[*Schema]bool))
}

func (p *documentPlanner) urlBranchValue(schema *Schema, active map[*Schema]bool) (result httpencoding.BranchShape) {
	nullable := schema != nil && schema.Nullable
	sourceType := schemaEncodingType(schema, p.schemas)
	defer func() {
		result.Nullable = result.Nullable || nullable
		result.SourceType = sourceType
	}()
	unsupported := httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape}
	aliases := make(map[*Schema]bool)
	for schema != nil && schema.Ref != "" {
		if aliases[schema] {
			return unsupported
		}
		aliases[schema] = true
		nullable = nullable || schema.Nullable
		name, err := openapiref.Name(schema.Ref, "#/components/schemas/")
		if err != nil {
			return unsupported
		}
		schema = p.schemas[name].Schema
		if sourceType == "" {
			sourceType = schemaEncodingType(schema, p.schemas)
		}
	}
	if schema == nil || active[schema] {
		return unsupported
	}
	nullable = nullable || schema.Nullable
	if schema.Unconstrained {
		return httpencoding.BranchShape{Shape: httpencoding.AnyValue}
	}
	active[schema] = true
	defer delete(active, schema)
	if len(schema.OneOf) > 0 {
		shape := httpencoding.BranchShape{Shape: httpencoding.Union}
		for _, branch := range schema.OneOf {
			shape.Branches = append(shape.Branches, p.urlBranchValue(branch, active))
		}
		return shape
	}
	switch schema.Type {
	case "string", "boolean", "integer", "number":
		return httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: importedValueDomain(schema), Binary: schema.Type == "string" && stringRepresentation(schema.Format).raw}
	case "array":
		element := p.urlBranchValue(schema.Items, active)
		if element.Shape == httpencoding.Scalar && !element.Nullable {
			return httpencoding.BranchShape{Shape: httpencoding.Sequence, Domain: element.Domain, Items: &element}
		}
		unsupported.Items = &element
		return unsupported
	case "object":
		return p.urlObjectShape(schema, active)
	default:
		if len(schema.Bases) > 0 {
			return p.urlObjectShape(schema, active)
		}
		return unsupported
	}
}

func (p *documentPlanner) urlObjectShape(schema *Schema, active map[*Schema]bool) httpencoding.BranchShape {
	representation := classifyObject(schema)
	if representation == objectFreeMap {
		return httpencoding.BranchShape{Shape: httpencoding.Nested}
	}
	if representation == objectTypedMap {
		value := p.urlBranchValue(schema.AdditionalProperties.Schema, active)
		if value.Shape == httpencoding.Scalar && !value.Nullable {
			return httpencoding.BranchShape{Shape: httpencoding.ScalarMap, Domain: value.Domain}
		}
		return httpencoding.BranchShape{Shape: httpencoding.Nested}
	}
	if objectAdmitsTypedMembers(schema, p.schemas) {
		return httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape}
	}
	shape := httpencoding.BranchShape{Shape: httpencoding.Object}
	properties := make(map[string]*Schema)
	p.formProperties(schema, properties, make(map[*Schema]bool))
	required := make(map[string]bool)
	p.urlRequired(schema, required, make(map[*Schema]bool))
	for _, name := range slices.Sorted(maps.Keys(properties)) {
		property := p.urlSchema(properties[name])
		value := p.urlBranchValue(properties[name], active)
		if value.Shape != httpencoding.Scalar || value.Nullable {
			return httpencoding.BranchShape{Shape: httpencoding.Nested}
		}
		shape.Properties = append(shape.Properties, httpencoding.PropertyFact{
			WireName: name, Required: required[name], Kind: importedScalarKind(property), Domain: value.Domain,
		})
	}
	return shape
}

func (p *documentPlanner) urlRequired(schema *Schema, required map[string]bool, seen map[*Schema]bool) {
	schema = p.urlSchema(schema)
	if schema == nil || seen[schema] {
		return
	}
	seen[schema] = true
	for _, name := range schema.Required {
		required[name] = true
	}
	for _, base := range schema.Bases {
		p.urlRequired(base, required, seen)
	}
}

func importedScalarKind(schema *Schema) httpencoding.ScalarKind {
	switch schema.Type {
	case "boolean":
		return httpencoding.Boolean
	case "integer":
		return httpencoding.Integer
	case "number":
		return httpencoding.Number
	default:
		if stringRepresentation(schema.Format).raw {
			return httpencoding.Bytes
		}
		return httpencoding.String
	}
}

func importedValueDomain(schema *Schema) httpencoding.ValueDomain {
	if schema.Type == "integer" || schema.Type == "boolean" || schema.Type == "string" && (schema.Format == "uuid" || schema.Format == "date") {
		return httpencoding.ValueDomain{Unreserved: true}
	}
	if len(schema.Enum) > 0 {
		for _, value := range schema.Enum {
			if strings.IndexFunc(fmt.Sprint(value), func(r rune) bool {
				return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-._~", r))
			}) >= 0 {
				return httpencoding.ValueDomain{}
			}
		}
		return httpencoding.ValueDomain{Unreserved: true}
	}
	return httpencoding.ValueDomain{}
}
