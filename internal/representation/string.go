// Package representation identifies authored physical representations shared by
// code generators. Runtime codec selection belongs to the corresponding runtime
// capability helpers.
package representation

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

// CustomString reports a semantic String with authored custom physical storage.
// Explicit raw JSON keeps its own representation contract.
func CustomString(attribute *expr.AttributeExpr) bool {
	if attribute == nil {
		return false
	}
	typ := attribute.Type
	for {
		named, ok := typ.(expr.UserType)
		if !ok {
			break
		}
		typ = named.Attribute().Type
	}
	if typ.Kind() != expr.StringKind {
		return false
	}
	name := encodingmeta.GoType(attribute.Meta).Name
	return name != "" && name != "string" && name != "jsontext.Value"
}

// ContainsCustomString reports whether the occurrence's reachable storage graph
// contains an authored String representation. Cycles are visited once.
func ContainsCustomString(attribute *expr.AttributeExpr) bool {
	seen := make(map[expr.DataType]bool)
	var visit func(*expr.AttributeExpr) bool
	visit = func(current *expr.AttributeExpr) bool {
		if current == nil || current.Type == nil {
			return false
		}
		if CustomString(current) {
			return true
		}
		if seen[current.Type] {
			return false
		}
		seen[current.Type] = true
		switch actual := current.Type.(type) {
		case expr.UserType:
			return visit(actual.Attribute())
		case *expr.Object:
			for _, field := range *actual {
				if visit(field.Attribute) {
					return true
				}
			}
		case *expr.Array:
			return visit(actual.ElemType)
		case *expr.Map:
			return visit(actual.KeyType) || visit(actual.ElemType)
		case *expr.Union:
			for _, branch := range actual.Values {
				if visit(branch.Attribute) {
					return true
				}
			}
		}
		return false
	}
	return visit(attribute)
}
