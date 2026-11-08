package codegen

import (
	"slices"
	"sort"

	"github.com/CaliLuke/loom/http/codegen/internal/representation"

	"github.com/CaliLuke/loom/codegen/unionnames"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/uniongen"
	"github.com/CaliLuke/loom/codegen/unionjson"
	"github.com/CaliLuke/loom/expr"
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
	return makeHTTPTypeRecursive(att, make(map[expr.UserType]struct{}))
}

// makeHTTPMappedType keeps mapping defaults and validation while taking Go
// representation metadata from the service field it encodes or decodes.
func makeHTTPMappedType(mapped, service *expr.AttributeExpr) *expr.AttributeExpr {
	attr := *mapped
	attr.Meta = service.Meta
	return makeHTTPType(&attr)
}

func makeHTTPTypeRecursive(att *expr.AttributeExpr, seen map[expr.UserType]struct{}) *expr.AttributeExpr {
	constraints := selectedTransportConstraints(att)
	defaultValue, hasDefault := constraints.Default()
	switch dt := att.Type.(type) {
	case expr.UserType:
		_, result := dt.(*expr.ResultTypeExpr)
		alias := !result && !expr.IsObject(dt) && !expr.IsUnion(dt)
		if alias {
			att.Meta = expr.HTTPMediaMetadata(att)
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
		if _, ok := seen[dt]; ok {
			return att
		}
		seen[dt] = struct{}{}
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
		if att.AdditionalAttributes != nil {
			att.AdditionalAttributes = makeHTTPTypeRecursive(att.AdditionalAttributes, seen)
		}
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
		collectUnionBranchUserTypesSeen(att.AdditionalAttributes, hashes, seen)
	case *expr.Array:
		collectUnionBranchUserTypesSeen(actual.ElemType, hashes, seen)
	case *expr.Map:
		collectUnionBranchUserTypesSeen(actual.KeyType, hashes, seen)
		collectUnionBranchUserTypesSeen(actual.ElemType, hashes, seen)
	case *expr.Union:
		for _, nat := range actual.Values {
			representation.WalkUserTypes(nat.Attribute, func(ut expr.UserType) {
				hashes[ut.Hash()] = struct{}{}
			})
			collectUnionBranchUserTypesSeen(nat.Attribute, hashes, seen)
		}
	}
}

func collectHTTPUnionTypes(att *expr.AttributeExpr, scope *codegen.NameScope, context *codegen.AttributeContext, unions map[string]*uniongen.Type, seen map[string]struct{}, closedObjects bool, allocated map[string]*uniongen.Type) {
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
			name := scope.GoValueTypeName(&expr.AttributeExpr{Type: dt})
			if _, ok := unions[name]; !ok {
				if allocated[name] == nil {
					allocated[name] = buildHTTPUnionTypeData(dt, scope, context, closedObjects)
				}
				unions[name] = bindHTTPUnionDeclaration(allocated[name], union, context)
			}
			for _, nat := range union.Values {
				collectHTTPUnionTypes(nat.Attribute, scope, context, unions, seen, closedObjects, allocated)
			}
			return
		}
		collectHTTPUnionTypes(dt.Attribute(), scope, context, unions, seen, closedObjects, allocated)
	case *expr.Object:
		for _, nat := range sortedNamedAttributes(*dt) {
			collectHTTPUnionTypes(nat.Attribute, scope, context, unions, seen, closedObjects, allocated)
		}
		collectHTTPUnionTypes(att.AdditionalAttributes, scope, context, unions, seen, closedObjects, allocated)
	case *expr.Array:
		collectHTTPUnionTypes(dt.ElemType, scope, context, unions, seen, closedObjects, allocated)
	case *expr.Map:
		collectHTTPUnionTypes(dt.KeyType, scope, context, unions, seen, closedObjects, allocated)
		collectHTTPUnionTypes(dt.ElemType, scope, context, unions, seen, closedObjects, allocated)
	case *expr.Union:
		name := scope.GoTypeName(&expr.AttributeExpr{Type: dt})
		if _, ok := unions[name]; !ok {
			if allocated[name] == nil {
				allocated[name] = buildHTTPUnionTypeData(dt, scope, context, closedObjects)
			}
			unions[name] = bindHTTPUnionDeclaration(allocated[name], dt, context)
		}
		for _, nat := range dt.Values {
			collectHTTPUnionTypes(nat.Attribute, scope, context, unions, seen, closedObjects, allocated)
		}
	}
}

func buildHTTPUnionTypeData(typ expr.DataType, scope *codegen.NameScope, context *codegen.AttributeContext, closedObjects bool) *uniongen.Type {
	u := expr.AsUnion(typ)
	name := scope.HashedUniqueFamily(typ, codegen.Goify(typ.Name(), true), func(name string) []string {
		return unionnames.PackageDeclarations(u, name).Names
	})
	symbols := unionnames.PackageDeclarations(u, name)
	kindName := symbols.KindName

	members := unionnames.Resolve(u)
	fields := make([]*uniongen.Field, len(u.Values))
	for i, nat := range u.Values {
		member := members[nat.Name]
		fieldName := member.Field
		fieldType := scope.GoTypeRef(nat.Attribute)
		kindConst := symbols.Branches[nat.Name].KindConst
		fields[i] = &uniongen.Field{
			Name:               expr.AttributeName(nat.Name),
			KindConst:          kindConst,
			FieldName:          fieldName,
			MethodName:         member.Method,
			FieldType:          fieldType,
			AllowsNull:         expr.AllowsNull(nat.Attribute),
			TypeTag:            expr.UnionVariantTag(nat),
			EmitPrimitiveAlias: false,
		}
	}

	var jsonUnion *unionjson.Union
	if u.Untagged {
		branches := make([]*unionjson.Branch, len(fields))
		for i, field := range fields {
			branches[i] = &unionjson.Branch{Type: field.FieldType, Field: field.FieldName, Kind: field.KindConst}
		}
		jsonUnion = unionjson.Analyze(name, u, branches, expr.ElementName, closedObjects)
	}

	return bindHTTPUnionDeclaration(&uniongen.Type{
		Name:     name,
		KindName: kindName,
		Fields:   fields,
		TypeKey:  u.GetTypeKey(),
		ValueKey: u.GetValueKey(),
		Untagged: u.Untagged,
		Closed:   u.Closed(),
		JSON:     jsonUnion,
	}, u, context)
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
