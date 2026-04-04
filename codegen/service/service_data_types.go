package service

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func collectServiceUnions(service *expr.ServiceExpr, types, errTypes []*UserTypeData, scope *codegen.NameScope) []*UnionTypeData {
	unionByHash := make(map[string]*UnionTypeData)
	seen := make(map[string]struct{})
	collect := func(att *expr.AttributeExpr, loc *codegen.Location) {
		collectUnionTypes(att, scope, loc, unionByHash, seen)
	}
	for _, t := range types {
		collect(&expr.AttributeExpr{Type: t.Type}, t.Loc)
	}
	for _, t := range errTypes {
		collect(&expr.AttributeExpr{Type: t.Type}, t.Loc)
	}
	for _, method := range service.Methods {
		if method.Payload != nil {
			collect(method.Payload, codegen.UserTypeLocation(method.Payload.Type))
		}
		if method.StreamingPayload != nil {
			collect(method.StreamingPayload, codegen.UserTypeLocation(method.StreamingPayload.Type))
		}
		if method.Result != nil {
			collect(method.Result, codegen.UserTypeLocation(method.Result.Type))
		}
		for _, errExpr := range method.Errors {
			collect(errExpr.AttributeExpr, codegen.UserTypeLocation(errExpr.Type))
		}
	}
	unions := make([]*UnionTypeData, 0, len(unionByHash))
	for _, union := range unionByHash {
		unions = append(unions, union)
	}
	sort.Slice(unions, func(i, j int) bool {
		return unions[i].Name < unions[j].Name
	})
	return unions
}

// collectTypes recurses through the attribute to gather all user types and
// records them in userTypes.
func collectTypes(at *expr.AttributeExpr, scope *codegen.NameScope, seen map[string]struct{}, loc *codegen.Location) (data []*UserTypeData) {
	if at == nil || at.Type == expr.Empty {
		return data
	}
	collect := func(at *expr.AttributeExpr, loc *codegen.Location) []*UserTypeData {
		return collectTypes(at, scope, seen, loc)
	}
	switch dt := at.Type.(type) {
	case expr.UserType:
		if _, ok := seen[dt.ID()]; ok {
			return nil
		}
		typeLoc := codegen.UserTypeLocation(dt)
		if typeLoc == nil {
			typeLoc = loc
		}
		data = append(data, &UserTypeData{
			Name:        dt.Name(),
			VarName:     scope.GoTypeName(at),
			Description: dt.Attribute().Description,
			Def:         scope.GoTypeDef(dt.Attribute(), false, true),
			Ref:         scope.GoTypeRef(at),
			Loc:         typeLoc,
			Type:        dt,
		})
		seen[dt.ID()] = struct{}{}
		data = append(data, collect(dt.Attribute(), typeLoc)...)
	case *expr.Object:
		for _, nat := range *dt {
			data = append(data, collect(nat.Attribute, loc)...)
		}
	case *expr.Array:
		data = append(data, collect(dt.ElemType, loc)...)
	case *expr.Map:
		data = append(data, collect(dt.KeyType, loc)...)
		data = append(data, collect(dt.ElemType, loc)...)
	case *expr.Union:
		for _, nat := range dt.Values {
			data = append(data, collect(nat.Attribute, loc)...)
		}
	}
	return data
}

// collectUnionTypes traverses the attribute to gather all union sum-type
// definitions referenced by the service. It records each union by its hash to
// avoid generating duplicate types.
func collectUnionTypes(att *expr.AttributeExpr, scope *codegen.NameScope, loc *codegen.Location, unions map[string]*UnionTypeData, seen map[string]struct{}) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	switch dt := att.Type.(type) {
	case expr.UserType:
		if _, ok := seen[dt.ID()]; ok {
			return
		}
		seen[dt.ID()] = struct{}{}
		collectUnionTypes(dt.Attribute(), scope, codegen.UserTypeLocation(dt), unions, seen)
	case *expr.Object:
		for _, nat := range sortedNamedAttributes(*dt) {
			collectUnionTypes(nat.Attribute, scope, loc, unions, seen)
		}
	case *expr.Array:
		collectUnionTypes(dt.ElemType, scope, loc, unions, seen)
	case *expr.Map:
		collectUnionTypes(dt.KeyType, scope, loc, unions, seen)
		collectUnionTypes(dt.ElemType, scope, loc, unions, seen)
	case *expr.Union:
		hash := dt.Hash()
		if _, ok := unions[hash]; !ok {
			unions[hash] = buildUnionTypeData(dt, scope, loc)
		}
		for _, nat := range dt.Values {
			collectUnionTypes(nat.Attribute, scope, loc, unions, seen)
		}
	}
}

// buildUnionTypeData creates the data needed to generate a sum-type union
// struct, its discriminator kind, and branch metadata.
func buildUnionTypeData(u *expr.Union, scope *codegen.NameScope, loc *codegen.Location) *UnionTypeData {
	att := &expr.AttributeExpr{Type: u}
	name := scope.GoTypeName(att)
	kindName := scope.Unique(name + "Kind")
	unionPkg := loc.PackageName()

	fields := make([]*UnionFieldData, len(u.Values))
	hasScalarFormBranch := false
	for i, nat := range u.Values {
		fieldName := codegen.Goify(nat.Name, true)
		var pkg string
		if tloc := codegen.UserTypeLocation(nat.Attribute.Type); tloc != nil {
			pkg = tloc.PackageName()
			if pkg == unionPkg {
				pkg = ""
			}
		}
		fieldType := scope.GoFullTypeRef(nat.Attribute, pkg)
		primitiveAliasType, hasPrimitiveAlias := primitiveAliasGoType(nat.Attribute.Type)
		_, isUserType := nat.Attribute.Type.(expr.UserType)
		emitPrimitiveAlias := hasPrimitiveAlias && !isUserType && pkg == ""
		kindConst := kindName + codegen.Goify(nat.Name, true)
		fields[i] = &UnionFieldData{
			Name:                      nat.Name,
			KindConst:                 kindConst,
			FieldName:                 fieldName,
			FieldType:                 fieldType,
			FlatFormObject:            expr.IsObject(nat.Attribute.Type),
			FlatFormObjectAllowsEmpty: flatFormObjectAllowsEmpty(nat.Attribute),
			EmptyValueExpr:            emptyObjectValueExpr(fieldType),
			EmitPrimitiveAlias:        emitPrimitiveAlias,
			PrimitiveAliasType:        primitiveAliasType,
			TypeTag:                   expr.UnionVariantTag(nat),
		}
		hasScalarFormBranch = hasScalarFormBranch || !fields[i].FlatFormObject
	}

	return &UnionTypeData{
		Name:                name,
		KindName:            kindName,
		Fields:              fields,
		Loc:                 loc,
		TypeKey:             u.GetTypeKey(),
		ValueKey:            u.GetValueKey(),
		HasScalarFormBranch: hasScalarFormBranch,
	}
}

// buildViewUnionTypeData creates the data needed to generate a sum-type union
// in the views package. Field types are computed using the view scope and are
// always emitted unqualified so they refer to the view-local projected types.
func buildViewUnionTypeData(u *expr.Union, scope *codegen.NameScope, loc *codegen.Location) *UnionTypeData {
	att := &expr.AttributeExpr{Type: u}
	name := scope.GoTypeName(att)
	kindName := scope.Unique(name + "Kind")

	fields := make([]*UnionFieldData, len(u.Values))
	hasScalarFormBranch := false
	for i, nat := range u.Values {
		fieldName := codegen.Goify(nat.Name, true)
		fieldType := scope.GoTypeRef(nat.Attribute)
		primitiveAliasType, hasPrimitiveAlias := primitiveAliasGoType(nat.Attribute.Type)
		_, isUserType := nat.Attribute.Type.(expr.UserType)
		emitPrimitiveAlias := hasPrimitiveAlias && !isUserType
		kindConst := kindName + codegen.Goify(nat.Name, true)
		fields[i] = &UnionFieldData{
			Name:                      nat.Name,
			KindConst:                 kindConst,
			FieldName:                 fieldName,
			FieldType:                 fieldType,
			FlatFormObject:            expr.IsObject(nat.Attribute.Type),
			FlatFormObjectAllowsEmpty: flatFormObjectAllowsEmpty(nat.Attribute),
			EmptyValueExpr:            emptyObjectValueExpr(fieldType),
			EmitPrimitiveAlias:        emitPrimitiveAlias,
			PrimitiveAliasType:        primitiveAliasType,
			TypeTag:                   expr.UnionVariantTag(nat),
		}
		hasScalarFormBranch = hasScalarFormBranch || !fields[i].FlatFormObject
	}

	return &UnionTypeData{
		Name:                name,
		KindName:            kindName,
		Fields:              fields,
		Loc:                 loc,
		TypeKey:             u.GetTypeKey(),
		ValueKey:            u.GetValueKey(),
		HasScalarFormBranch: hasScalarFormBranch,
	}
}

// sortedNamedAttributes returns object fields sorted by attribute name.
// Union naming uses NameScope uniqueness, so callers that discover unions while
// traversing objects must use a deterministic field order to avoid oscillating
// generated identifiers across runs.
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

// primitiveAliasGoType resolves the native Go type for a primitive alias branch.
// It uses expr.IsPrimitive to enforce the type contract and then unwraps aliases.
func primitiveAliasGoType(dt expr.DataType) (string, bool) {
	if !expr.IsPrimitive(dt) {
		return "", false
	}
	for {
		ut, ok := dt.(expr.UserType)
		if !ok {
			return codegen.GoNativeTypeName(dt), true
		}
		dt = ut.Attribute().Type
	}
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

// buildErrorInitData creates the data needed to generate code around endpoint error return values.
func buildErrorInitData(er *expr.ErrorExpr, scope *codegen.NameScope) *ErrorInitData {
	_, temporary := er.Meta["loom:error:temporary"]
	_, timeout := er.Meta["loom:error:timeout"]
	_, fault := er.Meta["loom:error:fault"]
	var pkg string
	if ut, ok := er.Type.(expr.UserType); ok {
		pkg = codegen.UserTypeLocation(ut).PackageName()
	}
	return &ErrorInitData{
		Name:        fmt.Sprintf("Make%s", codegen.Goify(er.Name, true)),
		Description: er.Description,
		ErrName:     er.Name,
		TypeName:    scope.GoTypeName(er.AttributeExpr),
		TypeRef:     scope.GoFullTypeRef(er.AttributeExpr, pkg),
		Temporary:   temporary,
		Timeout:     timeout,
		Fault:       fault,
		RemedyCode:  errorRemedyCode(er),
		SafeMessage: errorSafeMessage(er),
		RetryHint:   errorRetryHint(er),
	}
}

func errorRemedyCode(er *expr.ErrorExpr) string {
	if er.Remedy == nil {
		return ""
	}
	return er.Remedy.Code
}

func errorSafeMessage(er *expr.ErrorExpr) string {
	if er.Remedy == nil {
		return ""
	}
	return er.Remedy.SafeMessage
}

func errorRetryHint(er *expr.ErrorExpr) string {
	if er.Remedy == nil {
		return ""
	}
	return er.Remedy.RetryHint
}

// hasResultType returns true if the given attribute has a result type recursively.
func hasResultType(att *expr.AttributeExpr, seens ...map[string]struct{}) bool {
	if _, ok := att.Type.(*expr.ResultTypeExpr); ok {
		return true
	}
	var seen map[string]struct{}
	if len(seens) > 0 {
		seen = seens[0]
	} else {
		seen = make(map[string]struct{})
	}
	switch a := att.Type.(type) {
	case expr.UserType:
		if _, ok := seen[a.ID()]; ok {
			return false
		}
		seen[a.ID()] = struct{}{}
		return hasResultType(a.Attribute(), seen)
	case *expr.Array:
		return hasResultType(a.ElemType, seen)
	case *expr.Map:
		return hasResultType(a.KeyType, seen) || hasResultType(a.ElemType, seen)
	case *expr.Object:
		for _, nat := range *a {
			if hasResultType(nat.Attribute, seen) {
				return true
			}
		}
	case *expr.Union:
		for _, nat := range a.Values {
			if hasResultType(nat.Attribute, seen) {
				return true
			}
		}
	}
	return false
}
