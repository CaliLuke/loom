package codegen

import (
	"slices"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
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
	return makeHTTPTypeRecursive(att, make(map[string]struct{}))
}

func makeHTTPTypeRecursive(att *expr.AttributeExpr, seen map[string]struct{}) *expr.AttributeExpr {
	switch dt := att.Type.(type) {
	case expr.UserType:
		if _, ok := dt.(*expr.ResultTypeExpr); !ok && !expr.IsObject(dt) && !expr.IsUnion(dt) {
			att.Type = dt.Attribute().Type
			if v := dt.Attribute().Validation; v != nil {
				if att.Validation == nil {
					att.Validation = v
				} else {
					att.Validation.Merge(v)
				}
			}
			att.DefaultValue = dt.Attribute().DefaultValue
			att.UserExamples = dt.Attribute().UserExamples
		}
		if _, ok := seen[dt.ID()]; ok {
			return att
		}
		seen[dt.ID()] = struct{}{}
		dt.SetAttribute(makeHTTPTypeRecursive(dt.Attribute(), seen))
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

// collectUserTypes traverses the given data type recursively and calls back the
// given function for each attribute using a user type.
func collectUserTypes(dt expr.DataType, cb func(expr.UserType), seen ...map[string]struct{}) {
	if dt == expr.Empty {
		return
	}
	var s map[string]struct{}
	if len(seen) > 0 {
		s = seen[0]
	} else {
		s = make(map[string]struct{})
	}
	switch actual := dt.(type) {
	case *expr.Object:
		for _, nat := range *actual {
			collectUserTypes(nat.Attribute.Type, cb, s)
		}
	case *expr.Union:
		for _, nat := range actual.Values {
			collectUserTypes(nat.Attribute.Type, cb, s)
		}
	case *expr.Array:
		collectUserTypes(actual.ElemType.Type, cb, s)
	case *expr.Map:
		collectUserTypes(actual.KeyType.Type, cb, s)
		collectUserTypes(actual.ElemType.Type, cb, s)
	case expr.UserType:
		if _, ok := s[actual.ID()]; ok {
			return
		}
		s[actual.ID()] = struct{}{}
		cb(actual)
		collectUserTypes(actual.Attribute().Type, cb, s)
	}
}

func collectUnionBranchUserTypes(att *expr.AttributeExpr, ids map[string]struct{}) {
	collectUnionBranchUserTypesSeen(att, ids, make(map[string]struct{}))
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

func collectUnionBranchUserTypesSeen(att *expr.AttributeExpr, ids, seen map[string]struct{}) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	switch actual := att.Type.(type) {
	case expr.UserType:
		if _, ok := seen[actual.ID()]; ok {
			return
		}
		seen[actual.ID()] = struct{}{}
		collectUnionBranchUserTypesSeen(actual.Attribute(), ids, seen)
	case *expr.Object:
		for _, nat := range *actual {
			collectUnionBranchUserTypesSeen(nat.Attribute, ids, seen)
		}
	case *expr.Array:
		collectUnionBranchUserTypesSeen(actual.ElemType, ids, seen)
	case *expr.Map:
		collectUnionBranchUserTypesSeen(actual.KeyType, ids, seen)
		collectUnionBranchUserTypesSeen(actual.ElemType, ids, seen)
	case *expr.Union:
		for _, nat := range actual.Values {
			collectUserTypes(nat.Attribute.Type, func(ut expr.UserType) {
				ids[ut.ID()] = struct{}{}
			})
			collectUnionBranchUserTypesSeen(nat.Attribute, ids, seen)
		}
	}
}

func (sds *ServicesData) collectEndpointUnionTypes(httpSvc *expr.HTTPServiceExpr, scope *codegen.NameScope) []*service.UnionTypeData {
	unionByHash := make(map[string]*service.UnionTypeData)
	seenUnionTypes := make(map[string]struct{})
	for _, endpoint := range httpSvc.HTTPEndpoints {
		collectHTTPUnionTypes(endpoint.Body, scope, unionByHash, seenUnionTypes)
		if endpoint.MethodExpr.StreamingPayload.Type != expr.Empty {
			collectHTTPUnionTypes(endpoint.StreamingBody, scope, unionByHash, seenUnionTypes)
		}
		if endpoint.MethodExpr.Result != nil {
			svcData := sds.ServicesData.Get(httpSvc.ServiceExpr.Name)
			md := svcData.Method(endpoint.MethodExpr.Name)
			for _, response := range endpoint.Responses {
				body := effectiveClientResponseBody(response.Body, endpoint.MethodExpr.Result, md)
				collectHTTPUnionTypes(body, scope, unionByHash, seenUnionTypes)
			}
		}
		for _, httpError := range endpoint.HTTPErrors {
			collectHTTPUnionTypes(httpError.Response.Body, scope, unionByHash, seenUnionTypes)
		}
	}
	unions := make([]*service.UnionTypeData, 0, len(unionByHash))
	for _, union := range unionByHash {
		unions = append(unions, union)
	}
	sort.Slice(unions, func(i, j int) bool {
		return unions[i].Name < unions[j].Name
	})
	return unions
}

func collectHTTPUnionTypes(att *expr.AttributeExpr, scope *codegen.NameScope, unions map[string]*service.UnionTypeData, seen map[string]struct{}) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	switch dt := att.Type.(type) {
	case expr.UserType:
		if _, ok := seen[dt.ID()]; ok {
			return
		}
		seen[dt.ID()] = struct{}{}
		if union := expr.AsUnion(dt.Attribute().Type); union != nil {
			hash := union.Hash()
			if _, ok := unions[hash]; !ok {
				name := scope.GoTypeName(&expr.AttributeExpr{Type: dt})
				unions[hash] = buildHTTPUnionTypeData(union, scope, name)
			}
			for _, nat := range union.Values {
				collectHTTPUnionTypes(nat.Attribute, scope, unions, seen)
			}
			return
		}
		collectHTTPUnionTypes(dt.Attribute(), scope, unions, seen)
	case *expr.Object:
		for _, nat := range sortedNamedAttributes(*dt) {
			collectHTTPUnionTypes(nat.Attribute, scope, unions, seen)
		}
	case *expr.Array:
		collectHTTPUnionTypes(dt.ElemType, scope, unions, seen)
	case *expr.Map:
		collectHTTPUnionTypes(dt.KeyType, scope, unions, seen)
		collectHTTPUnionTypes(dt.ElemType, scope, unions, seen)
	case *expr.Union:
		hash := dt.Hash()
		if _, ok := unions[hash]; !ok {
			unions[hash] = buildHTTPUnionTypeData(dt, scope)
		}
		for _, nat := range dt.Values {
			collectHTTPUnionTypes(nat.Attribute, scope, unions, seen)
		}
	}
}

func buildHTTPUnionTypeData(u *expr.Union, scope *codegen.NameScope, names ...string) *service.UnionTypeData {
	att := &expr.AttributeExpr{Type: u}
	name := scope.GoTypeName(att)
	if len(names) > 0 {
		name = names[0]
	}
	kindName := scope.Unique(name + "Kind")

	fields := make([]*service.UnionFieldData, len(u.Values))
	hasScalarFormBranch := false
	for i, nat := range u.Values {
		fieldName := codegen.Goify(nat.Name, true)
		fieldType := scope.GoTypeRef(nat.Attribute)
		kindConst := kindName + fieldName
		fields[i] = &service.UnionFieldData{
			Name:                      nat.Name,
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
			fields[i].RequiredFields, fields[i].NonNullableFields, fields[i].JSONFields, fields[i].RejectUnknownJSONFields = serviceUnionBranchJSONFields(nat.Attribute)
		}
		hasScalarFormBranch = hasScalarFormBranch || !fields[i].FlatFormObject
	}

	return &service.UnionTypeData{
		Name:                name,
		KindName:            kindName,
		Fields:              fields,
		TypeKey:             u.GetTypeKey(),
		ValueKey:            u.GetValueKey(),
		Untagged:            u.Untagged,
		HasScalarFormBranch: hasScalarFormBranch,
	}
}

func unionBranchValidateRef(fieldType string) string {
	typeName := strings.TrimPrefix(fieldType, "*")
	return "Validate" + typeName + "(v)"
}

func serviceUnionBranchJSONFields(att *expr.AttributeExpr) ([]string, []string, []string, bool) {
	ut := att.Type.(expr.UserType)
	parent := ut.Attribute()
	object := expr.AsObject(ut.Attribute().Type)
	required := make([]string, 0, len(parent.AllRequired()))
	nonNullable := make([]string, 0, len(*object))
	fields := make([]string, 0, len(*object))
	for _, field := range *object {
		name := codegen.JSONFieldName(field.Name, field.Attribute)
		fields = append(fields, name)
		if parent.IsRequired(field.Name) {
			required = append(required, name)
		}
		if !expr.AllowsNull(field.Attribute) {
			nonNullable = append(nonNullable, name)
		}
	}
	sort.Strings(required)
	sort.Strings(nonNullable)
	sort.Strings(fields)
	closed, _ := parent.Meta.Last("openapi:additionalProperties")
	return required, nonNullable, fields, closed == "false"
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
