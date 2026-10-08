package codegen

import (
	"slices"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/uniongen"
	"github.com/CaliLuke/loom/internal/unionjson"
)

// makeHTTPType traverses the attribute recursively and performs these actions:
//
// * removes aliased user type by replacing them with the underlying type.
// * changes unions into structs with Type and Value fields.
func makeHTTPType(att *expr.AttributeExpr) *expr.AttributeExpr {
	if att == nil {
		return nil
	}
	att = expr.DupAtt(att)
	return makeHTTPTypeRecursive(att, make(map[string]struct{}))
}

// makeHTTPMappedType keeps mapping defaults and validation while taking Go
// representation metadata from the service field it encodes or decodes.
func makeHTTPMappedType(mapped, service *expr.AttributeExpr) *expr.AttributeExpr {
	attr := *mapped
	attr.Meta = service.Meta
	return makeHTTPType(&attr)
}

func makeHTTPTypeRecursive(att *expr.AttributeExpr, seen map[string]struct{}) *expr.AttributeExpr {
	constraints := selectedTransportConstraints(att)
	defaultValue, hasDefault := constraints.Default()
	switch dt := att.Type.(type) {
	case expr.UserType:
		_, result := dt.(*expr.ResultTypeExpr)
		alias := !result && !expr.IsObject(dt) && !expr.IsUnion(dt)
		if alias {
			att.Type = dt.Attribute().Type
			att.Validation = constraints.Validation().Lowered()
			if att.Validation.HasRequiredOnly() && len(att.Validation.Required) == 0 {
				att.Validation = nil
			}
			if hasDefault {
				att.DefaultValue = defaultValue
			}
			att.UserExamples = dt.Attribute().UserExamples
		}
		if _, ok := seen[dt.ID()]; ok {
			return att
		}
		seen[dt.ID()] = struct{}{}
		dt.SetAttribute(makeHTTPTypeRecursive(dt.Attribute(), seen))
		if alias {
			// Use the fully normalized type without replacing the outer
			// attribute's default, examples, or validation with an inner alias's.
			att.Type = dt.Attribute().Type
		}
	case *expr.Array:
		dt.ElemType = makeHTTPTypeRecursive(dt.ElemType, seen)
	case *expr.Map:
		dt.KeyType = makeHTTPTypeRecursive(dt.KeyType, seen)
		dt.ElemType = makeHTTPTypeRecursive(dt.ElemType, seen)
	case *expr.Object:
		obj := make(expr.Object, len(*dt))
		for i, nat := range *dt {
			obj[i] = &expr.NamedAttributeExpr{Name: nat.Name, Attribute: makeHTTPTypeRecursive(nat.Attribute, seen)}
		}
		att.Type = &obj
	case *expr.Union:
	}
	return att
}

func selectedTransportConstraints(att *expr.AttributeExpr) expr.EffectiveConstraints {
	constraints, err := expr.EffectiveConstraintsFor(att)
	if err != nil {
		panic(codegen.NewError(nil, att, err))
	}
	return constraints
}

// collectUserTypes calls cb once per generated user type in dt. Generated body
// variants can share a design identifier, so traversal uses the same type hashes
// as NameScope and the physical layout registry.

func collectUnionBranchUserTypes(att *expr.AttributeExpr, hashes map[string]struct{}) {
	collectUnionBranchUserTypesSeen(att, hashes, make(map[string]struct{}))
}

func containsUntaggedUnion(att *expr.AttributeExpr) bool {
	if att == nil || att.Type == expr.Empty {
		return false
	}
	found := false
	err := codegen.Walk(att, func(current *expr.AttributeExpr) error {
		if union := expr.AsUnion(current.Type); union != nil && union.Untagged {
			found = true
		}
		return nil
	})
	if err != nil {
		panic(codegen.NewError(nil, att, err))
	}
	return found
}

func collectUnionBranchUserTypesSeen(att *expr.AttributeExpr, hashes, seen map[string]struct{}) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	switch actual := att.Type.(type) {
	case expr.UserType:
		if _, ok := seen[actual.Hash()]; ok {
			return
		}
		seen[actual.Hash()] = struct{}{}
		collectUnionBranchUserTypesSeen(actual.Attribute(), hashes, seen)
	case *expr.Object:
		for _, nat := range *actual {
			collectUnionBranchUserTypesSeen(nat.Attribute, hashes, seen)
		}
	case *expr.Array:
		collectUnionBranchUserTypesSeen(actual.ElemType, hashes, seen)
	case *expr.Map:
		collectUnionBranchUserTypesSeen(actual.KeyType, hashes, seen)
		collectUnionBranchUserTypesSeen(actual.ElemType, hashes, seen)
	case *expr.Union:
		for _, nat := range actual.Values {
			representation.WalkUserTypes(nat.Attribute.Type, func(ut expr.UserType) {
				hashes[ut.Hash()] = struct{}{}
			})
			collectUnionBranchUserTypesSeen(nat.Attribute, hashes, seen)
		}
	}
}

func (sds *ServicesData) collectEndpointUnionTypes(serviceName string, endpoints []*transportir.Endpoint, scope *codegen.NameScope) []*uniongen.Type {
	unionByName := make(map[string]*uniongen.Type)
	seenUnionTypes := make(map[string]struct{})
	closed, _ := sds.ServicesData.Root.API.Meta.Last("openapi:closed-objects")
	for _, endpoint := range endpoints {
		collectHTTPUnionTypes(endpoint.Request.Body, scope, unionByName, seenUnionTypes, closed == "true")
		if hasHTTPStreamingBody(endpoint) {
			collectHTTPUnionTypes(endpoint.Request.StreamingBody, scope, unionByName, seenUnionTypes, closed == "true")
		}
		if endpoint.Response.Result != nil {
			md := sds.ServicesData.Get(serviceName).Method(endpoint.MethodName)
			for _, response := range endpoint.Response.Responses {
				body := effectiveClientResponseBody(response.Body, endpoint.Response.Result, md)
				collectHTTPUnionTypes(body, scope, unionByName, seenUnionTypes, closed == "true")
			}
		}
		for _, response := range endpoint.Response.ErrorResponses {
			collectHTTPUnionTypes(response.Body, scope, unionByName, seenUnionTypes, closed == "true")
		}
	}
	unions := make([]*uniongen.Type, 0, len(unionByName))
	for _, union := range unionByName {
		unions = append(unions, union)
	}
	sort.Slice(unions, func(i, j int) bool {
		return unions[i].Name < unions[j].Name
	})
	return unions
}

func collectHTTPUnionTypes(att *expr.AttributeExpr, scope *codegen.NameScope, unions map[string]*uniongen.Type, seen map[string]struct{}, closedObjects bool) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	switch dt := att.Type.(type) {
	case expr.UserType:
		// The body types of a result type, such as the response body of
		// the result type and the element of a collection of it, share
		// the identifier of the result type but hold different branch
		// types. Visit each Go type, which the scope names after the hash.
		if _, ok := seen[dt.Hash()]; ok {
			return
		}
		seen[dt.Hash()] = struct{}{}
		if union := expr.AsUnion(dt.Attribute().Type); union != nil {
			// References use the enclosing type's allocated Go name. Distinct
			// declarations can share an inner union shape, so that shape cannot
			// be the deduplication key.
			name := scope.GoTypeName(&expr.AttributeExpr{Type: dt})
			if _, ok := unions[name]; !ok {
				unions[name] = buildHTTPUnionTypeData(union, scope, closedObjects, name)
			}
			for _, nat := range union.Values {
				collectHTTPUnionTypes(nat.Attribute, scope, unions, seen, closedObjects)
			}
			return
		}
		collectHTTPUnionTypes(dt.Attribute(), scope, unions, seen, closedObjects)
	case *expr.Object:
		for _, nat := range sortedNamedAttributes(*dt) {
			collectHTTPUnionTypes(nat.Attribute, scope, unions, seen, closedObjects)
		}
	case *expr.Array:
		collectHTTPUnionTypes(dt.ElemType, scope, unions, seen, closedObjects)
	case *expr.Map:
		collectHTTPUnionTypes(dt.KeyType, scope, unions, seen, closedObjects)
		collectHTTPUnionTypes(dt.ElemType, scope, unions, seen, closedObjects)
	case *expr.Union:
		name := scope.GoTypeName(&expr.AttributeExpr{Type: dt})
		if _, ok := unions[name]; !ok {
			unions[name] = buildHTTPUnionTypeData(dt, scope, closedObjects, name)
		}
		for _, nat := range dt.Values {
			collectHTTPUnionTypes(nat.Attribute, scope, unions, seen, closedObjects)
		}
	}
}

func buildHTTPUnionTypeData(u *expr.Union, scope *codegen.NameScope, closedObjects bool, names ...string) *uniongen.Type {
	att := &expr.AttributeExpr{Type: u}
	name := scope.GoTypeName(att)
	if len(names) > 0 {
		name = names[0]
	}
	kindName := scope.Unique(name + "Kind")

	fields := make([]*uniongen.Field, len(u.Values))
	hasScalarFormBranch := false
	for i, nat := range u.Values {
		fieldName := codegen.Goify(nat.Name, true)
		fieldType := scope.GoTypeRef(nat.Attribute)
		kindConst := kindName + fieldName
		fields[i] = &uniongen.Field{
			Name:                      expr.AttributeName(nat.Name),
			KindConst:                 kindConst,
			FieldName:                 fieldName,
			FieldType:                 fieldType,
			TypeTag:                   expr.UnionVariantTag(nat),
			FlatFormObject:            expr.IsObject(nat.Attribute.Type),
			FlatFormObjectAllowsEmpty: flatFormObjectAllowsEmpty(nat.Attribute),
			EmptyValueExpr:            emptyObjectValueExpr(fieldType),
			EmitPrimitiveAlias:        false,
		}
		if u.Untagged {
			fields[i].ValidateRef = unionBranchValidateRef(fieldType)
		}
		hasScalarFormBranch = hasScalarFormBranch || !fields[i].FlatFormObject
	}

	var jsonUnion *unionjson.Union
	if u.Untagged {
		branches := make([]*unionjson.Branch, len(fields))
		for i, field := range fields {
			branches[i] = &unionjson.Branch{Type: field.FieldType, Field: field.FieldName, Kind: field.KindConst, Validate: "err = " + field.ValidateRef}
		}
		jsonUnion = unionjson.Analyze(name, u, branches, expr.ElementName, closedObjects)
	}

	return &uniongen.Type{
		Name:                name,
		KindName:            kindName,
		Fields:              fields,
		TypeKey:             u.GetTypeKey(),
		ValueKey:            u.GetValueKey(),
		Untagged:            u.Untagged,
		JSON:                jsonUnion,
		HasScalarFormBranch: hasScalarFormBranch,
	}
}

func unionBranchValidateRef(fieldType string) string {
	typeName := strings.TrimPrefix(fieldType, "*")
	return "Validate" + typeName + "(v)"
}

func sortedNamedAttributes(attrs []*expr.NamedAttributeExpr) []*expr.NamedAttributeExpr {
	if len(attrs) < 2 {
		return attrs
	}
	sorted := slices.Clone(attrs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

func flatFormObjectAllowsEmpty(att *expr.AttributeExpr) bool {
	return expr.IsObject(att.Type) && len(att.AllRequired()) == 0
}

func emptyObjectValueExpr(fieldType string) string {
	if strings.HasPrefix(fieldType, "*") {
		return "&" + strings.TrimPrefix(fieldType, "*") + "{}"
	}
	return fieldType + "{}"
}
