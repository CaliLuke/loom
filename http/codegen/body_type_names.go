package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type (
	bodyTypeName struct {
		body *expr.AttributeExpr
		key  bodyTypeKey
		name string
	}

	bodyTypeKey struct {
		id    string
		name  string
		union string
	}
)

// nameBodyTypes separates endpoint body identities from nested user types
// before the hash-based type and layout registries see them. These attributes
// belong to the copied Go transport IR; evaluated design names stay intact.
func nameBodyTypes(endpoints []*transportir.Endpoint) {
	bodies := endpointBodyTypeNames(endpoints)
	scope, nested := bodyTypeNameScope(bodies)
	allocated := make(map[bodyTypeKey]string)
	claimed := make(map[string]bool)
	for _, body := range bodies {
		if body.key == (bodyTypeKey{}) {
			continue
		}
		name, exists := allocated[body.key]
		if !exists {
			name = body.name
			if nested[name] || claimed[name] {
				name = scope.Unique(name)
			}
			allocated[body.key] = name
			claimed[name] = true
		}
		if name == body.name {
			continue
		}
		// Copies retain graph sharing, so renaming the root also renames its
		// recursive references. A nested authored type may have the same public
		// ID and name but a distinct copy identity; it must retain its name.
		switch actual := body.body.Type.(type) {
		case expr.UserType:
			actual.Rename(name)
		case *expr.Union:
			actual.TypeName = name
		}
	}
}

// endpointBodyTypeNames collects body roots in stable endpoint order, including
// anonymous collections whose nested types occupy the same Go namespace.
func endpointBodyTypeNames(endpoints []*transportir.Endpoint) []bodyTypeName {
	var bodies []bodyTypeName
	add := func(body *expr.AttributeExpr) {
		if body == nil || body.Type == expr.Empty {
			return
		}
		bodies = append(bodies, bodyTypeName{body: body, key: bodyTypeIdentity(body.Type), name: codegen.Goify(body.Type.Name(), true)})
	}
	for _, endpoint := range endpoints {
		add(endpoint.Request.Body)
		add(endpoint.Request.StreamingBody)
		for _, response := range endpoint.Response.Responses {
			add(response.Body)
		}
		for _, response := range endpoint.Response.ErrorResponses {
			add(response.Body)
		}
	}
	return bodies
}

// bodyTypeNameScope reserves existing names without claiming their identities.
func bodyTypeNameScope(bodies []bodyTypeName) (*codegen.NameScope, map[string]bool) {
	// Reserve every existing name first so a suffix cannot steal the name of
	// a later body or nested type. The allocation scope is separate from the
	// rendering scope: the latter must hash the final transport names.
	scope := codegen.NewNameScope()
	reserved := make(map[string]bool)
	reserve := func(name string) {
		if !reserved[name] {
			scope.Unique(name)
			reserved[name] = true
		}
	}
	nested := make(map[string]bool)
	for _, body := range bodies {
		if body.key != (bodyTypeKey{}) {
			reserve(body.name)
		}
		walkBodyTypes(body.body.Type, func(dt expr.DataType) {
			if key := bodyTypeIdentity(dt); key != (bodyTypeKey{}) && dt != body.body.Type {
				name := codegen.Goify(dt.Name(), true)
				nested[name] = true
				reserve(name)
			}
		})
	}
	return scope, nested
}

// bodyTypeIdentity keeps copies of a body together while distinguishing the
// synthesized endpoint wrapper from an authored type with the same name.
func bodyTypeIdentity(dt expr.DataType) bodyTypeKey {
	switch actual := dt.(type) {
	case expr.UserType:
		return bodyTypeKey{id: actual.ID(), name: actual.Name()}
	case *expr.Union:
		return bodyTypeKey{union: actual.Hash()}
	default:
		return bodyTypeKey{}
	}
}

// walkBodyTypes uses object identity because name hashes are not unique until
// nameBodyTypes has resolved collisions. Pointer identity also bounds cycles.
func walkBodyTypes(dt expr.DataType, visit func(expr.DataType)) {
	seen := make(map[expr.DataType]bool)
	var walk func(expr.DataType)
	walk = func(dt expr.DataType) {
		if dt == nil || seen[dt] {
			return
		}
		seen[dt] = true
		visit(dt)
		switch actual := dt.(type) {
		case expr.UserType:
			if union, ok := actual.Attribute().Type.(*expr.Union); ok {
				// A named union declares its wrapper's name, not a second
				// type under the underlying union's name.
				for _, branch := range union.Values {
					walk(branch.Attribute.Type)
				}
			} else {
				walk(actual.Attribute().Type)
			}
		case *expr.Object:
			for _, field := range *actual {
				walk(field.Attribute.Type)
			}
		case *expr.Union:
			for _, branch := range actual.Values {
				walk(branch.Attribute.Type)
			}
		case *expr.Array:
			walk(actual.ElemType.Type)
		case *expr.Map:
			walk(actual.KeyType.Type)
			walk(actual.ElemType.Type)
		}
	}
	walk(dt)
}
