package representation

import "github.com/CaliLuke/loom/expr"

type (
	httpValueDeclarationKey struct {
		typ    expr.DataType
		branch bool
	}
	// httpValueGraph shares declarations only within the same decoder context.
	// Untagged alternatives inspect unknown root members before ordinary decoding.
	httpValueGraph struct {
		types map[httpValueDeclarationKey]expr.DataType
	}
)

func TargetGraph(target *expr.AttributeExpr) *expr.AttributeExpr {
	// DupAtt establishes immutable source ancestry on every copied occurrence.
	// The following shallow occurrence copies share only this owned snapshot's
	// metadata; NewValuePlan captures that state before the graph is discarded.
	owned := expr.DupAtt(target)
	graph := httpValueGraph{types: make(map[httpValueDeclarationKey]expr.DataType)}
	return graph.attribute(owned, false)
}

func (g *httpValueGraph) attribute(source *expr.AttributeExpr, branch bool) *expr.AttributeExpr {
	copy := *source
	copy.Type = g.declaration(source.Type, branch)
	return &copy
}

func (g *httpValueGraph) declaration(source expr.DataType, branch bool) expr.DataType {
	key := httpValueDeclarationKey{typ: source, branch: branch}
	if previous, found := g.types[key]; found {
		return previous
	}
	switch actual := source.(type) {
	case expr.UserType:
		copy := actual.Dup(nil)
		g.types[key] = copy
		copy.SetAttribute(g.attribute(actual.Attribute(), branch))
		return copy
	case *expr.Object:
		copy := make(expr.Object, 0, len(*actual))
		g.types[key] = &copy
		for _, field := range *actual {
			copy = append(copy, &expr.NamedAttributeExpr{Name: field.Name, Attribute: g.attribute(field.Attribute, false)})
		}
		return &copy
	case *expr.Array:
		copy := &expr.Array{NonNullableElems: actual.NonNullableElems}
		g.types[key] = copy
		copy.ElemType = g.attribute(actual.ElemType, false)
		return copy
	case *expr.Map:
		copy := &expr.Map{}
		g.types[key] = copy
		copy.KeyType, copy.ElemType = g.attribute(actual.KeyType, false), g.attribute(actual.ElemType, false)
		return copy
	case *expr.Union:
		copy := *actual
		copy.Values = make([]*expr.NamedAttributeExpr, 0, len(actual.Values))
		g.types[key] = &copy
		for _, alternative := range actual.Values {
			copy.Values = append(copy.Values, &expr.NamedAttributeExpr{Name: alternative.Name, Attribute: g.attribute(alternative.Attribute, actual.Untagged)})
		}
		return &copy
	default:
		return source
	}
}
