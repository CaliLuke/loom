package service

import (
	"slices"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/uniongen"
	"github.com/CaliLuke/loom/codegen/unionjson"
	"github.com/CaliLuke/loom/codegen/unionnames"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
)

func collectServiceUnions(service *expr.ServiceExpr, types, errTypes []*UserTypeData, scope *codegen.NameScope) []*UnionTypeData {
	unionsByIdentity := make(map[destinationTypeIdentity]*UnionTypeData)
	defaultPath := naming.ServiceDir(service.Name)
	seen := make(map[destinationTypeIdentity]struct{})
	collect := func(att *expr.AttributeExpr, loc *codegen.Location) {
		collectUnionTypes(att, scope, loc, defaultPath, unionsByIdentity, seen)
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
	keys := make([]destinationTypeIdentity, 0, len(unionsByIdentity))
	for key := range unionsByIdentity {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left, right := keys[i], keys[j]
		if unionsByIdentity[left].Name != unionsByIdentity[right].Name {
			return unionsByIdentity[left].Name < unionsByIdentity[right].Name
		}
		if left.path != right.path {
			return left.path < right.path
		}
		return left.hash < right.hash
	})
	unions := make([]*UnionTypeData, 0, len(keys))
	for _, key := range keys {
		unions = append(unions, unionsByIdentity[key])
	}
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
		typeReference := at
		typeLoc := codegen.UserTypeLocation(dt)
		if typeLoc == nil {
			typeLoc = loc
		}
		seen[dt.ID()] = struct{}{}
		typeScope := scope.WithTypeNameLocation(typeLoc)
		if expr.AsUnion(dt.Attribute().Type) == nil {
			data = append(data, &UserTypeData{
				Name:        dt.Name(),
				VarName:     typeScope.GoValueTypeName(typeReference),
				Description: dt.Attribute().Description,
				Def:         typeScope.GoValueTypeDef(dt.Attribute(), false, true),
				Ref:         typeScope.GoValueTypeRef(typeReference),
				Loc:         typeLoc,
				Type:        dt,
			})
		}
		data = append(data, collect(dt.Attribute(), typeLoc)...)
	case *expr.Object:
		for _, nat := range *dt {
			data = append(data, collect(nat.Attribute, loc)...)
		}
		data = append(data, collect(at.AdditionalAttributes, loc)...)
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
// definitions referenced by the service. Allocation and emission share physical
// package plus type-hash identity: one raw union can be declared in more than one
// destination. defaultPath canonicalizes an implicit service location without
// changing the declaration's original location or import qualification.
func collectUnionTypes(att *expr.AttributeExpr, scope *codegen.NameScope, loc *codegen.Location, defaultPath string, unions map[destinationTypeIdentity]*UnionTypeData, seen map[destinationTypeIdentity]struct{}) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	if explicit := codegen.UserTypeLocation(att.Type); explicit != nil {
		loc = explicit
	}
	path := defaultPath
	if loc != nil {
		path = loc.RelImportPath
	}
	switch dt := att.Type.(type) {
	case expr.UserType:
		key := destinationTypeIdentity{path: path, hash: dt.Hash()}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		if union := expr.AsUnion(dt.Attribute().Type); union != nil {
			if _, ok := unions[key]; !ok {
				unions[key] = buildUnionTypeData(dt, scope, loc)
			}
			for _, nat := range union.Values {
				collectUnionTypes(nat.Attribute, scope, loc, defaultPath, unions, seen)
			}
			return
		}
		collectUnionTypes(dt.Attribute(), scope, loc, defaultPath, unions, seen)
	case *expr.Object:
		for _, nat := range sortedNamedAttributes(*dt) {
			collectUnionTypes(nat.Attribute, scope, loc, defaultPath, unions, seen)
		}
		collectUnionTypes(att.AdditionalAttributes, scope, loc, defaultPath, unions, seen)
	case *expr.Array:
		collectUnionTypes(dt.ElemType, scope, loc, defaultPath, unions, seen)
	case *expr.Map:
		collectUnionTypes(dt.KeyType, scope, loc, defaultPath, unions, seen)
		collectUnionTypes(dt.ElemType, scope, loc, defaultPath, unions, seen)
	case *expr.Union:
		key := destinationTypeIdentity{path: path, hash: dt.Hash()}
		if _, ok := unions[key]; !ok {
			unions[key] = buildUnionTypeData(dt, scope, loc)
		}
		for _, nat := range dt.Values {
			collectUnionTypes(nat.Attribute, scope, loc, defaultPath, unions, seen)
		}
	}
}

// buildUnionTypeData creates the data needed to generate a sum-type union
// struct, its discriminator kind, and branch metadata.
func buildUnionTypeData(typ expr.DataType, scope *codegen.NameScope, loc *codegen.Location) *UnionTypeData {
	scope = scope.WithTypeNameLocation(loc)
	u := expr.AsUnion(typ)
	name := scope.GoValueTypeName(&expr.AttributeExpr{Type: typ})
	declarations := nativeTypeDeclarations(typ, name, nil)
	kindName := declarations.union.KindName
	unionPkg := scope.PackageName(loc)

	members := unionnames.Resolve(u)
	fields := make([]*uniongen.Field, len(u.Values))
	hasScalarFormBranch := false
	for i, nat := range u.Values {
		member := members[nat.Name]
		fieldName := member.Field
		var pkg string
		if tloc := codegen.UserTypeLocation(nat.Attribute.Type); tloc != nil {
			pkg = scope.PackageName(tloc)
			if pkg == unionPkg {
				pkg = ""
			}
		}
		fieldType := scope.GoFullTypeRef(nat.Attribute, pkg)
		primitiveAliasType, hasPrimitiveAlias := primitiveAliasGoType(nat.Attribute.Type)
		_, isUserType := nat.Attribute.Type.(expr.UserType)
		// Bare primitive branches use their native Go type. Only an Any branch
		// gets a named alias of loom.JSONValue.
		emitPrimitiveAlias := hasPrimitiveAlias && primitiveAliasType == "loom.JSONValue" && !isUserType && pkg == ""
		if emitPrimitiveAlias {
			fieldType = declarations.aliases[nat.Name]
		}
		kindConst := declarations.union.Branches[nat.Name].KindConst
		fields[i] = &uniongen.Field{
			Name:                      expr.AttributeName(nat.Name),
			KindConst:                 kindConst,
			FieldName:                 fieldName,
			MethodName:                member.Method,
			FieldType:                 fieldType,
			AllowsNull:                expr.AllowsNull(nat.Attribute),
			FlatFormObject:            expr.IsObject(nat.Attribute.Type),
			FlatFormObjectAllowsEmpty: flatFormObjectAllowsEmpty(nat.Attribute),
			EmptyValueExpr:            emptyObjectValueExpr(fieldType),
			EmitPrimitiveAlias:        emitPrimitiveAlias,
			PrimitiveAliasType:        primitiveAliasType,
			TypeTag:                   expr.UnionVariantTag(nat),
		}
		if u.Untagged {
			fields[i].ValidateCode = unionBranchValidationCode(nat.Attribute, typeContext(scope), false)
		}
		hasScalarFormBranch = hasScalarFormBranch || !fields[i].FlatFormObject
	}

	return &UnionTypeData{
		Type: uniongen.Type{
			Name:                name,
			KindName:            kindName,
			Fields:              fields,
			Loc:                 loc,
			TypeKey:             u.GetTypeKey(),
			ValueKey:            u.GetValueKey(),
			Untagged:            u.Untagged,
			Closed:              u.Closed(),
			JSON:                analyzeUnionJSON(name, u, fields),
			HasScalarFormBranch: hasScalarFormBranch,
		},
		validations: buildUntaggedUnionValidations(u, scope, loc),
	}
}

// buildViewUnionTypeData creates the data needed to generate a sum-type union
// in the views package. Field types are computed using the view scope and are
// always emitted unqualified so they refer to the view-local projected types.
// The declaration uses the actual nominal type identity, as do its references.
func buildViewUnionTypeData(typ expr.DataType, scope *codegen.NameScope, loc *codegen.Location) *UnionTypeData {
	u := expr.AsUnion(typ)
	name := scope.GoValueTypeName(&expr.AttributeExpr{Type: typ})
	declarations := nativeTypeDeclarations(typ, name, viewValidatorNames(typ, name))
	kindName := declarations.union.KindName

	members := unionnames.Resolve(u)
	fields := make([]*uniongen.Field, len(u.Values))
	hasScalarFormBranch := false
	for i, nat := range u.Values {
		member := members[nat.Name]
		fieldName := member.Field
		fieldType := scope.GoTypeRef(nat.Attribute)
		primitiveAliasType, hasPrimitiveAlias := primitiveAliasGoType(nat.Attribute.Type)
		_, isUserType := nat.Attribute.Type.(expr.UserType)
		emitPrimitiveAlias := hasPrimitiveAlias && primitiveAliasType == "loom.JSONValue" && !isUserType
		if emitPrimitiveAlias {
			fieldType = declarations.aliases[nat.Name]
		}
		kindConst := declarations.union.Branches[nat.Name].KindConst
		fields[i] = &uniongen.Field{
			Name:                      expr.AttributeName(nat.Name),
			KindConst:                 kindConst,
			FieldName:                 fieldName,
			MethodName:                member.Method,
			FieldType:                 fieldType,
			AllowsNull:                expr.AllowsNull(nat.Attribute),
			FlatFormObject:            expr.IsObject(nat.Attribute.Type),
			FlatFormObjectAllowsEmpty: flatFormObjectAllowsEmpty(nat.Attribute),
			EmptyValueExpr:            emptyObjectValueExpr(fieldType),
			EmitPrimitiveAlias:        emitPrimitiveAlias,
			PrimitiveAliasType:        primitiveAliasType,
			TypeTag:                   expr.UnionVariantTag(nat),
		}
		if u.Untagged {
			fields[i].ValidateCode = unionBranchValidationCode(nat.Attribute, projectedTypeContext("", !expr.IsPrimitive(nat.Attribute.Type), scope), true)
		}
		hasScalarFormBranch = hasScalarFormBranch || !fields[i].FlatFormObject
	}

	return &UnionTypeData{
		Type: uniongen.Type{
			Name:                name,
			KindName:            kindName,
			Fields:              fields,
			Loc:                 loc,
			TypeKey:             u.GetTypeKey(),
			ValueKey:            u.GetValueKey(),
			Untagged:            u.Untagged,
			Closed:              u.Closed(),
			JSON:                analyzeUnionJSON(name, u, fields),
			HasScalarFormBranch: hasScalarFormBranch,
		},
	}
}

func unionBranchValidationCode(att *expr.AttributeExpr, ctx *codegen.AttributeContext, view bool) string {
	if ut, named := att.Type.(expr.UserType); named && !expr.IsNullable(att) {
		return codegen.ValidationCode(ut.Attribute(), ut, ctx, true, expr.IsAlias(ut), view, "v")
	}
	return codegen.ValidationCode(att, nil, ctx, true, expr.IsAlias(att.Type), view, "v")
}

func buildUntaggedUnionValidations(union *expr.Union, scope *codegen.NameScope, loc *codegen.Location) []*unionValidationData {
	if !union.Untagged {
		return nil
	}
	types := collectTypes(&expr.AttributeExpr{Type: union}, scope, make(map[string]struct{}), loc)
	validations := make([]*unionValidationData, 0, len(types))
	for _, data := range types {
		if !expr.IsObject(data.Type) && !expr.IsArray(data.Type) && !expr.IsMap(data.Type) {
			continue
		}
		ctx := typeContext(scope)
		function := codegen.NamedValidationFunction(&expr.AttributeExpr{Type: data.Type}, ctx, "result")
		validations = append(validations, &unionValidationData{
			data: &ValidateData{
				Name:        function.Name,
				Ref:         function.Ref,
				Description: "runs the validations defined on " + data.VarName + ".",
				Validate:    codegen.ValidationCode(data.Type.Attribute(), data.Type, function.Context(ctx), true, false, false, "result"),
			},
			loc: data.Loc,
		})
	}
	return validations
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
			if dt.Kind() == expr.AnyKind {
				return "loom.JSONValue", true
			}
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
		pkg = scope.PackageName(codegen.UserTypeLocation(ut))
	}
	attribute := er.AttributeExpr
	if expr.IsNullable(attribute) {
		attribute = nullableErrorAttribute(scope, er.Name, attribute)
		pkg = ""
	}
	return &ErrorInitData{
		Name:        errorConstructorName(er, scope),
		Description: er.Description,
		ErrName:     er.Name,
		Nullable:    expr.IsNullable(er.AttributeExpr),
		TypeName:    scope.GoTypeName(attribute),
		TypeRef:     scope.GoFullTypeRef(attribute, pkg),
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
		return att.AdditionalAttributes != nil && hasResultType(att.AdditionalAttributes, seen)
	case *expr.Union:
		for _, nat := range a.Values {
			if hasResultType(nat.Attribute, seen) {
				return true
			}
		}
	}
	return false
}

func analyzeUnionJSON(name string, u *expr.Union, fields []*uniongen.Field) *unionjson.Union {
	if !u.Untagged {
		return nil
	}
	branches := make([]*unionjson.Branch, len(fields))
	for i, field := range fields {
		branches[i] = &unionjson.Branch{Type: field.FieldType, Field: field.FieldName, Kind: field.KindConst, Validate: field.ValidateCode}
	}
	return unionjson.Analyze(name, u, branches, expr.AttributeName, false)
}
