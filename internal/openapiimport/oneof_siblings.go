package openapiimport

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"

	"github.com/CaliLuke/loom/internal/openapiref"
)

type oneOfSiblingCheck struct {
	schema   *Schema
	source   *base.Schema
	path     string
	branches []*Schema
}

func (a *analyzer) normalizeOneOfSiblings(schema *Schema, source *base.Schema, path string, branches []*Schema) bool {
	fail := func(keyword string) bool {
		schema.compositionError = &Diagnostic{Code: "schema-composition", Path: path + "/" + keyword, Message: fmt.Sprintf("oneOf sibling %s is not redundant or has no faithful union representation (#1018)", keyword)}
		return false
	}
	var node *yaml.Node
	if source.GoLow() != nil {
		node = utils.ResolveMergeNodes(source.GoLow().RootNode)
	}
	if node == nil {
		return true
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		keyword := node.Content[index].Value
		switch keyword {
		case "oneOf", "title", "description", "example", "examples", "deprecated", "readOnly", "writeOnly", "default", "unevaluatedProperties":
			continue
		case "type":
			if schema.Nullable && !slices.Contains(source.Type, "null") {
				return fail(keyword)
			}
			for _, branch := range branches {
				if !a.oneOfTypeSubset(branch, source.Type, make(map[*Schema]bool)) {
					return fail(keyword)
				}
			}
		case "required":
			for _, branch := range branches {
				if !a.oneOfRequires(branch, source.Required, make(map[*Schema]bool)) {
					return fail(keyword)
				}
			}
		case "properties":
			for name, property := range source.Properties.FromOldest() {
				if hasSchemaAnnotations(property) {
					return fail("properties/" + escapeJSONPointer(name))
				}
				for _, branch := range source.OneOf {
					if !oneOfPropertyIncluded(branch, name, property, make(map[*base.Schema]bool)) {
						return fail("properties/" + escapeJSONPointer(name))
					}
				}
			}
		default:
			if !strings.HasPrefix(keyword, "x-") {
				return fail(keyword)
			}
		}
	}
	return true
}

func (a *analyzer) oneOfTypeSubset(schema *Schema, types []string, active map[*Schema]bool) bool {
	if schema == nil || active[schema] {
		return false
	}
	active[schema] = true
	defer delete(active, schema)
	if schema.Nullable && !slices.Contains(types, "null") {
		return false
	}
	if schema.Ref != "" {
		name, err := openapiref.Name(schema.Ref, "#/components/schemas/")
		return err == nil && a.oneOfTypeSubset(a.schemas[name], types, active)
	}
	if len(schema.OneOf) > 0 {
		for _, branch := range schema.OneOf {
			if !a.oneOfTypeSubset(branch, types, active) {
				return false
			}
		}
		return true
	}
	return slices.Contains(types, schema.Type) || schema.Type == "integer" && slices.Contains(types, "number")
}

func (a *analyzer) oneOfRequires(schema *Schema, required []string, active map[*Schema]bool) bool {
	if schema == nil || active[schema] {
		return false
	}
	active[schema] = true
	defer delete(active, schema)
	if schema.Ref != "" {
		name, err := openapiref.Name(schema.Ref, "#/components/schemas/")
		return err == nil && a.oneOfRequires(a.schemas[name], required, active)
	}
	if len(schema.OneOf) > 0 {
		for _, branch := range schema.OneOf {
			if !a.oneOfRequires(branch, required, active) {
				return false
			}
		}
		return true
	}
	if schema.Type != "object" {
		return false
	}
	for _, name := range required {
		if slices.Contains(schema.Required, name) {
			continue
		}
		inherited := false
		for _, base := range schema.Bases {
			if a.oneOfRequires(base, []string{name}, active) {
				inherited = true
				break
			}
		}
		if !inherited {
			return false
		}
	}
	return true
}

func oneOfPropertyIncluded(proxy *base.SchemaProxy, name string, wanted *base.SchemaProxy, active map[*base.Schema]bool) bool {
	if proxy == nil || proxy.Schema() == nil || active[proxy.Schema()] {
		return false
	}
	source := proxy.Schema()
	active[source] = true
	defer delete(active, source)
	if len(source.OneOf) > 0 {
		for _, branch := range source.OneOf {
			if !oneOfPropertyIncluded(branch, name, wanted, active) {
				return false
			}
		}
		return true
	}
	for _, part := range source.AllOf {
		if oneOfPropertyIncluded(part, name, wanted, active) {
			return true
		}
	}
	var property *base.SchemaProxy
	exists := false
	if source.Properties != nil {
		property, exists = source.Properties.Get(name)
	}
	if !exists {
		return source.AdditionalProperties != nil && source.AdditionalProperties.IsB() && !source.AdditionalProperties.B
	}
	have, want := make(map[string]any), make(map[string]any)
	haveNode, wantNode := semanticSchemaNode(property), semanticSchemaNode(wanted)
	if haveNode == nil || wantNode == nil || haveNode.Decode(&have) != nil || wantNode.Decode(&want) != nil {
		return false
	}
	for key, value := range want {
		if !reflect.DeepEqual(have[key], value) {
			return false
		}
	}
	return true
}

func hasSchemaAnnotations(proxy *base.SchemaProxy) bool {
	seen := make(map[*base.Schema]bool)
	var visit func(*base.SchemaProxy) bool
	visit = func(proxy *base.SchemaProxy) bool {
		if proxy == nil || proxy.Schema() == nil || seen[proxy.Schema()] {
			return false
		}
		source := proxy.Schema()
		seen[source] = true
		if source.Title != "" || source.Description != "" || source.Example != nil || len(source.Examples) > 0 || source.Deprecated != nil || source.ReadOnly != nil || source.WriteOnly != nil || orderedmap.Len(source.Extensions) > 0 {
			return true
		}
		for _, property := range source.Properties.FromOldest() {
			if visit(property) {
				return true
			}
		}
		for _, branches := range [][]*base.SchemaProxy{source.OneOf, source.AnyOf, source.AllOf} {
			for _, branch := range branches {
				if visit(branch) {
					return true
				}
			}
		}
		return source.Items != nil && source.Items.IsA() && visit(source.Items.A) || source.AdditionalProperties != nil && source.AdditionalProperties.IsA() && visit(source.AdditionalProperties.A)
	}
	return visit(proxy)
}
