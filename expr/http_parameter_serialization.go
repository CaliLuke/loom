package expr

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

// HTTPValueShape describes a non-null HTTP value independently of its Go carrier.
func HTTPValueShape(attribute *AttributeExpr) httpencoding.BranchShape {
	return httpValueShape(attribute, make(map[*AttributeExpr]bool))
}

// HTTPParameterSerialization reads occurrence-first parameter or encoding metadata.
// Prefix selects openapi: or a named openapi:encoding:field: namespace.
func HTTPParameterSerialization(attribute *AttributeExpr, prefix string) httpencoding.Serialization {
	meta := HTTPMediaMetadata(attribute)
	source := httpencoding.Serialization{}
	source.Style, _ = meta.Last(prefix + "style")
	source.MediaType, _ = meta.Last(prefix + "contentType")
	if raw, ok := meta.Last(prefix + "explode"); ok {
		value := raw != "false"
		source.Explode = &value
	}
	if raw, ok := meta.Last(prefix + "allowReserved"); ok {
		value := raw != "false"
		source.AllowReserved = &value
	}
	return source
}

func httpValueShape(attribute *AttributeExpr, active map[*AttributeExpr]bool) httpencoding.BranchShape {
	unsupported := httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape}
	if attribute == nil || attribute.Type == nil || active[attribute] || AllowsNull(attribute) {
		return unsupported
	}
	active[attribute] = true
	defer delete(active, attribute)
	typ := httpParameterBaseType(attribute.Type)
	if typ == nil {
		return unsupported
	}
	if _, ok := httpScalarKind(typ); ok {
		return httpencoding.BranchShape{Shape: httpencoding.Scalar, Domain: httpValueDomain(attribute)}
	}
	if array, ok := typ.(*Array); ok {
		element := httpValueShape(array.ElemType, active)
		if element.Shape == httpencoding.Scalar {
			return httpencoding.BranchShape{Shape: httpencoding.Sequence, Domain: element.Domain}
		}
		return unsupported
	}
	if mapping, ok := typ.(*Map); ok {
		key := httpValueShape(mapping.KeyType, active)
		value := httpValueShape(mapping.ElemType, active)
		if key.Shape == httpencoding.Scalar && value.Shape == httpencoding.Scalar {
			return httpencoding.BranchShape{Shape: httpencoding.ScalarMap, Keys: httpencoding.KeyDomain{Unreserved: key.Domain.Unreserved}, Domain: value.Domain}
		}
		return httpencoding.BranchShape{Shape: httpencoding.Nested}
	}
	if object, ok := typ.(*Object); ok {
		return httpObjectShape(attribute, object, active)
	}
	if union, ok := typ.(*Union); ok {
		if !union.Untagged {
			return httpencoding.BranchShape{Shape: httpencoding.Nested}
		}
		shape := httpencoding.BranchShape{Shape: httpencoding.Union}
		for _, branch := range union.Values {
			shape.Branches = append(shape.Branches, httpValueShape(branch.Attribute, active))
		}
		return shape
	}
	return unsupported
}

func httpObjectShape(attribute *AttributeExpr, object *Object, active map[*AttributeExpr]bool) httpencoding.BranchShape {
	if attribute.TypedObjectDeclaration() != nil {
		return httpencoding.BranchShape{Shape: httpencoding.UnsupportedShape}
	}
	shape := httpencoding.BranchShape{Shape: httpencoding.Object}
	for _, property := range *object {
		value := httpValueShape(property.Attribute, active)
		if value.Shape != httpencoding.Scalar {
			return httpencoding.BranchShape{Shape: httpencoding.Nested}
		}
		kind, ok := httpScalarKind(httpParameterBaseType(property.Attribute.Type))
		if !ok {
			return httpencoding.BranchShape{Shape: httpencoding.Nested}
		}
		shape.Properties = append(shape.Properties, httpencoding.PropertyFact{
			WireName: ElementName(property.Name), Required: attribute.IsRequired(property.Name) || attribute.IsRequired(AttributeName(property.Name)),
			Kind: kind, Domain: value.Domain,
		})
	}
	return shape
}

func httpParameterBaseType(typ DataType) DataType {
	seen := make(map[string]bool)
	for typ != nil {
		named, ok := typ.(UserType)
		if !ok {
			return typ
		}
		if seen[named.ID()] {
			return nil
		}
		seen[named.ID()] = true
		if named.Attribute() == nil {
			return nil
		}
		typ = named.Attribute().Type
	}
	return nil
}

func httpScalarKind(typ DataType) (httpencoding.ScalarKind, bool) {
	switch typ {
	case Boolean:
		return httpencoding.Boolean, true
	case Int, Int32, Int64, UInt, UInt32, UInt64:
		return httpencoding.Integer, true
	case Float32, Float64:
		return httpencoding.Number, true
	case String:
		return httpencoding.String, true
	case Bytes:
		return httpencoding.Bytes, true
	default:
		return 0, false
	}
}

func httpValueDomain(attribute *AttributeExpr) httpencoding.ValueDomain {
	kind, ok := httpScalarKind(httpParameterBaseType(attribute.Type))
	if !ok {
		return httpencoding.ValueDomain{}
	}
	if kind == httpencoding.Integer || kind == httpencoding.Boolean {
		return httpencoding.ValueDomain{Unreserved: true}
	}
	constraints, err := EffectiveConstraintsFor(attribute)
	if err != nil {
		return httpencoding.ValueDomain{}
	}
	if values, present := constraints.EnumCandidates(); present {
		for _, value := range values {
			if !httpUnreservedText(fmt.Sprint(value)) {
				return httpencoding.ValueDomain{}
			}
		}
		return httpencoding.ValueDomain{Unreserved: true}
	}
	if kind == httpencoding.String {
		for _, format := range constraints.Validation().Lowered().Formats() {
			if format == FormatUUID || format == FormatDate {
				return httpencoding.ValueDomain{Unreserved: true}
			}
		}
	}
	return httpencoding.ValueDomain{}
}

func httpUnreservedText(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-._~", r))
	}) == -1
}
