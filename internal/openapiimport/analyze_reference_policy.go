package openapiimport

import (
	"slices"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

type (
	// policyCarrier is the Loom type identity of a schema that constrains
	// object members, mirroring the composition rules of Extend.
	policyCarrier struct {
		// neutral reports a schema without keywords, which Loom renders as an
		// unvalidated Any that admits every value and constrains nothing.
		neutral bool
		// identity names the Loom type, or is empty when the importer cannot
		// prove that two schemas share it.
		identity string
		// underlying is the identity of a referenced schema's own type, which
		// a named alias refines.
		underlying string
		// kind is the JSON value kind: boolean, number, string, array or
		// object, or empty when the schema has no single kind.
		kind string
		// nullable reports whether the schema admits null.
		nullable bool
	}
)

// policyIntersection reports why Loom cannot compose the typed
// additionalProperties of a reference sibling with the target's, or "" when
// the sibling's policy refines the target's or their JSON kinds are disjoint,
// which composes to a closed object. The sibling's policy is authored on the
// derived type, so like an authored Extend field it must refine the inherited
// type and must not admit null when the inherited policy does not.
func policyIntersection(local, target *base.SchemaProxy) string {
	if local == nil || target == nil {
		return ""
	}
	left, right := schemaPolicyCarrier(local), schemaPolicyCarrier(target)
	switch {
	case left.neutral || right.neutral:
		return ""
	case left.refines(right):
		if left.nullable && !right.nullable {
			return "typed additionalProperties cannot admit null when the referenced object's additionalProperties does not"
		}
		return ""
	case left.kind != "" && right.kind != "" && left.kind != right.kind:
		if left.nullable && right.nullable {
			return "typed additionalProperties intersections that admit only null are not representable"
		}
		return ""
	}
	return "typed additionalProperties intersections whose types overlap without one refining the other are not representable"
}

// policyMemberRefinement reports why a member that only the sibling declares
// cannot refine the target's typed additionalProperties, or "" when it does.
func policyMemberRefinement(member, policy *base.SchemaProxy) string {
	if member == nil || policy == nil {
		return ""
	}
	own, inherited := schemaPolicyCarrier(member), schemaPolicyCarrier(policy)
	if inherited.neutral || own.refines(inherited) && (!own.nullable || inherited.nullable) {
		return ""
	}
	return "a property that the referenced object does not declare must refine its typed additionalProperties"
}

// schemaPolicyCarrier derives the Loom type identity of a schema as the
// importer renders it.
func schemaPolicyCarrier(proxy *base.SchemaProxy) policyCarrier {
	if proxy == nil {
		return policyCarrier{neutral: true}
	}
	if proxy.IsReference() && !proxy.IsTransformedRefWithSiblings() {
		target := schemaPolicyCarrier(baseSchemaProxy(referenceSchemaShape(proxy, make(map[*base.Schema]bool))))
		return policyCarrier{identity: "ref:" + proxy.GetReference(), underlying: target.identity, kind: target.kind, nullable: target.nullable}
	}
	if proxy.IsReference() {
		return policyCarrier{}
	}
	schema := proxy.Schema()
	if schema == nil {
		return policyCarrier{}
	}
	node := semanticSchemaNode(proxy)
	if node != nil && len(node.Content) == 0 {
		return policyCarrier{neutral: true}
	}
	types := slices.DeleteFunc(slices.Clone(schema.Type), func(name string) bool { return name == "null" })
	carrier := policyCarrier{nullable: len(types) < len(schema.Type) || schema.Nullable != nil && *schema.Nullable}
	if len(types) != 1 {
		return carrier
	}
	switch types[0] {
	case "boolean":
		carrier.kind, carrier.identity = "boolean", "Boolean"
	case "integer":
		carrier.kind, carrier.identity = "number", integerSchemaExpression(schema.Format)
	case "number":
		carrier.kind, carrier.identity = "number", numberSchemaExpression(schema.Format)
	case "string":
		carrier.kind, carrier.identity = "string", stringRepresentation(schema.Format).typeName
	case "array":
		carrier.kind = "array"
		if schema.Items != nil && schema.Items.IsA() {
			if item := schemaPolicyCarrier(schema.Items.A); item.identity != "" && !item.nullable {
				carrier.identity = "ArrayOf(" + item.identity + ")"
			}
		}
	case "object":
		// Inline objects refine one another only field by field; the
		// importer does not prove that, so they share no identity.
		carrier.kind = "object"
		if orderedmap.Len(schema.Properties) == 0 && schema.AdditionalProperties == nil {
			carrier.identity = "MapOf(String, Any)"
		}
	}
	return carrier
}

// baseSchemaProxy wraps a resolved schema so that its carrier can be derived.
func baseSchemaProxy(schema *base.Schema) *base.SchemaProxy {
	if schema == nil {
		return nil
	}
	return base.CreateSchemaProxy(schema)
}

// refines reports whether a value of the carrier's type is a value of other's
// type in Loom: the same type, or a named alias of it.
func (c policyCarrier) refines(other policyCarrier) bool {
	if c.identity == "" {
		return false
	}
	return c.identity == other.identity || c.underlying != "" && c.underlying == other.identity
}

// memberCarriesOwnType reports whether a member's Loom type does not take the
// policy's carrier during refinement: a reference, or an array with items.
func memberCarriesOwnType(member *base.SchemaProxy) bool {
	if member.IsReference() {
		return true
	}
	schema := member.Schema()
	return schema != nil && slices.Contains(schema.Type, "array") && schema.Items != nil
}
